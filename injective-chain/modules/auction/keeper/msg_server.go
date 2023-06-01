package keeper

import (
	"context"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "auction_h",
		},
	}
}

func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	k.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) Bid(goCtx context.Context, msg *types.MsgBid) (*types.MsgBidResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// prepare context
	ctx := sdk.UnwrapSDKContext(goCtx)

	round := k.GetAuctionRound(ctx)
	if msg.Round != round {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrBidRound, "current round is %d but got bid for %d", round, msg.Round)
	}
	// check valid bid
	lastBid := k.GetHighestBid(ctx)
	if msg.BidAmount.Amount.LT(lastBid.Amount.Amount) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "Bid must exceed current highest bid")
	}

	// ensure last_bid * (1+min_next_increment_rate) <= new_bid
	params := k.GetParams(ctx)
	if lastBid.Amount.Amount.ToDec().Mul(sdk.OneDec().Add(params.MinNextBidIncrementRate)).GT(msg.BidAmount.Amount.ToDec()) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "new bid should be bigger than last bid + min increment percentage")
	}

	// process bid
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "invalid sender address")
	}

	// deposit new bid
	newBidAmount := sdk.NewCoins(msg.BidAmount)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, newBidAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("Bidder deposit failed", "senderAddr", senderAddr.String(), "coin", msg.BidAmount.String())
		return nil, errors.Wrap(err, "deposit failed")
	}

	// check first bidder
	isFirstBidder := !lastBid.Amount.Amount.IsPositive()
	if !isFirstBidder {
		err := k.refundLastBidder(ctx)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	}

	// set new bid to store
	k.SetBid(ctx, msg.Sender, msg.BidAmount)

	// emit typed event for bid
	auctionRound := k.GetAuctionRound(ctx)
	_ = ctx.EventManager().EmitTypedEvent(&types.EventBid{
		Bidder: msg.Sender,
		Amount: msg.BidAmount,
		Round:  auctionRound,
	})
	return &types.MsgBidResponse{}, nil
}

func (k msgServer) refundLastBidder(ctx sdk.Context) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	lastBid := k.GetHighestBid(ctx)
	lastBidAmount := lastBid.Amount.Amount
	lastBidder, err := sdk.AccAddressFromBech32(lastBid.Bidder)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error(err.Error())
		return err
	}

	bidAmount := sdk.NewCoins(sdk.NewCoin(chaintypes.InjectiveCoin, lastBidAmount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, lastBidder, bidAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("Bidder refund failed", "lastBidderAddr", lastBidder.String(), "coin", bidAmount.String())
		return errors.Wrap(err, "deposit failed")
	}

	return nil
}
