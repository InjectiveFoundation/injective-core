package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type BinaryOptionsMsgServer struct {
	*Keeper
	svcTags metrics.Tags
}

// NewBinaryOptionsMsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper for binary options market functions.
func NewBinaryOptionsMsgServerImpl(keeper *Keeper) BinaryOptionsMsgServer {
	return BinaryOptionsMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "bin_msg_h",
		},
	}
}

func (k BinaryOptionsMsgServer) InstantBinaryOptionsMarketLaunch(
	goCtx context.Context, msg *v2.MsgInstantBinaryOptionsMarketLaunch,
) (*v2.MsgInstantBinaryOptionsMarketLaunchResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	fee := k.GetParams(ctx).BinaryOptionsMarketInstantListingFee
	if err := k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching binary options market", err)
		return nil, err
	}

	if err := k.checkDenomMinNotional(ctx, senderAddr, msg.QuoteDenom, msg.MinNotional); err != nil {
		return nil, err
	}

	// check if the market launch proposal already exists
	marketID := types.NewBinaryOptionsMarketID(msg.Ticker, msg.QuoteDenom, msg.OracleSymbol, msg.OracleProvider, msg.OracleType)
	if k.checkIfMarketLaunchProposalExist(
		ctx, marketID, types.ProposalTypeBinaryOptionsMarketLaunch, v2.ProposalTypeBinaryOptionsMarketLaunch,
	) {
		metrics.ReportFuncError(k.svcTags)
		ctx.Logger().Info("the binary options market launch proposal already exists", "marketID", marketID.Hex())
		return nil, errors.Wrapf(
			types.ErrMarketLaunchProposalAlreadyExists,
			"the binary options market launch proposal already exists: marketID=%s", marketID.Hex(),
		)
	}

	_, err := k.BinaryOptionsMarketLaunch(
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
		msg.MinNotional,
		msg.OpenNotionalCap,
	)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching binary options market", err)
		return nil, err
	}

	return &v2.MsgInstantBinaryOptionsMarketLaunchResponse{}, nil
}

func (k BinaryOptionsMsgServer) CreateBinaryOptionsLimitOrder(
	goCtx context.Context, msg *v2.MsgCreateBinaryOptionsLimitOrder,
) (*v2.MsgCreateBinaryOptionsLimitOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateBinaryOptionsLimitOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CreateBinaryOptionsLimitOrder",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

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

	orderHash, err := k.createDerivativeLimitOrder(ctx, account, &msg.Order, market, math.LegacyDec{})

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	return &v2.MsgCreateBinaryOptionsLimitOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.OrderInfo.Cid,
	}, nil
}

func (k BinaryOptionsMsgServer) CreateBinaryOptionsMarketOrder(
	goCtx context.Context, msg *v2.MsgCreateBinaryOptionsMarketOrder,
) (*v2.MsgCreateBinaryOptionsMarketOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateBinaryOptionsMarketOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CreateBinaryOptionsMarketOrder",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	account, _ := sdk.AccAddressFromBech32(msg.Sender)

	market := k.GetBinaryOptionsMarket(ctx, msg.Order.MarketID(), true)
	if market == nil {
		k.Logger(ctx).Error("active binary options market doesn't exist", "marketId", msg.Order.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "marketID %s", msg.Order.MarketId)
	}

	orderHash, results, err := k.createBinaryOptionsMarketOrderWithResultsForAtomicExecution(
		ctx,
		account,
		&msg.Order,
		market,
		math.LegacyDec{},
	)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	resp := &v2.MsgCreateBinaryOptionsMarketOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}

	if results != nil {
		resp.Results = results
	}

	return resp, nil
}

func (k BinaryOptionsMsgServer) CancelBinaryOptionsOrder(
	goCtx context.Context, msg *v2.MsgCancelBinaryOptionsOrder,
) (*v2.MsgCancelBinaryOptionsOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCancelBinaryOptionsOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CancelBinaryOptionsOrder",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	var (
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SubaccountId)
		marketID     = common.HexToHash(msg.MarketId)
		identifier   = types.GetOrderIdentifier(msg.OrderHash, msg.Cid)
	)

	market := k.GetBinaryOptionsMarketByID(ctx, marketID)
	err := k.cancelDerivativeOrder(ctx, subaccountID, identifier, market, marketID, msg.OrderMask)

	if err != nil {
		k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, msg.OrderHash, msg.Cid, err))
		return nil, err
	}

	return &v2.MsgCancelBinaryOptionsOrderResponse{}, nil
}

func (k BinaryOptionsMsgServer) AdminUpdateBinaryOptionsMarket(
	goCtx context.Context, msg *v2.MsgAdminUpdateBinaryOptionsMarket,
) (*v2.MsgAdminUpdateBinaryOptionsMarketResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

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

	if market.Status == v2.MarketStatus_Demolished {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidMarketStatus, "can't update market that was demolished already")
	}

	expTimestamp, settlementTimestamp := market.ExpirationTimestamp, market.SettlementTimestamp

	if msg.ExpirationTimestamp > 0 {
		if msg.ExpirationTimestamp <= ctx.BlockTime().Unix() {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrapf(types.ErrInvalidExpiry, "expiration timestamp %d is in the past", msg.ExpirationTimestamp)
		}
		if market.Status != v2.MarketStatus_Active {
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
	newParams := v2.BinaryOptionsMarketParamUpdateProposal{
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

	return &v2.MsgAdminUpdateBinaryOptionsMarketResponse{}, nil
}

func (k BinaryOptionsMsgServer) BatchCancelBinaryOptionsOrders(
	goCtx context.Context, msg *v2.MsgBatchCancelBinaryOptionsOrders,
) (*v2.MsgBatchCancelBinaryOptionsOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	gasConsumedBefore := ctx.GasMeter().GasConsumed()

	// todo: remove after QA
	defer func() {
		// no need to do anything here with gas meter, since it's handled per CancelBinaryOptionsOrder call
		totalGas := ctx.GasMeter().GasConsumed()
		k.Logger(ctx).Info("BatchCancelBinaryOptionsOrders",
			"gas_ante", gasConsumedBefore,
			"gas_msg", totalGas-gasConsumedBefore,
			"gas_total", totalGas,
			"sender", msg.Sender,
		)
	}()

	successes := make([]bool, len(msg.Data))
	for idx := range msg.Data {
		if _, err := k.CancelBinaryOptionsOrder(goCtx, &v2.MsgCancelBinaryOptionsOrder{
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

	return &v2.MsgBatchCancelBinaryOptionsOrdersResponse{Success: successes}, nil
}

func (k DerivativesMsgServer) LaunchBinaryOptionsMarket(
	goCtx context.Context, msg *v2.MsgBinaryOptionsMarketLaunch,
) (*v2.MsgBinaryOptionsMarketLaunchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleBinaryOptionsMarketLaunchProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgBinaryOptionsMarketLaunchResponse{}, nil
}

func (k DerivativesMsgServer) BinaryOptionsMarketParamUpdate(
	goCtx context.Context, msg *v2.MsgBinaryOptionsMarketParamUpdate,
) (*v2.MsgBinaryOptionsMarketParamUpdateResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleBinaryOptionsMarketParamUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgBinaryOptionsMarketParamUpdateResponse{}, nil
}
