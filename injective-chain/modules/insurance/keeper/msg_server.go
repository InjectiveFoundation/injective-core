package keeper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	"github.com/InjectiveLabs/metrics"
)

type msgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the insurance MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "insurance_h",
		},
	}
}

var _ types.MsgServer = msgServer{}

// CreateInsuranceFund is wrapper of keeper.CreateInsuranceFund
func (k msgServer) CreateInsuranceFund(goCtx context.Context, msg *types.MsgCreateInsuranceFund) (*types.MsgCreateInsuranceFundResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if msg.Expiry != -1 && msg.Expiry < ctx.BlockTime().Unix() {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrInvalidExpirationTime
	}

	if err := k.Keeper.CreateInsuranceFund(ctx, sender, msg.InitialDeposit, msg.Ticker, msg.QuoteDenom, msg.OracleBase, msg.OracleQuote, msg.OracleType, msg.Expiry); err != nil {
		metrics.ReportFuncError(k.svcTags)
		logger.Errorln("Insurance fund creation failed", err)
		return nil, err
	}

	logger.Debugf("successfully created insurance fund for %s", msg.Ticker)
	return &types.MsgCreateInsuranceFundResponse{}, nil
}

// Underwrite is wrapper of keeper.UnderwriteInsuranceFund
func (k msgServer) Underwrite(goCtx context.Context, msg *types.MsgUnderwrite) (*types.MsgUnderwriteResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	marketID := common.HexToHash(msg.MarketId)
	if err := k.Keeper.UnderwriteInsuranceFund(ctx, sender, marketID, msg.Deposit); err != nil {
		metrics.ReportFuncError(k.svcTags)
		logger.Errorln("underwriting insurance fund failed", err)
		return nil, err
	}

	logger.Debugf("successfully underwrote insurance fund for %s market", marketID.Hex())
	return &types.MsgUnderwriteResponse{}, nil
}

// RequestRedemption is wrapper of keeper.RequestInsuranceFundRedemption
func (k msgServer) RequestRedemption(goCtx context.Context, msg *types.MsgRequestRedemption) (*types.MsgRequestRedemptionResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}
	marketID := common.HexToHash(msg.MarketId)
	err = k.Keeper.RequestInsuranceFundRedemption(ctx, sender, marketID, msg.Amount)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		logger.Errorln("requesting redemption for insurance fund failed", err)
		return nil, err
	}

	logger.Debugf("successfully requested redemption for insurance fund on %s market", marketID.Hex())
	return &types.MsgRequestRedemptionResponse{}, nil
}
