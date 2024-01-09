package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type BinaryOptionsMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewBinaryOptionsMsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper for binary options market functions.
func NewBinaryOptionsMsgServerImpl(keeper Keeper) BinaryOptionsMsgServer {
	return BinaryOptionsMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "bin_msg_h",
		},
	}
}

func (k BinaryOptionsMsgServer) InstantBinaryOptionsMarketLaunch(goCtx context.Context, msg *types.MsgInstantBinaryOptionsMarketLaunch) (*types.MsgInstantBinaryOptionsMarketLaunchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	fee := k.GetParams(ctx).BinaryOptionsMarketInstantListingFee
	err := k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching binary options market", err)
		return nil, err
	}

	// check if the market launch proposal already exists
	marketID := types.NewBinaryOptionsMarketID(msg.Ticker, msg.QuoteDenom, msg.OracleSymbol, msg.OracleProvider, msg.OracleType)
	if k.checkIfMarketLaunchProposalExist(ctx, types.ProposalTypeBinaryOptionsMarketLaunch, marketID) {
		metrics.ReportFuncError(k.svcTags)
		log.Infof("the binary options market launch proposal already exists: marketID=%s", marketID.Hex())
		return nil, errors.Wrapf(types.ErrMarketLaunchProposalAlreadyExists, "the binary options market launch proposal already exists: marketID=%s", marketID.Hex())
	}

	_, err = k.BinaryOptionsMarketLaunch(
		ctx,
		msg.Ticker,
		msg.OracleSymbol,
		msg.OracleProvider,
		msg.OracleType,
		msg.OracleScaleFactor,
		msg.MakerFeeRate,
		msg.TakerFeeRate,
		msg.ExpirationTimestamp,
		msg.SettlementTimestamp,
		msg.Admin,
		msg.QuoteDenom,
		msg.MinPriceTickSize,
		msg.MinQuantityTickSize,
	)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching binary options market", err)
		return nil, err
	}

	return &types.MsgInstantBinaryOptionsMarketLaunchResponse{}, nil
}

func (k BinaryOptionsMsgServer) CreateBinaryOptionsLimitOrder(goCtx context.Context, msg *types.MsgCreateBinaryOptionsLimitOrder) (*types.MsgCreateBinaryOptionsLimitOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	account, _ := sdk.AccAddressFromBech32(msg.Sender)

	market := k.GetBinaryOptionsMarket(ctx, msg.Order.MarketID(), true)
	if market == nil {
		k.Logger(ctx).Error("active binary options market doesn't exist", "marketId", msg.Order.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "marketID %s", msg.Order.MarketId)
	}

	requiredMargin := msg.Order.GetRequiredBinaryOptionsMargin(market.OracleScaleFactor)
	if msg.Order.Margin.GT(requiredMargin) {
		// decrease order margin to the required amount if greater, since there's no need to overpay
		msg.Order.Margin = requiredMargin
	}

	orderHash, err := k.createDerivativeLimitOrder(ctx, account, &msg.Order, market, sdk.Dec{})

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	return &types.MsgCreateBinaryOptionsLimitOrderResponse{
		OrderHash: orderHash.Hex(),
	}, nil
}

func (k BinaryOptionsMsgServer) CreateBinaryOptionsMarketOrder(goCtx context.Context, msg *types.MsgCreateBinaryOptionsMarketOrder) (*types.MsgCreateBinaryOptionsMarketOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	account, _ := sdk.AccAddressFromBech32(msg.Sender)

	market := k.GetBinaryOptionsMarket(ctx, msg.Order.MarketID(), true)
	if market == nil {
		k.Logger(ctx).Error("active binary options market doesn't exist", "marketId", msg.Order.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "marketID %s", msg.Order.MarketId)
	}

	requiredMargin := msg.Order.GetRequiredBinaryOptionsMargin(market.OracleScaleFactor)
	if msg.Order.Margin.GT(requiredMargin) {
		// decrease order margin to the required amount if greater, since there's no need to overpay
		msg.Order.Margin = requiredMargin
	}

	orderHash, results, err := k.createDerivativeMarketOrder(ctx, account, &msg.Order, market, sdk.Dec{})
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	resp := &types.MsgCreateBinaryOptionsMarketOrderResponse{
		OrderHash: orderHash.Hex(),
	}

	if results != nil {
		resp.Results = results
	}
	return resp, nil
}

func (k BinaryOptionsMsgServer) CancelBinaryOptionsOrder(goCtx context.Context, msg *types.MsgCancelBinaryOptionsOrder) (*types.MsgCancelBinaryOptionsOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	var (
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SubaccountId)
		marketID     = common.HexToHash(msg.MarketId)
		identifier   = types.GetOrderIdentifier(msg.OrderHash, msg.Cid)
	)

	market := k.GetBinaryOptionsMarketByID(ctx, marketID)

	err := k.cancelDerivativeOrder(ctx, subaccountID, identifier, market, marketID, msg.OrderMask)

	if err != nil {
		return nil, err
	}

	return &types.MsgCancelBinaryOptionsOrderResponse{}, nil
}

func (k BinaryOptionsMsgServer) AdminUpdateBinaryOptionsMarket(goCtx context.Context, msg *types.MsgAdminUpdateBinaryOptionsMarket) (*types.MsgAdminUpdateBinaryOptionsMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	marketID := common.HexToHash(msg.MarketId)

	market := k.GetBinaryOptionsMarketByID(ctx, marketID)
	if market == nil {
		k.Logger(ctx).Error("binary options market doesn't exist", "marketID", msg.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "marketID %s", msg.MarketId)
	}

	if market.Admin != msg.Sender {
		k.Logger(ctx).Error("message sender is not an admin of binary options market", "sender", msg.Sender, "admin", market.Admin)
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrSenderIsNotAnAdmin, "sender %s, admin %s", msg.Sender, market.Admin)
	}

	if market.Status == types.MarketStatus_Demolished {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidMarketStatus, "can't update market that was demolished already")
	}

	expTimestamp, settlementTimestamp := market.ExpirationTimestamp, market.SettlementTimestamp

	if msg.ExpirationTimestamp > 0 {
		if msg.ExpirationTimestamp <= ctx.BlockTime().Unix() {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrapf(types.ErrInvalidExpiry, "expiration timestamp %d is in the past", msg.ExpirationTimestamp)
		}
		if market.Status != types.MarketStatus_Active {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrap(types.ErrInvalidExpiry, "cannot change expiration time of an expired market")
		}
		expTimestamp = msg.ExpirationTimestamp
	}

	if msg.SettlementTimestamp > 0 {
		if msg.SettlementTimestamp <= ctx.BlockTime().Unix() {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrapf(types.ErrInvalidSettlement, "SettlementTimestamp %d should be in future", msg.SettlementTimestamp)
		}
		if msg.SettlementTimestamp <= expTimestamp {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrap(types.ErrInvalidSettlement, "settlement time must be after expiration time")
		}
		settlementTimestamp = msg.SettlementTimestamp
	}

	if expTimestamp >= settlementTimestamp {
		return nil, errors.Wrap(types.ErrInvalidExpiry, "expiration timestamp should be prior to settlement timestamp")
	}

	// we convert it to UpdateProposal type to not duplicate the code
	newParams := types.BinaryOptionsMarketParamUpdateProposal{
		MarketId:            msg.MarketId,
		Status:              msg.Status,
		ExpirationTimestamp: msg.ExpirationTimestamp,
		SettlementTimestamp: msg.SettlementTimestamp,
		SettlementPrice:     msg.SettlementPrice,
	}
	// schedule market param change in transient store
	if err := k.ScheduleBinaryOptionsMarketParamUpdate(ctx, &newParams); err != nil {
		return nil, err
	}

	return &types.MsgAdminUpdateBinaryOptionsMarketResponse{}, nil
}

func (k BinaryOptionsMsgServer) BatchCancelBinaryOptionsOrders(goCtx context.Context, msg *types.MsgBatchCancelBinaryOptionsOrders) (*types.MsgBatchCancelBinaryOptionsOrdersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	successes := make([]bool, len(msg.Data))
	for idx := range msg.Data {
		if _, err := k.CancelBinaryOptionsOrder(goCtx, &types.MsgCancelBinaryOptionsOrder{
			Sender:       msg.Sender,
			MarketId:     msg.Data[idx].MarketId,
			SubaccountId: msg.Data[idx].SubaccountId,
			OrderHash:    msg.Data[idx].OrderHash,
			Cid:          msg.Data[idx].Cid,
		}); err != nil {
			metrics.ReportFuncError(k.svcTags)
		} else {
			successes[idx] = true
		}
	}

	return &types.MsgBatchCancelBinaryOptionsOrdersResponse{
		Success: successes,
	}, nil
}
