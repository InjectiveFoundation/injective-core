package keeper

import (
	"context"
	"fmt"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type SpotMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewSpotMsgServerImpl returns an implementation of the bank MsgServer interface for the provided Keeper for spot market functions.
func NewSpotMsgServerImpl(keeper Keeper) SpotMsgServer {
	return SpotMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "spot_msg_h",
		},
	}
}

func (k SpotMsgServer) InstantSpotMarketLaunch(goCtx context.Context, msg *types.MsgInstantSpotMarketLaunch) (*types.MsgInstantSpotMarketLaunchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	// check if the market launch proposal already exists
	marketID := types.NewSpotMarketID(msg.BaseDenom, msg.QuoteDenom)
	if k.checkIfMarketLaunchProposalExist(ctx, types.ProposalTypeSpotMarketLaunch, marketID) {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("the spot market launch proposal already exists: marketID=%s", marketID.Hex())
		return nil, sdkerrors.Wrapf(types.ErrMarketLaunchProposalAlreadyExists, "the spot market launch proposal already exists: marketID=%s", marketID.Hex())
	}

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	_, err := k.SpotMarketLaunch(ctx, msg.Ticker, msg.BaseDenom, msg.QuoteDenom, msg.MinPriceTickSize, msg.MinQuantityTickSize)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching spot market", err)
		return nil, err
	}

	fee := k.GetParams(ctx).SpotMarketInstantListingFee
	err = k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("failed launching spot market", err)
		return nil, err
	}

	return &types.MsgInstantSpotMarketLaunchResponse{}, nil
}

func (k SpotMsgServer) CreateSpotLimitOrder(goCtx context.Context, msg *types.MsgCreateSpotLimitOrder) (*types.MsgCreateSpotLimitOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	account, _ := sdk.AccAddressFromBech32(msg.Sender)

	orderHash, err := k.createSpotLimitOrder(ctx, account, &msg.Order, nil)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreateSpotLimitOrderResponse{
		OrderHash: orderHash.Hex(),
	}, nil
}

func (k *Keeper) createSpotLimitOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order *types.SpotOrder,
	market *types.SpotMarket,
) (hash common.Hash, err error) {

	marketID := common.HexToHash(order.MarketId)

	// 0. Derive the subaccountID and populate the order with it
	subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, order.OrderInfo.SubaccountId)

	// set the actual subaccountID value in the order, since it might be a nonce value
	order.OrderInfo.SubaccountId = subaccountID.Hex()

	// 1. Check and increment Subaccount Nonce, Compute Order Hash
	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, subaccountID)
	orderHash, err := order.ComputeOrderHash(subaccountNonce.Nonce)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, err
	}

	// 2. Reject if spot market id does not reference an active spot market
	if market == nil {
		market = k.GetSpotMarket(ctx, marketID, true)
		if market == nil {
			k.Logger(ctx).Error("active spot market doesn't exist", "marketId", order.MarketId)
			metrics.ReportFuncError(k.svcTags)
			return orderHash, sdkerrors.Wrapf(types.ErrSpotMarketNotFound, "active spot market doesn't exist %s", order.MarketId)
		}
	}

	if err := order.CheckTickSize(market.MinPriceTickSize, market.MinQuantityTickSize); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, err
	}

	if order.OrderType.IsPostOnly() && k.SpotOrderCrossesTopOfBook(ctx, order) {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, types.ErrExceedsTopOfBookPrice
	}

	// 3. Reject if the subaccount's available deposits does not have at least the required funds for the trade
	balanceHoldIncrement, marginDenom := order.GetBalanceHoldAndMarginDenom(market)

	// 4. Decrement the available balance or bank by the funds amount needed to fund the order
	if err := k.chargeAccount(ctx, subaccountID, marginDenom, balanceHoldIncrement); err != nil {
		return orderHash, err
	}

	// 5. If Post Only, add the order to the resting orderbook
	//    Otherwise store the order in the transient limit order store and transient market indicator store
	spotLimitOrder := order.GetNewSpotLimitOrder(sender, orderHash)

	// 4. store the order in the conditional spot limit order store
	if order.IsConditional() {
		markPrice := k.GetSpotMidPriceOrBestPrice(ctx, marketID)
		if markPrice == nil {
			return orderHash, types.ErrInvalidMarketStatus.Wrapf("Mid or Best price for market: %v doesn't exist", marketID)
		}
		k.SetConditionalSpotLimitOrder(ctx, spotLimitOrder, marketID, *markPrice)
		return orderHash, nil
	}

	if order.OrderType.IsPostOnly() {
		k.SetNewSpotLimitOrder(ctx, spotLimitOrder, marketID, spotLimitOrder.IsBuy(), spotLimitOrder.Hash())

		var (
			buyOrders  = make([]*types.SpotLimitOrder, 0)
			sellOrders = make([]*types.SpotLimitOrder, 0)
		)
		if order.IsBuy() {
			buyOrders = append(buyOrders, spotLimitOrder)
		} else {
			sellOrders = append(sellOrders, spotLimitOrder)
		}

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventNewSpotOrders{
			MarketId:   marketID.Hex(),
			BuyOrders:  buyOrders,
			SellOrders: sellOrders,
		})
	} else {
		k.SetTransientSpotLimitOrder(ctx, spotLimitOrder, marketID, order.IsBuy(), orderHash)
		k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)
	}

	return orderHash, nil
}

func (k SpotMsgServer) CreateSpotMarketOrder(goCtx context.Context, msg *types.MsgCreateSpotMarketOrder) (*types.MsgCreateSpotMarketOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	var (
		marketID     = common.HexToHash(msg.Order.MarketId)
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.Order.OrderInfo.SubaccountId)
	)

	// populate the order with the actual subaccountID value, since it might be a nonce value
	msg.Order.OrderInfo.SubaccountId = subaccountID.Hex()

	// 1a. Reject if spot market id does not reference an active spot market
	market := k.GetSpotMarket(ctx, marketID, true)
	if market == nil {
		k.Logger(ctx).Error("active spot market doesn't exist", "marketId", msg.Order.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.Wrapf(types.ErrSpotMarketNotFound, "active spot market doesn't exist %s", msg.Order.MarketId)
	}

	if err := msg.Order.CheckTickSize(market.MinPriceTickSize, market.MinQuantityTickSize); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// 1b. Check access level if order type is atomic
	isAtomic := msg.Order.OrderType.IsAtomic()
	if isAtomic {
		err := k.ensureValidAccessLevelForAtomicExecution(ctx, sender)
		if err != nil {
			return nil, err
		}
	}

	// 2. Check and increment Subaccount Nonce, Compute Order Hash
	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, subaccountID)
	orderHash, err := msg.Order.ComputeOrderHash(subaccountNonce.Nonce)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	marginDenom := msg.Order.GetMarginDenom(market)

	// 3. Check the order crosses TOB
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

	// 4. Check available balance to fund the market order factoring in fee discounts, based on the worst acceptable price for the market order
	feeRate := market.TakerFeeRate
	if msg.Order.OrderType.IsAtomic() {
		feeRate = feeRate.Mul(k.Keeper.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, types.MarketType_Spot))
	}

	balanceHold := msg.Order.GetMarketOrderBalanceHold(feeRate, *bestPrice)

	// 5. Decrement deposit's AvailableBalance by the balance hold
	if err := k.chargeAccount(ctx, subaccountID, marginDenom, balanceHold); err != nil {
		return nil, err
	}

	marketOrder := msg.Order.ToSpotMarketOrder(sender, balanceHold, orderHash)

	var marketOrderResults *types.SpotMarketOrderResults
	if isAtomic {
		marketOrderResults = k.ExecuteAtomicSpotMarketOrder(ctx, market, marketOrder, feeRate)
	} else {
		// 6. Store the order in the transient spot market order store and transient market indicator store
		k.SetTransientSpotMarketOrder(ctx, marketOrder, &msg.Order, orderHash)
	}

	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)

	response := &types.MsgCreateSpotMarketOrderResponse{
		OrderHash: orderHash.Hex(),
	}

	if marketOrderResults != nil {
		response.Results = marketOrderResults
	}
	return response, nil
}

func (k SpotMsgServer) BatchCreateSpotLimitOrders(goCtx context.Context, msg *types.MsgBatchCreateSpotLimitOrders) (*types.MsgBatchCreateSpotLimitOrdersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// Naive, unoptimized implementation
	orderHashes := make([]string, len(msg.Orders))

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	orderFailEvent := types.EventOrderFail{
		Account: sender.Bytes(),
		Hashes:  make([][]byte, 0),
		Flags:   make([]uint32, 0),
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	for idx := range msg.Orders {
		if orderHash, err := k.createSpotLimitOrder(ctx, sender, &msg.Orders[idx], nil); err != nil {
			metrics.ReportFuncError(k.svcTags)
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				orderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, sdkerror.ABCICode())
			}
		} else {
			orderHashes[idx] = orderHash.Hex()
		}
	}
	if !orderFailEvent.IsEmpty() {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&orderFailEvent)
	}

	return &types.MsgBatchCreateSpotLimitOrdersResponse{
		OrderHashes: orderHashes,
	}, nil
}

func (k SpotMsgServer) CancelSpotOrder(goCtx context.Context, msg *types.MsgCancelSpotOrder) (*types.MsgCancelSpotOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	var (
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SubaccountId)
		marketID     = common.HexToHash(msg.MarketId)
		orderHash    = common.HexToHash(msg.OrderHash)
	)

	// Reject if spot market id does not reference an active, suspended or demolished spot market
	market := k.GetSpotMarketByID(ctx, marketID)
	err := k.cancelSpotLimitOrder(ctx, subaccountID, orderHash, market, marketID)
	return &types.MsgCancelSpotOrderResponse{}, err
}

func (k *Keeper) cancelSpotLimitOrder(
	ctx sdk.Context,
	subaccountID common.Hash,
	orderHash common.Hash,
	market *types.SpotMarket,
	marketID common.Hash,
) (err error) {

	if market == nil || !market.StatusSupportsOrderCancellations() {
		k.Logger(ctx).Error("active spot market doesn't exist")
		metrics.ReportFuncError(k.svcTags)
		return sdkerrors.Wrapf(types.ErrSpotMarketNotFound, "active spot market doesn't exist %s", marketID.Hex())
	}

	order := k.GetSpotLimitOrderBySubaccountID(ctx, marketID, nil, subaccountID, orderHash)
	var isTransient bool
	if order == nil {
		order = k.GetTransientSpotLimitOrderBySubaccountID(ctx, marketID, nil, subaccountID, orderHash)
		if order == nil {
			return sdkerrors.Wrap(types.ErrOrderDoesntExist, "Spot Limit Order is nil")
		}
		isTransient = true
	}

	if isTransient {
		k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, order)
	} else {
		k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, order.IsBuy(), order)
	}
	return nil
}

func (k SpotMsgServer) BatchCancelSpotOrders(goCtx context.Context, msg *types.MsgBatchCancelSpotOrders) (*types.MsgBatchCancelSpotOrdersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// Naive, unoptimized implementation
	successes := make([]bool, len(msg.Data))
	for idx := range msg.Data {
		if _, err := k.CancelSpotOrder(goCtx, &types.MsgCancelSpotOrder{
			Sender:       msg.Sender,
			MarketId:     msg.Data[idx].MarketId,
			SubaccountId: msg.Data[idx].SubaccountId,
			OrderHash:    msg.Data[idx].OrderHash,
		}); err != nil {
			metrics.ReportFuncError(k.svcTags)
		} else {
			successes[idx] = true
		}
	}

	return &types.MsgBatchCancelSpotOrdersResponse{
		Success: successes,
	}, nil
}
