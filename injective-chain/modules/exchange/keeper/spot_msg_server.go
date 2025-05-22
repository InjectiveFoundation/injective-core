package keeper

import (
	"context"
	"errors"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type SpotMsgServer struct {
	*Keeper
	svcTags metrics.Tags
}

// NewSpotMsgServerImpl returns an implementation of the bank MsgServer interface for the provided Keeper for spot market functions.
func NewSpotMsgServerImpl(keeper *Keeper) SpotMsgServer {
	return SpotMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "spot_msg_h",
		},
	}
}

func (k SpotMsgServer) InstantSpotMarketLaunch(
	goCtx context.Context, msg *v2.MsgInstantSpotMarketLaunch,
) (*v2.MsgInstantSpotMarketLaunchResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.checkDenomMinNotional(ctx, sender, msg.QuoteDenom, msg.MinNotional); err != nil {
		return nil, err
	}

	// check if the market launch proposal already exists
	marketID := types.NewSpotMarketID(msg.BaseDenom, msg.QuoteDenom)
	if k.checkIfMarketLaunchProposalExist(ctx, marketID, types.ProposalTypeSpotMarketLaunch, v2.ProposalTypeSpotMarketLaunch) {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("the spot market launch proposal already exists: marketID=%s", marketID.Hex())
		return nil, types.ErrMarketLaunchProposalAlreadyExists.Wrapf(
			"the spot market launch proposal already exists: marketID=%s", marketID.Hex(),
		)
	}

	_, err := k.SpotMarketLaunch(
		ctx,
		msg.Ticker,
		msg.BaseDenom,
		msg.QuoteDenom,
		msg.MinPriceTickSize,
		msg.MinQuantityTickSize,
		msg.MinNotional,
		msg.BaseDecimals,
		msg.QuoteDecimals,
	)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching spot market", err)
		return nil, err
	}

	fee := k.GetParams(ctx).SpotMarketInstantListingFee
	if err = k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, sender); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching spot market", err)
		return nil, err
	}

	return &v2.MsgInstantSpotMarketLaunchResponse{}, nil
}

func (k SpotMsgServer) UpdateSpotMarket(c context.Context, msg *v2.MsgUpdateSpotMarket) (*v2.MsgUpdateSpotMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	market := k.GetSpotMarketByID(ctx, common.HexToHash(msg.MarketId))
	if market == nil {
		return nil, sdkerrors.Wrap(types.ErrSpotMarketNotFound, "unknown market id")
	}

	if market.Admin == "" || market.Admin != msg.Admin {
		return nil, sdkerrors.Wrapf(types.ErrInvalidAccessLevel, "market belongs to another admin (%v)", market.Admin)
	}

	if market.AdminPermissions == 0 {
		return nil, sdkerrors.Wrap(types.ErrInvalidAccessLevel, "no permissions found")
	}

	permissions := types.MarketAdminPermissions(market.AdminPermissions)

	if msg.HasTickerUpdate() {
		if !permissions.HasPerm(types.TickerPerm) {
			return nil, sdkerrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update market ticker")
		}

		market.Ticker = msg.NewTicker
	}

	if msg.HasMinPriceTickSizeUpdate() {
		if !permissions.HasPerm(types.MinPriceTickSizePerm) {
			return nil, sdkerrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update min_price_tick_size")
		}

		market.MinPriceTickSize = msg.NewMinPriceTickSize
	}

	if msg.HasMinQuantityTickSizeUpdate() {
		if !permissions.HasPerm(types.MinQuantityTickSizePerm) {
			return nil, sdkerrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update market min_quantity_tick_size")
		}

		market.MinQuantityTickSize = msg.NewMinQuantityTickSize

	}

	if msg.HasMinNotionalUpdate() {
		if !permissions.HasPerm(types.MinNotionalPerm) {
			return nil, sdkerrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update market min_notional")
		}
		if err := k.checkDenomMinNotional(ctx, sdk.AccAddress(msg.Admin), market.QuoteDenom, msg.NewMinNotional); err != nil {
			return nil, err
		}

		market.MinNotional = msg.NewMinNotional
	}

	k.SetSpotMarket(ctx, market)

	return &v2.MsgUpdateSpotMarketResponse{}, nil
}

func (k SpotMsgServer) CreateSpotLimitOrder(
	goCtx context.Context, msg *v2.MsgCreateSpotLimitOrder,
) (*v2.MsgCreateSpotLimitOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateSpotLimitOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CreateSpotLimitOrder",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
				"cid", msg.Order.Cid(),
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	account, _ := sdk.AccAddressFromBech32(msg.Sender)
	orderHash, err := k.createSpotLimitOrder(ctx, account, &msg.Order, nil)
	if err != nil {
		return nil, err
	}

	return &v2.MsgCreateSpotLimitOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}, nil
}

func (k SpotMsgServer) CreateSpotMarketOrder(
	goCtx context.Context, msg *v2.MsgCreateSpotMarketOrder,
) (*v2.MsgCreateSpotMarketOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateSpotMarketOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CreateSpotMarketOrder",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
				"cid", msg.Order.Cid(),
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	if k.IsPostOnlyMode(ctx) {
		return nil, types.ErrPostOnlyMode.Wrapf(
			"cannot create market orders in post only mode until height %d",
			k.GetParams(ctx).PostOnlyModeHeightThreshold,
		)
	}

	var (
		marketID     = common.HexToHash(msg.Order.MarketId)
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.Order.OrderInfo.SubaccountId)
	)

	// populate the order with the actual subaccountID value, since it might be a nonce value
	msg.Order.OrderInfo.SubaccountId = subaccountID.Hex()

	market := k.GetSpotMarket(ctx, marketID, true)
	if market == nil {
		k.Logger(ctx).Error("active spot market doesn't exist", "marketId", msg.Order.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrSpotMarketNotFound.Wrapf("active spot market doesn't exist %s", msg.Order.MarketId)
	}

	if err := msg.Order.CheckTickSize(market.MinPriceTickSize, market.MinQuantityTickSize); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if err := msg.Order.CheckNotional(market.MinNotional); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if k.existsCid(ctx, subaccountID, msg.Order.OrderInfo.Cid) {
		return nil, types.ErrClientOrderIdAlreadyExists
	}

	isAtomic := msg.Order.OrderType.IsAtomic()
	if isAtomic {
		err := k.ensureValidAccessLevelForAtomicExecution(ctx, sender)
		if err != nil {
			return nil, err
		}
	}

	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, subaccountID)

	orderHash, err := msg.Order.ComputeOrderHash(subaccountNonce.Nonce)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	marginDenom := msg.Order.GetMarginDenom(market)

	bestPrice := k.GetBestSpotLimitOrderPrice(ctx, marketID, !msg.Order.IsBuy())

	if bestPrice == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrNoLiquidity
	} else if msg.Order.IsBuy() && msg.Order.OrderInfo.Price.LT(*bestPrice) ||
		!msg.Order.IsBuy() && msg.Order.OrderInfo.Price.GT(*bestPrice) {
		// If market buy order worst price less than best sell order price
		// or market sell order worst price greater than best buy order price
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrSlippageExceedsWorstPrice
	}

	feeRate := market.TakerFeeRate
	if isAtomic {
		feeRate = feeRate.Mul(k.Keeper.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, types.MarketType_Spot))
	}

	balanceHold := msg.Order.GetMarketOrderBalanceHold(feeRate, *bestPrice)
	var chainFormattedBalanceHold math.LegacyDec
	if msg.Order.IsBuy() {
		chainFormattedBalanceHold = market.NotionalToChainFormat(balanceHold)
	} else {
		chainFormattedBalanceHold = market.QuantityToChainFormat(balanceHold)
	}

	if err := k.chargeAccount(ctx, subaccountID, marginDenom, chainFormattedBalanceHold); err != nil {
		return nil, err
	}

	marketOrder := msg.Order.ToSpotMarketOrder(sender, balanceHold, orderHash)

	var marketOrderResults *v2.SpotMarketOrderResults
	if isAtomic {
		marketOrderResults = k.ExecuteAtomicSpotMarketOrder(ctx, market, marketOrder, feeRate)
	} else {
		k.SetTransientSpotMarketOrder(ctx, marketOrder, &msg.Order, orderHash)
	}

	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)

	response := &v2.MsgCreateSpotMarketOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}

	if marketOrderResults != nil {
		response.Results = marketOrderResults
	}

	return response, nil
}

func (k SpotMsgServer) BatchCreateSpotLimitOrders(
	goCtx context.Context, msg *v2.MsgBatchCreateSpotLimitOrders,
) (*v2.MsgBatchCreateSpotLimitOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	// Naive, unoptimized implementation
	var (
		orderHashes       = make([]string, len(msg.Orders))
		createdOrdersCids = make([]string, 0)
		failedOrdersCids  = make([]string, 0)

		sender         = sdk.MustAccAddressFromBech32(msg.Sender)
		orderFailEvent = v2.EventOrderFail{
			Account: sender.Bytes(),
			Hashes:  make([][]byte, 0),
			Flags:   make([]uint32, 0),
			Cids:    make([]string, 0),
		}
	)

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgBatchCreateSpotLimitOrders")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("BatchCreateSpotLimitOrders",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	for idx := range msg.Orders {
		order := msg.Orders[idx]
		if orderHash, err := k.createSpotLimitOrder(ctx, sender, &order, nil); err != nil {
			metrics.ReportFuncError(k.svcTags)
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				orderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, order.Cid(), sdkerror.ABCICode())
				failedOrdersCids = append(failedOrdersCids, order.Cid())
			}
		} else {
			orderHashes[idx] = orderHash.Hex()
			createdOrdersCids = append(createdOrdersCids, order.Cid())
		}
	}
	if !orderFailEvent.IsEmpty() {
		k.EmitEvent(ctx, &orderFailEvent)
	}

	return &v2.MsgBatchCreateSpotLimitOrdersResponse{
		OrderHashes:       orderHashes,
		CreatedOrdersCids: createdOrdersCids,
		FailedOrdersCids:  failedOrdersCids,
	}, nil
}

func (k SpotMsgServer) CancelSpotOrder(goCtx context.Context, msg *v2.MsgCancelSpotOrder) (*v2.MsgCancelSpotOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	var (
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SubaccountId)
		marketID     = common.HexToHash(msg.MarketId)
		identifier   = types.GetOrderIdentifier(msg.OrderHash, msg.Cid)
	)

	// Reject if spot market id does not reference an active, suspended or demolished spot market
	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCancelSpotOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CancelSpotOrder",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
				"cid", msg.Cid,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	market := k.GetSpotMarketByID(ctx, marketID)
	err := k.cancelSpotLimitOrderWithIdentifier(ctx, subaccountID, identifier, market, marketID)
	if err != nil {
		k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, msg.OrderHash, msg.Cid, err))
	}

	return &v2.MsgCancelSpotOrderResponse{}, err
}

func (k SpotMsgServer) BatchCancelSpotOrders(
	goCtx context.Context, msg *v2.MsgBatchCancelSpotOrders,
) (*v2.MsgBatchCancelSpotOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	gasConsumedBefore := ctx.GasMeter().GasConsumed()

	// todo: remove after QA
	defer func() {
		// no need to do anything here with gas meter, since it's handled per MsgCancelSpotOrder call
		totalGas := ctx.GasMeter().GasConsumed()
		k.Logger(ctx).Info("MsgBatchCancelSpotOrders",
			"gas_ante", gasConsumedBefore,
			"gas_msg", totalGas-gasConsumedBefore,
			"gas_total", totalGas,
			"sender", msg.Sender,
		)
	}()

	// Naive, unoptimized implementation
	successes := make([]bool, len(msg.Data))
	for idx := range msg.Data {
		if _, err := k.CancelSpotOrder(goCtx, &v2.MsgCancelSpotOrder{
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

	return &v2.MsgBatchCancelSpotOrdersResponse{Success: successes}, nil
}

func (k SpotMsgServer) LaunchSpotMarket(goCtx context.Context, msg *v2.MsgSpotMarketLaunch) (*v2.MsgSpotMarketLaunchResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleSpotMarketLaunchProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgSpotMarketLaunchResponse{}, nil
}

func (k SpotMsgServer) SpotMarketParamUpdate(
	goCtx context.Context, msg *v2.MsgSpotMarketParamUpdate,
) (*v2.MsgSpotMarketParamUpdateResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleSpotMarketParamUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgSpotMarketParamUpdateResponse{}, nil
}
