package keeper

import (
	"context"
	"errors"
	"fmt"

	cosmoserrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type DerivativesMsgServer struct {
	*Keeper
	svcTags metrics.Tags
}

// Using a map for the list of enabled oracle types to improve lookup time
var enabledOracleTypes = map[oracletypes.OracleType]struct{}{
	oracletypes.OracleType_Coinbase:  {},
	oracletypes.OracleType_Chainlink: {},
	oracletypes.OracleType_Razor:     {},
	oracletypes.OracleType_Dia:       {},
	oracletypes.OracleType_API3:      {},
	oracletypes.OracleType_Uma:       {},
	oracletypes.OracleType_Pyth:      {},
	oracletypes.OracleType_BandIBC:   {},
	oracletypes.OracleType_Stork:     {},
}

// NewDerivativesMsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper
// for derivatives market functions.
func NewDerivativesMsgServerImpl(keeper *Keeper) DerivativesMsgServer {
	return DerivativesMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "dvt_msg_h",
		},
	}
}

func (k DerivativesMsgServer) InstantPerpetualMarketLaunch(
	goCtx context.Context, msg *v2.MsgInstantPerpetualMarketLaunch,
) (*v2.MsgInstantPerpetualMarketLaunchResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	isRegistrationAllowed := k.IsAdmin(ctx, msg.Sender)

	if !k.GetIsInstantDerivativeMarketLaunchEnabled(ctx) {
		return nil, types.ErrFeatureDisabled
	}

	if !isRegistrationAllowed {
		return nil, sdkerrors.ErrUnauthorized.Wrap("Unauthorized to instant launch a perpetual market")
	}

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.checkDenomMinNotional(ctx, senderAddr, msg.QuoteDenom, msg.MinNotional); err != nil {
		return nil, err
	}

	// check if the market launch proposal already exists
	marketID := types.NewPerpetualMarketID(msg.Ticker, msg.QuoteDenom, msg.OracleBase, msg.OracleQuote, msg.OracleType)
	if k.checkIfMarketLaunchProposalExist(ctx, marketID, types.ProposalTypePerpetualMarketLaunch, v2.ProposalTypePerpetualMarketLaunch) {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("the perpetual market launch proposal already exists: marketID=%s", marketID.Hex())
		return nil, types.ErrMarketLaunchProposalAlreadyExists.Wrapf(
			"the perpetual market launch proposal already exists: marketID=%s", marketID.Hex(),
		)
	}

	fee := k.GetParams(ctx).DerivativeMarketInstantListingFee
	err := k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	adminInfo := v2.EmptyAdminInfo()
	_, _, err = k.PerpetualMarketLaunch(
		ctx, msg.Ticker, msg.QuoteDenom, msg.OracleBase, msg.OracleQuote, msg.OracleScaleFactor, msg.OracleType,
		msg.InitialMarginRatio, msg.MaintenanceMarginRatio, msg.ReduceMarginRatio,
		msg.MakerFeeRate, msg.TakerFeeRate, msg.MinPriceTickSize, msg.MinQuantityTickSize, msg.MinNotional, &adminInfo,
	)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	return &v2.MsgInstantPerpetualMarketLaunchResponse{}, err
}

func (k DerivativesMsgServer) InstantExpiryFuturesMarketLaunch(
	goCtx context.Context, msg *v2.MsgInstantExpiryFuturesMarketLaunch,
) (*v2.MsgInstantExpiryFuturesMarketLaunchResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	isRegistrationAllowed := k.IsAdmin(ctx, msg.Sender)

	if !k.GetIsInstantDerivativeMarketLaunchEnabled(ctx) {
		return nil, types.ErrFeatureDisabled
	}

	if !isRegistrationAllowed {
		return nil, sdkerrors.ErrUnauthorized.Wrap("Unauthorized to instant launch an expiry futures market")
	}

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.checkDenomMinNotional(ctx, senderAddr, msg.QuoteDenom, msg.MinNotional); err != nil {
		return nil, err
	}

	// check if the market launch proposal already exists
	marketID := types.NewExpiryFuturesMarketID(msg.Ticker, msg.QuoteDenom, msg.OracleBase, msg.OracleQuote, msg.OracleType, msg.Expiry)
	if k.checkIfMarketLaunchProposalExist(
		ctx, marketID, types.ProposalTypeExpiryFuturesMarketLaunch, v2.ProposalTypeExpiryFuturesMarketLaunch,
	) {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("the expiry futures market launch proposal already exists: marketID=%s", marketID.Hex())
		return nil, types.ErrMarketLaunchProposalAlreadyExists.Wrapf(
			"the expiry futures market launch proposal already exists: marketID=%s", marketID.Hex(),
		)
	}

	fee := k.GetParams(ctx).DerivativeMarketInstantListingFee
	err := k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	adminInfo := v2.EmptyAdminInfo()
	if _, _, err := k.ExpiryFuturesMarketLaunch(
		ctx, msg.Ticker, msg.QuoteDenom,
		msg.OracleBase, msg.OracleQuote, msg.OracleScaleFactor, msg.OracleType, msg.Expiry,
		msg.InitialMarginRatio, msg.MaintenanceMarginRatio, msg.ReduceMarginRatio,
		msg.MakerFeeRate, msg.TakerFeeRate, msg.MinPriceTickSize, msg.MinQuantityTickSize, msg.MinNotional, &adminInfo,
	); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	return &v2.MsgInstantExpiryFuturesMarketLaunchResponse{}, err
}

func (k DerivativesMsgServer) UpdateDerivativeMarket(c context.Context, msg *v2.MsgUpdateDerivativeMarket) (*v2.MsgUpdateDerivativeMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	market := k.GetDerivativeMarketByID(ctx, common.HexToHash(msg.MarketId))
	if market == nil {
		return nil, cosmoserrors.Wrap(types.ErrDerivativeMarketNotFound, "unknown market id")
	}

	if market.Admin == "" || market.Admin != msg.Admin {
		return nil, cosmoserrors.Wrapf(types.ErrInvalidAccessLevel, "market belongs to another admin (%v)", market.Admin)
	}

	if market.AdminPermissions == 0 {
		return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "no permissions found")
	}

	permissions := types.MarketAdminPermissions(market.AdminPermissions)

	if msg.HasTickerUpdate() {
		if !permissions.HasPerm(types.TickerPerm) {
			return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update ticker")
		}

		market.Ticker = msg.NewTicker
	}

	if msg.HasMinPriceTickSizeUpdate() {
		if !permissions.HasPerm(types.MinPriceTickSizePerm) {
			return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update min_price_tick_size")
		}

		market.MinPriceTickSize = msg.NewMinPriceTickSize
	}

	if msg.HasMinQuantityTickSizeUpdate() {
		if !permissions.HasPerm(types.MinQuantityTickSizePerm) {
			return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update min_quantity_tick_size")
		}

		market.MinQuantityTickSize = msg.NewMinQuantityTickSize
	}

	if msg.HasMinNotionalUpdate() {
		if !permissions.HasPerm(types.MinNotionalPerm) {
			return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update market min_notional")
		}
		if err := k.checkDenomMinNotional(ctx, sdk.AccAddress(msg.Admin), market.QuoteDenom, msg.NewMinNotional); err != nil {
			return nil, err
		}
		market.MinNotional = msg.NewMinNotional
	}

	params := k.GetParams(ctx)

	if msg.HasInitialMarginRatioUpdate() {
		if !permissions.HasPerm(types.InitialMarginRatioPerm) {
			return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update initial_margin_ratio")
		}

		// disallow admins from decreasing initial margin ratio below the default param
		if msg.NewInitialMarginRatio.LT(params.DefaultInitialMarginRatio) {
			return nil, types.ErrInvalidMarginRatio
		}

		market.InitialMarginRatio = msg.NewInitialMarginRatio
	}

	if msg.HasMaintenanceMarginRatioUpdate() {
		if !permissions.HasPerm(types.MaintenanceMarginRatioPerm) {
			return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update maintenance_margin_ratio")
		}

		// disallow admins from decreasing maintenance margin ratio below the default param
		if msg.NewMaintenanceMarginRatio.LT(params.DefaultMaintenanceMarginRatio) {
			return nil, types.ErrInvalidMarginRatio
		}

		market.MaintenanceMarginRatio = msg.NewMaintenanceMarginRatio
	}

	if msg.HasReduceMarginRatioUpdate() {
		if !permissions.HasPerm(types.ReduceMarginRatioPerm) {
			return nil, cosmoserrors.Wrap(types.ErrInvalidAccessLevel, "admin does not have permission to update reduce_margin_ratio")
		}

		// disallow admins from decreasing reduce margin ratio below the default param
		if msg.NewReduceMarginRatio.LT(params.DefaultReduceMarginRatio) {
			return nil, types.ErrInvalidMarginRatio
		}

		market.ReduceMarginRatio = msg.NewReduceMarginRatio
	}

	if market.InitialMarginRatio.LTE(market.MaintenanceMarginRatio) {
		return nil, types.ErrMarginsRelation
	}

	if market.ReduceMarginRatio.LT(market.InitialMarginRatio) {
		return nil, types.ErrMarginsRelation
	}

	k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, nil)

	return &v2.MsgUpdateDerivativeMarketResponse{}, nil
}

func (k DerivativesMsgServer) CreateDerivativeLimitOrder(
	goCtx context.Context, msg *v2.MsgCreateDerivativeLimitOrder,
) (*v2.MsgCreateDerivativeLimitOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateDerivativeLimitOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CreateDerivativeLimitOrder",
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

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, msg.Order.MarketID(), true)
	if market == nil || markPrice.IsNil() {
		k.Logger(ctx).Error(
			"active derivative market with valid mark price doesn't exist",
			"marketId", msg.Order.MarketId,
			"mark price", markPrice.String(),
		)
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", msg.Order.MarketId)
	}

	orderHash, err := k.createDerivativeLimitOrder(ctx, account, &msg.Order, market, markPrice)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	return &v2.MsgCreateDerivativeLimitOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}, nil
}

func (k DerivativesMsgServer) BatchCreateDerivativeLimitOrders(
	goCtx context.Context, msg *v2.MsgBatchCreateDerivativeLimitOrders,
) (*v2.MsgBatchCreateDerivativeLimitOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgBatchCreateDerivativeLimitOrders")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("BatchCreateDerivativeLimitOrders",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	orderFailEvent := v2.EventOrderFail{
		Account: sender.Bytes(),
		Hashes:  make([][]byte, 0),
		Flags:   make([]uint32, 0),
		Cids:    make([]string, 0),
	}

	marketsCache := make(map[common.Hash]*v2.FullDerivativeMarket)
	orderHashes := make([]string, len(msg.Orders))
	createdOrdersCids := make([]string, 0)
	failedOrdersCids := make([]string, 0)

	for idx := range msg.Orders {
		orderHash, createdCid, failedCid := k.createDerivativeLimitOrderFromBatch(ctx, sender, msg.Orders[idx], marketsCache, &orderFailEvent)
		orderHashes[idx] = orderHash

		if createdCid != "" {
			createdOrdersCids = append(createdOrdersCids, createdCid)
		}

		if failedCid != "" {
			failedOrdersCids = append(failedOrdersCids, failedCid)
		}
	}

	if !orderFailEvent.IsEmpty() {
		k.EmitEvent(ctx, &orderFailEvent)
	}

	return &v2.MsgBatchCreateDerivativeLimitOrdersResponse{
		OrderHashes:       orderHashes,
		CreatedOrdersCids: createdOrdersCids,
		FailedOrdersCids:  failedOrdersCids,
	}, nil
}

// createDerivativeLimitOrderFromBatch processes a single derivative limit order from a batch
func (k DerivativesMsgServer) createDerivativeLimitOrderFromBatch(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order v2.DerivativeOrder,
	marketsCache map[common.Hash]*v2.FullDerivativeMarket,
	orderFailEvent *v2.EventOrderFail,
) (orderHashString, createdCid, failedCid string) {
	marketID := order.MarketID()

	fullMarket, ok := marketsCache[marketID]
	if !ok {
		market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)

		// edge case when active market doesn't exist
		if market == nil || markPrice.IsNil() {
			return fmt.Sprintf("%d", types.ErrDerivativeMarketNotFound.ABCICode()), "", ""
		}

		fullMarket = &v2.FullDerivativeMarket{Market: market, MarkPrice: markPrice}
		marketsCache[marketID] = fullMarket
	}

	orderHash, err := k.createDerivativeLimitOrder(ctx, sender, &order, fullMarket.Market, fullMarket.MarkPrice)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		sdkerror := &cosmoserrors.Error{}
		if errors.As(err, &sdkerror) {
			orderFailEvent.AddOrderFail(orderHash, order.Cid(), sdkerror.ABCICode())
			return fmt.Sprintf("%d", sdkerror.ABCICode()), "", order.Cid()
		}
		return "", "", ""
	}

	return orderHash.Hex(), order.Cid(), ""
}

func (k DerivativesMsgServer) CreateDerivativeMarketOrder(
	goCtx context.Context, msg *v2.MsgCreateDerivativeMarketOrder,
) (*v2.MsgCreateDerivativeMarketOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateDerivativeMarketOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CreateDerivativeMarketOrder",
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

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, msg.Order.MarketID(), true)
	if market == nil {
		k.Logger(ctx).Error("active derivative market doesn't exist", "marketId", msg.Order.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", msg.Order.MarketId)
	}

	orderHash, results, err := k.createDerivativeMarketOrder(ctx, account, &msg.Order, market, markPrice)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	resp := &v2.MsgCreateDerivativeMarketOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}
	if results != nil {
		resp.Results = results
	}
	return resp, nil
}

func (k DerivativesMsgServer) CancelDerivativeOrder(
	goCtx context.Context, msg *v2.MsgCancelDerivativeOrder,
) (*v2.MsgCancelDerivativeOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCancelDerivativeOrder")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("CancelDerivativeOrder",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
				"cid", msg.Cid,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	var (
		marketID     = common.HexToHash(msg.MarketId)
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SubaccountId)
		identifier   = types.GetOrderIdentifier(msg.OrderHash, msg.Cid)
	)

	market := k.GetDerivativeMarketByID(ctx, marketID)

	err := k.cancelDerivativeOrder(ctx, subaccountID, identifier, market, marketID, msg.OrderMask)

	if err != nil {
		k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, msg.OrderHash, msg.Cid, err))
		return nil, err
	}

	return &v2.MsgCancelDerivativeOrderResponse{}, nil
}

func (k DerivativesMsgServer) BatchCancelDerivativeOrders(
	goCtx context.Context, msg *v2.MsgBatchCancelDerivativeOrders,
) (*v2.MsgBatchCancelDerivativeOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	gasConsumedBefore := ctx.GasMeter().GasConsumed()

	// todo: remove after QA
	defer func() {
		// no need to do anything here with gas meter, since it's handled per MsgCancelDerivativeOrder call
		totalGas := ctx.GasMeter().GasConsumed()
		k.Logger(ctx).Info("MsgBatchCancelDerivativeOrders",
			"gas_ante", gasConsumedBefore,
			"gas_msg", totalGas-gasConsumedBefore,
			"gas_total", totalGas,
			"sender", msg.Sender,
		)
	}()

	successes := make([]bool, len(msg.Data))
	for idx := range msg.Data {
		if _, err := k.CancelDerivativeOrder(goCtx, &v2.MsgCancelDerivativeOrder{
			Sender:       msg.Sender,
			MarketId:     msg.Data[idx].MarketId,
			SubaccountId: msg.Data[idx].SubaccountId,
			OrderHash:    msg.Data[idx].OrderHash,
			OrderMask:    msg.Data[idx].OrderMask,
			Cid:          msg.Data[idx].Cid,
		}); err != nil {
			metrics.ReportFuncError(k.svcTags)
		} else {
			successes[idx] = true
		}
	}

	return &v2.MsgBatchCancelDerivativeOrdersResponse{
		Success: successes,
	}, nil
}

func (k DerivativesMsgServer) IncreasePositionMargin(
	goCtx context.Context, msg *v2.MsgIncreasePositionMargin,
) (*v2.MsgIncreasePositionMarginResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgIncreasePositionMargin")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("IncreasePositionMargin",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	var (
		sender                  = sdk.MustAccAddressFromBech32(msg.Sender)
		sourceSubaccountID      = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
		destinationSubaccountID = common.HexToHash(msg.DestinationSubaccountId)
		marketID                = common.HexToHash(msg.MarketId)
	)

	market := k.GetDerivativeMarket(ctx, marketID, true)
	if market == nil {
		k.Logger(ctx).Error("active derivative market doesn't exist", "marketId", marketID)
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", marketID.Hex())
	}

	chainFormatAmount := market.NotionalToChainFormat(msg.Amount)
	chainFormatMarginIncrement, err := k.DecrementDepositOrChargeFromBank(ctx, sourceSubaccountID, market.QuoteDenom, chainFormatAmount)
	if err != nil {
		return nil, err
	}

	k.IncrementMarketBalance(ctx, marketID, chainFormatMarginIncrement)
	marginIncrement := market.NotionalFromChainFormat(chainFormatMarginIncrement)

	position := k.GetPosition(ctx, marketID, destinationSubaccountID)
	if position == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrPositionNotFound.Wrapf("subaccountID %s marketID %s", destinationSubaccountID.Hex(), marketID.Hex())
	}

	position.Margin = position.Margin.Add(marginIncrement)
	k.SetPosition(ctx, marketID, destinationSubaccountID, position)

	return &v2.MsgIncreasePositionMarginResponse{}, nil
}

func (k DerivativesMsgServer) DecreasePositionMargin(
	goCtx context.Context, msg *v2.MsgDecreasePositionMargin,
) (*v2.MsgDecreasePositionMarginResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgDecreasePositionMargin")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("DecreasePositionMargin",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	var (
		sender                  = sdk.MustAccAddressFromBech32(msg.Sender)
		sourceSubaccountID      = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
		destinationSubaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.DestinationSubaccountId)
		marketID                = common.HexToHash(msg.MarketId)
	)

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
	if market == nil || markPrice.IsNil() {
		k.Logger(ctx).Error(
			"active derivative market with valid mark price doesn't exist",
			"marketId", msg.MarketId,
			"mark price", markPrice.String(),
		)
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", marketID.Hex())
	}

	hasAllowedOracleType := k.isMarginDecreaseEnabledForOracle(market.OracleType)
	if !hasAllowedOracleType {
		return nil, types.ErrUnsupportedOracleType.Wrapf("margin withdrawal for %s oracle not supported", market.OracleType.String())
	}

	pricePairState := k.OracleKeeper.GetPricePairState(ctx, market.OracleType, market.OracleBase, market.OracleQuote, nil)
	if pricePairState == nil {
		return nil, types.ErrInvalidOracle.Wrapf(
			"type %s base %s quote %s", market.OracleType.String(), market.OracleBase, market.OracleQuote,
		)
	}

	currTime := ctx.BlockTime().Unix()

	params := k.GetParams(ctx)
	maxDelayThreshold := params.MarginDecreasePriceTimestampThresholdSeconds

	// enforce freshness of price
	exceedsDelay := (currTime-pricePairState.BaseTimestamp > maxDelayThreshold) ||
		(currTime-pricePairState.QuoteTimestamp > maxDelayThreshold)
	if exceedsDelay {
		return nil, types.ErrStaleOraclePrice.Wrapf(
			"price timestamp (base %d quote %d) vs curr time %d exceeds max delay threshold %d",
			pricePairState.BaseTimestamp,
			pricePairState.QuoteTimestamp,
			currTime,
			maxDelayThreshold,
		)
	}

	position := k.GetPosition(ctx, marketID, sourceSubaccountID)
	if position == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrPositionNotFound.Wrapf("subaccountID %s marketID %s", sourceSubaccountID.Hex(), marketID.Hex())
	}

	if market.IsPerpetual {
		funding := k.GetPerpetualMarketFunding(ctx, marketID)
		position.ApplyFunding(funding)
	}

	position.Margin = position.Margin.Sub(msg.Amount)

	// check initial margin requirements
	notional := position.EntryPrice.Mul(position.Quantity)

	// Enforce that Margin ≥ ReduceMarginRatio * Price * Quantity
	if position.Margin.LT(market.ReduceMarginRatio.Mul(notional)) {
		return nil, types.ErrInsufficientMargin
	}

	// For Longs: MarkPrice ≥ (Margin - Price * Quantity) / ((ReduceMarginRatio - 1) * Quantity)
	// For Shorts: MarkPrice ≤ (Margin + Price * Quantity) / ((1 + ReduceMarginRatio) * Quantity)
	markPriceThreshold := types.ComputeMarkPriceThreshold(
		position.IsLong, position.EntryPrice, position.Quantity, position.Margin, market.ReduceMarginRatio,
	)
	if err := types.CheckInitialMarginMarkPriceRequirement(position.IsLong, markPriceThreshold, markPrice); err != nil {
		return nil, err
	}

	marketBalance := k.GetMarketBalance(ctx, marketID)
	chainFormatMarginDecrease := market.NotionalToChainFormat(msg.Amount)
	if marketBalance.LT(chainFormatMarginDecrease) {
		return nil, types.ErrInsufficientMarketBalance
	}
	k.DecrementMarketBalance(ctx, marketID, chainFormatMarginDecrease)

	k.SetPosition(ctx, marketID, sourceSubaccountID, position)
	k.IncrementDepositOrSendToBank(ctx, destinationSubaccountID, market.QuoteDenom, chainFormatMarginDecrease)
	return &v2.MsgDecreasePositionMarginResponse{}, nil
}

func (k DerivativesMsgServer) LaunchPerpetualMarket(
	goCtx context.Context, msg *v2.MsgPerpetualMarketLaunch,
) (*v2.MsgPerpetualMarketLaunchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, sdkerrors.ErrUnauthorized
	}

	if err := k.handlePerpetualMarketLaunchProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgPerpetualMarketLaunchResponse{}, nil
}

func (k DerivativesMsgServer) LaunchExpiryFuturesMarket(
	goCtx context.Context, msg *v2.MsgExpiryFuturesMarketLaunch,
) (*v2.MsgExpiryFuturesMarketLaunchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, sdkerrors.ErrUnauthorized
	}

	if err := k.handleExpiryFuturesMarketLaunchProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgExpiryFuturesMarketLaunchResponse{}, nil
}

func (k DerivativesMsgServer) DerivativeMarketParamUpdate(
	goCtx context.Context, msg *v2.MsgDerivativeMarketParamUpdate,
) (*v2.MsgDerivativeMarketParamUpdateResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgDerivativeMarketParamUpdate")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("DerivativeMarketParamUpdate",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, sdkerrors.ErrUnauthorized
	}

	if err := k.handleDerivativeMarketParamUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgDerivativeMarketParamUpdateResponse{}, nil
}

func (DerivativesMsgServer) isMarginDecreaseEnabledForOracle(oracleType oracletypes.OracleType) bool {
	_, found := enabledOracleTypes[oracleType]
	return found
}
