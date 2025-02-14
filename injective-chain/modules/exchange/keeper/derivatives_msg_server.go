package keeper

import (
	"context"
	"errors"
	"fmt"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	cosmoserrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type DerivativesMsgServer struct {
	Keeper
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

// NewDerivativesMsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper for derivatives market functions.
func NewDerivativesMsgServerImpl(keeper Keeper) DerivativesMsgServer {
	return DerivativesMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "dvt_msg_h",
		},
	}
}

func (k DerivativesMsgServer) InstantPerpetualMarketLaunch(goCtx context.Context, msg *types.MsgInstantPerpetualMarketLaunch) (*types.MsgInstantPerpetualMarketLaunchResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	isRegistrationAllowed := k.isAdmin(ctx, msg.Sender)

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
	if k.checkIfMarketLaunchProposalExist(ctx, types.ProposalTypePerpetualMarketLaunch, marketID) {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("the perpetual market launch proposal already exists: marketID=%s", marketID.Hex())
		return nil, types.ErrMarketLaunchProposalAlreadyExists.Wrapf("the perpetual market launch proposal already exists: marketID=%s", marketID.Hex())
	}

	fee := k.GetParams(ctx).DerivativeMarketInstantListingFee
	err := k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	_, _, err = k.PerpetualMarketLaunch(
		ctx, msg.Ticker, msg.QuoteDenom, msg.OracleBase, msg.OracleQuote, msg.OracleScaleFactor, msg.OracleType,
		msg.InitialMarginRatio, msg.MaintenanceMarginRatio,
		msg.MakerFeeRate, msg.TakerFeeRate, msg.MinPriceTickSize, msg.MinQuantityTickSize, msg.MinNotional,
	)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	return &types.MsgInstantPerpetualMarketLaunchResponse{}, err
}

func (k DerivativesMsgServer) InstantExpiryFuturesMarketLaunch(goCtx context.Context, msg *types.MsgInstantExpiryFuturesMarketLaunch) (*types.MsgInstantExpiryFuturesMarketLaunchResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	isRegistrationAllowed := k.isAdmin(ctx, msg.Sender)

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
	if k.checkIfMarketLaunchProposalExist(ctx, types.ProposalTypeExpiryFuturesMarketLaunch, marketID) {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("the expiry futures market launch proposal already exists: marketID=%s", marketID.Hex())
		return nil, types.ErrMarketLaunchProposalAlreadyExists.Wrapf("the expiry futures market launch proposal already exists: marketID=%s", marketID.Hex())
	}

	fee := k.GetParams(ctx).DerivativeMarketInstantListingFee
	err := k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	if _, _, err := k.ExpiryFuturesMarketLaunch(
		ctx, msg.Ticker, msg.QuoteDenom,
		msg.OracleBase, msg.OracleQuote, msg.OracleScaleFactor, msg.OracleType, msg.Expiry,
		msg.InitialMarginRatio, msg.MaintenanceMarginRatio,
		msg.MakerFeeRate, msg.TakerFeeRate, msg.MinPriceTickSize, msg.MinQuantityTickSize, msg.MinNotional,
	); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching derivative market", err)
		return nil, err
	}

	return &types.MsgInstantExpiryFuturesMarketLaunchResponse{}, err
}

func (k DerivativesMsgServer) CreateDerivativeLimitOrder(goCtx context.Context, msg *types.MsgCreateDerivativeLimitOrder) (*types.MsgCreateDerivativeLimitOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	account, _ := sdk.AccAddressFromBech32(msg.Sender)

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, msg.Order.MarketID(), true)
	if market == nil || markPrice.IsNil() {
		k.Logger(ctx).Error("active derivative market with valid mark price doesn't exist", "marketId", msg.Order.MarketId, "mark price", markPrice.String())
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", msg.Order.MarketId)
	}

	orderHash, err := k.createDerivativeLimitOrder(ctx, account, &msg.Order, market, markPrice)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	return &types.MsgCreateDerivativeLimitOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}, nil
}

func (k *Keeper) createDerivativeLimitOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order *types.DerivativeOrder,
	market DerivativeMarketI,
	markPrice math.LegacyDec,
) (hash common.Hash, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, order.OrderInfo.SubaccountId)

	// set the actual subaccountID value in the order, since it might be a nonce value
	order.OrderInfo.SubaccountId = subaccountID.Hex()

	marketID := order.MarketID()

	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())

	isMaker := order.OrderType.IsPostOnly()

	orderHash, err := k.ensureValidDerivativeOrder(ctx, order, market, metadata, markPrice, false, nil, isMaker)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, err
	}

	derivativeLimitOrder := types.NewDerivativeLimitOrder(order, sender, orderHash)

	// Store the order in the conditionals store -or- transient limit order store and transient market indicator store
	if order.IsConditional() {
		// store the order in the conditional derivative market order store
		k.SetConditionalDerivativeLimitOrderWithMetadata(ctx, derivativeLimitOrder, metadata, marketID, markPrice)
		return orderHash, nil
	}

	if order.OrderType.IsPostOnly() {
		k.SetPostOnlyDerivativeLimitOrderWithMetadata(ctx, derivativeLimitOrder, metadata, marketID)
		return orderHash, nil
	}

	k.SetNewTransientDerivativeLimitOrderWithMetadata(ctx, derivativeLimitOrder, metadata, marketID, derivativeLimitOrder.IsBuy(), orderHash)
	k.SetTransientSubaccountLimitOrderIndicator(ctx, marketID, subaccountID)
	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)
	return orderHash, nil
}

func (k DerivativesMsgServer) BatchCreateDerivativeLimitOrders(goCtx context.Context, msg *types.MsgBatchCreateDerivativeLimitOrders) (*types.MsgBatchCreateDerivativeLimitOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	orderFailEvent := types.EventOrderFail{
		Account: sender.Bytes(),
		Hashes:  make([][]byte, 0),
		Flags:   make([]uint32, 0),
		Cids:    make([]string, 0),
	}

	marketsCache := make(map[common.Hash]*types.FullDerivativeMarket)
	orderHashes := make([]string, len(msg.Orders))
	createdOrdersCids := make([]string, 0)
	failedOrdersCids := make([]string, 0)

	for idx := range msg.Orders {
		order := msg.Orders[idx]
		marketID := order.MarketID()

		fullMarket, ok := marketsCache[marketID]
		if !ok {
			market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)

			// edge case when active market doesn't exist
			if market == nil || markPrice.IsNil() {
				orderHashes[idx] = fmt.Sprintf("%d", types.ErrDerivativeMarketNotFound.ABCICode())
				continue
			}

			fullMarket = &types.FullDerivativeMarket{Market: market, MarkPrice: markPrice}
			marketsCache[marketID] = fullMarket
		}

		if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, &order, fullMarket.Market, fullMarket.MarkPrice); err != nil {
			metrics.ReportFuncError(k.svcTags)
			sdkerror := &cosmoserrors.Error{}
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
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&orderFailEvent)
	}

	return &types.MsgBatchCreateDerivativeLimitOrdersResponse{
		OrderHashes:       orderHashes,
		CreatedOrdersCids: createdOrdersCids,
		FailedOrdersCids:  failedOrdersCids,
	}, nil
}

func (k *Keeper) createDerivativeMarketOrder(ctx sdk.Context, sender sdk.AccAddress, derivativeOrder *types.DerivativeOrder, market DerivativeMarketI, markPrice math.LegacyDec) (orderHash common.Hash, results *types.DerivativeMarketOrderResults, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	var (
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, derivativeOrder.OrderInfo.SubaccountId)
		marketID     = derivativeOrder.MarketID()
	)

	// set the actual subaccountID value in the order, since it might be a nonce value
	derivativeOrder.OrderInfo.SubaccountId = subaccountID.Hex()

	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, derivativeOrder.IsBuy())

	var orderMarginHold math.LegacyDec
	orderHash, err = k.ensureValidDerivativeOrder(ctx, derivativeOrder, market, metadata, markPrice, true, &orderMarginHold, false)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, nil, err
	}

	if derivativeOrder.OrderType.IsAtomic() {
		err = k.ensureValidAccessLevelForAtomicExecution(ctx, sender)
		if err != nil {
			return orderHash, nil, err
		}
	}

	marketOrder := types.NewDerivativeMarketOrder(derivativeOrder, sender, orderHash)

	// 4. Check Order/Position Margin amount
	if marketOrder.IsVanilla() {
		// Check available balance to fund the market order
		marketOrder.MarginHold = orderMarginHold
	}

	if derivativeOrder.IsConditional() {
		k.SetConditionalDerivativeMarketOrderWithMetadata(ctx, marketOrder, metadata, marketID, markPrice)
		return orderHash, nil, nil
	}

	if derivativeOrder.OrderType.IsAtomic() {
		var funding *types.PerpetualMarketFunding
		if market.GetIsPerpetual() {
			funding = k.GetPerpetualMarketFunding(ctx, marketID)
		}
		positionStates := NewPositionStates()

		var isMarketSolvent bool

		results, isMarketSolvent, err = k.ExecuteDerivativeMarketOrderImmediately(ctx, market, markPrice, funding, marketOrder, positionStates, false)
		if err != nil {
			return orderHash, nil, err
		}

		if !isMarketSolvent {
			return orderHash, nil, types.ErrInsufficientMarketBalance
		}
	} else {
		// 5. Store the order in the transient derivative market order store and transient market indicator store
		k.SetTransientDerivativeMarketOrder(ctx, marketOrder, derivativeOrder, orderHash)
		k.SetTransientSubaccountMarketOrderIndicator(ctx, marketID, subaccountID)
	}
	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)
	return orderHash, results, nil
}

func (k DerivativesMsgServer) CreateDerivativeMarketOrder(goCtx context.Context, msg *types.MsgCreateDerivativeMarketOrder) (*types.MsgCreateDerivativeMarketOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

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

	resp := &types.MsgCreateDerivativeMarketOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}
	if results != nil {
		resp.Results = results
	}
	return resp, nil
}

func (k DerivativesMsgServer) CancelDerivativeOrder(goCtx context.Context, msg *types.MsgCancelDerivativeOrder) (*types.MsgCancelDerivativeOrderResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	var (
		marketID     = common.HexToHash(msg.MarketId)
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SubaccountId)
		identifier   = types.GetOrderIdentifier(msg.OrderHash, msg.Cid)
	)

	market := k.GetDerivativeMarketByID(ctx, marketID)

	err := k.cancelDerivativeOrder(ctx, subaccountID, identifier, market, marketID, msg.OrderMask)

	if err != nil {
		_ = ctx.EventManager().EmitTypedEvent(types.NewEventOrderCancelFail(marketID, subaccountID, msg.OrderHash, msg.Cid, err))
		return nil, err
	}

	return &types.MsgCancelDerivativeOrderResponse{}, nil
}

func (k *Keeper) cancelDerivativeOrder(
	ctx sdk.Context,
	subaccountID common.Hash,
	identifier any,
	market DerivativeMarketI,
	marketID common.Hash,
	orderMask int32,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderHash, err := k.getOrderHashFromIdentifier(ctx, subaccountID, identifier)
	if err != nil {
		return err
	}

	return k.cancelDerivativeOrderByOrderHash(ctx, subaccountID, orderHash, market, marketID, orderMask)
}

func (k *Keeper) cancelDerivativeOrderByOrderHash(
	ctx sdk.Context,
	subaccountID common.Hash,
	orderHash common.Hash,
	market DerivativeMarketI,
	marketID common.Hash,
	orderMask int32,
) (err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	// Reject if derivative market id does not reference an active derivative market
	if market == nil || !market.StatusSupportsOrderCancellations() {
		k.Logger(ctx).Debug("active derivative market doesn't exist", "marketID", marketID)
		metrics.ReportFuncError(k.svcTags)
		return types.ErrDerivativeMarketNotFound.Wrapf("active derivative market doesn't exist %s", marketID.Hex())
	}

	var (
		isBuy                    *bool // nil by default
		shouldCheckIsBuy         = orderMask&int32(types.OrderMask_BUY_OR_HIGHER) > 0
		shouldCheckIsSell        = orderMask&int32(types.OrderMask_SELL_OR_LOWER) > 0
		shouldCheckIsRegular     = orderMask&int32(types.OrderMask_REGULAR) > 0
		shouldCheckIsConditional = orderMask&int32(types.OrderMask_CONDITIONAL) > 0
		shouldCheckIsMarketOrder = orderMask&int32(types.OrderMask_MARKET) > 0
		shouldCheckIsLimitOrder  = orderMask&int32(types.OrderMask_LIMIT) > 0
	)

	areRegularAndConditionalFlagsBothUnspecified := !shouldCheckIsRegular && !shouldCheckIsConditional
	areBuyAndSellFlagsBothUnspecified := !shouldCheckIsBuy && !shouldCheckIsSell
	areMarketAndLimitFlagsBothUnspecified := !shouldCheckIsMarketOrder && !shouldCheckIsLimitOrder

	// if both conditional flags are unspecified, check both
	if areRegularAndConditionalFlagsBothUnspecified {
		shouldCheckIsRegular = true
		shouldCheckIsConditional = true
	}

	// if both market and limit flags are unspecified, check both
	if areMarketAndLimitFlagsBothUnspecified {
		shouldCheckIsMarketOrder = true
		shouldCheckIsLimitOrder = true
	}

	// if both buy/sell flags are unspecified, check both
	if areBuyAndSellFlagsBothUnspecified {
		shouldCheckIsBuy = true
		shouldCheckIsSell = true
	}

	isBuyOrSellFlagExplicitlySet := !(shouldCheckIsBuy && shouldCheckIsSell)

	// if the buy flag is explicitly set, check it
	if isBuyOrSellFlagExplicitlySet {
		isBuy = &shouldCheckIsBuy
	}

	if shouldCheckIsRegular {
		var isTransient = false

		order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)

		if order == nil {
			order = k.GetTransientDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
			if order == nil && !shouldCheckIsConditional {
				return types.ErrOrderDoesntExist.Wrap("Derivative Limit Order doesn't exist")
			}
			isTransient = true
		}

		if order != nil {
			if isTransient {
				err = k.CancelTransientDerivativeLimitOrder(ctx, market, order)
			} else {
				direction := order.OrderType.IsBuy()
				err = k.CancelRestingDerivativeLimitOrder(ctx, market, subaccountID, &direction, orderHash, true, true)
			}
			return err
		}
	}
	if shouldCheckIsConditional {
		// isBuy == isHigher
		if shouldCheckIsMarketOrder {
			order, direction := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
			if order != nil {
				err = k.CancelConditionalDerivativeMarketOrder(ctx, market, subaccountID, &direction, orderHash)
				return err
			}

			if !shouldCheckIsLimitOrder {
				return types.ErrOrderDoesntExist.Wrap("Derivative Market Order doesn't exist")
			}
		}
		if shouldCheckIsLimitOrder {
			order, direction := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
			if order == nil {
				return types.ErrOrderDoesntExist.Wrap("Derivative Limit Order doesn't exist")
			}
			err = k.CancelConditionalDerivativeLimitOrder(ctx, market, subaccountID, &direction, orderHash)
			return err
		}
	}
	return err
}

func (k DerivativesMsgServer) BatchCancelDerivativeOrders(goCtx context.Context, msg *types.MsgBatchCancelDerivativeOrders) (*types.MsgBatchCancelDerivativeOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	successes := make([]bool, len(msg.Data))
	for idx := range msg.Data {
		if _, err := k.CancelDerivativeOrder(goCtx, &types.MsgCancelDerivativeOrder{
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

	return &types.MsgBatchCancelDerivativeOrdersResponse{
		Success: successes,
	}, nil
}

func (k DerivativesMsgServer) IncreasePositionMargin(goCtx context.Context, msg *types.MsgIncreasePositionMargin) (*types.MsgIncreasePositionMarginResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

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

	marginIncrement, err := k.DecrementDepositOrChargeFromBank(ctx, sourceSubaccountID, market.QuoteDenom, msg.Amount)
	if err != nil {
		return nil, err
	}

	k.IncrementMarketBalance(ctx, marketID, marginIncrement)

	position := k.GetPosition(ctx, marketID, destinationSubaccountID)
	if position == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrPositionNotFound.Wrapf("subaccountID %s marketID %s", destinationSubaccountID.Hex(), marketID.Hex())
	}

	position.Margin = position.Margin.Add(marginIncrement)
	k.SetPosition(ctx, marketID, destinationSubaccountID, position)

	return &types.MsgIncreasePositionMarginResponse{}, nil
}

func (k DerivativesMsgServer) DecreasePositionMargin(goCtx context.Context, msg *types.MsgDecreasePositionMargin) (*types.MsgDecreasePositionMarginResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	var (
		sender                  = sdk.MustAccAddressFromBech32(msg.Sender)
		sourceSubaccountID      = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
		destinationSubaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.DestinationSubaccountId)
		marketID                = common.HexToHash(msg.MarketId)
	)

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
	if market == nil || markPrice.IsNil() {
		k.Logger(ctx).Error("active derivative market with valid mark price doesn't exist", "marketId", msg.MarketId, "mark price", markPrice.String())
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", marketID.Hex())
	}

	hasAllowedOracleType := k.isMarginDecreaseEnabledForOracle(market.OracleType)
	if !hasAllowedOracleType {
		return nil, types.ErrUnsupportedOracleType.Wrapf("margin withdrawal for %s oracle not supported", market.OracleType.String())
	}

	pricePairState := k.OracleKeeper.GetPricePairState(ctx, market.OracleType, market.OracleBase, market.OracleQuote, nil)
	if pricePairState == nil {
		return nil, types.ErrInvalidOracle.Wrapf("type %s base %s quote %s", market.OracleType.String(), market.OracleBase, market.OracleQuote)
	}

	currTime := ctx.BlockTime().Unix()

	params := k.GetParams(ctx)
	maxDelayThreshold := params.MarginDecreasePriceTimestampThresholdSeconds

	// enforce freshness of price
	exceedsDelay := (currTime-pricePairState.BaseTimestamp > maxDelayThreshold) || (currTime-pricePairState.QuoteTimestamp > maxDelayThreshold)
	if exceedsDelay {
		return nil, types.ErrStaleOraclePrice.Wrapf("price timestamp (base %d quote %d) vs curr time %d exceeds max delay threshold %d", pricePairState.BaseTimestamp, pricePairState.QuoteTimestamp, currTime, maxDelayThreshold)
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

	// Enforce that Margin ≥ InitialMarginRatio * Price * Quantity
	if position.Margin.LT(market.InitialMarginRatio.Mul(notional)) {
		return nil, types.ErrInsufficientMargin
	}

	// For Longs: MarkPrice ≥ (Margin - Price * Quantity) / ((InitialMarginRatio - 1) * Quantity)
	// For Shorts: MarkPrice ≤ (Margin + Price * Quantity) / ((1 + InitialMarginRatio) * Quantity)
	markPriceThreshold := types.ComputeMarkPriceThreshold(position.IsLong, position.EntryPrice, position.Quantity, position.Margin, market.InitialMarginRatio)
	if err := types.CheckInitialMarginMarkPriceRequirement(position.IsLong, markPriceThreshold, markPrice); err != nil {
		return nil, err
	}

	marketBalance := k.GetMarketBalance(ctx, marketID)
	if marketBalance.LT(msg.Amount) {
		return nil, types.ErrInsufficientMarketBalance
	}
	k.DecrementMarketBalance(ctx, marketID, msg.Amount)

	k.SetPosition(ctx, marketID, sourceSubaccountID, position)

	k.IncrementDepositOrSendToBank(ctx, destinationSubaccountID, market.QuoteDenom, msg.Amount)
	return &types.MsgDecreasePositionMarginResponse{}, nil
}

func (k DerivativesMsgServer) isMarginDecreaseEnabledForOracle(oracleType oracletypes.OracleType) bool {
	_, found := enabledOracleTypes[oracleType]
	return found
}
