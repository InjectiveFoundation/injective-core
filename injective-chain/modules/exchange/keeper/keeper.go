package keeper

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// Keeper of this module maintains collections of exchange.
type Keeper struct {
	storeKey  storetypes.StoreKey
	tStoreKey storetypes.StoreKey
	cdc       codec.BinaryCodec

	DistributionKeeper   distrkeeper.Keeper
	StakingKeeper        types.StakingKeeper
	AccountKeeper        authkeeper.AccountKeeper
	bankKeeper           bankkeeper.Keeper
	OracleKeeper         types.OracleKeeper
	insuranceKeeper      types.InsuranceKeeper
	govKeeper            govkeeper.Keeper
	wasmViewKeeper       types.WasmViewKeeper
	wasmxExecutionKeeper types.WasmxExecutionKeeper

	svcTags   metrics.Tags
	authority string

	// cached value from params (false by default)
	fixedGas bool
}

// NewKeeper creates new instances of the exchange Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	tstoreKey storetypes.StoreKey,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	ok types.OracleKeeper,
	ik types.InsuranceKeeper,
	dk distrkeeper.Keeper,
	sk types.StakingKeeper,
	authority string,
) *Keeper {
	return &Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		tStoreKey:          tstoreKey,
		AccountKeeper:      ak,
		OracleKeeper:       ok,
		DistributionKeeper: dk,
		StakingKeeper:      sk,
		bankKeeper:         bk,
		insuranceKeeper:    ik,
		authority:          authority,
		svcTags: metrics.Tags{
			"svc": "exchange_k",
		},
		fixedGas: false,
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) SetGovKeeper(gk govkeeper.Keeper) {
	k.govKeeper = gk
}

func (k *Keeper) SetWasmKeepers(
	wk wasmkeeper.Keeper,
	wxk types.WasmxExecutionKeeper,
) {
	k.wasmViewKeeper = types.WasmViewKeeper(wk)
	k.wasmxExecutionKeeper = wxk
}

// CreateModuleAccount creates a module account with minter and burning capabilities
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	//revive:disable:unchecked-type-assertion // we know the type is correct
	moduleAcc := (k.AccountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI)
	k.AccountKeeper.SetModuleAccount(ctx, moduleAcc)
}

func (k *Keeper) IsDenomDecimalsValid(ctx sdk.Context, tokenDenom string, tokenDecimals uint32) bool {
	tokenMetadata, found := k.bankKeeper.GetDenomMetaData(ctx, tokenDenom)
	return !found || tokenMetadata.Decimals == 0 || tokenMetadata.Decimals == tokenDecimals
}

func (k *Keeper) TokenDenomDecimals(ctx sdk.Context, tokenDenom string) (decimals uint32, err error) {
	tokenMetadata, found := k.bankKeeper.GetDenomMetaData(ctx, tokenDenom)
	if !found {
		return 0, sdkerrors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not have denom metadata", tokenDenom)
	}
	if tokenMetadata.Decimals == 0 {
		return 0, sdkerrors.Wrapf(types.ErrInvalidQuoteDenom, "denom units for %s are not correctly configured", tokenDenom)
	}

	return tokenMetadata.Decimals, nil
}

//revive:disable:argument-limit // We need all the parameters in the function
func (k *Keeper) ExecuteBatchUpdateOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	subaccountId string,
	spotMarketIDsToCancelAll []string,
	derivativeMarketIDsToCancelAll []string,
	binaryOptionsMarketIDsToCancelAll []string,
	spotOrdersToCancel []*v2.OrderData,
	derivativeOrdersToCancel []*v2.OrderData,
	binaryOptionsOrdersToCancel []*v2.OrderData,
	spotOrdersToCreate []*v2.SpotOrder,
	derivativeOrdersToCreate []*v2.DerivativeOrder,
	binaryOptionsOrdersToCreate []*v2.DerivativeOrder,
) (*v2.MsgBatchUpdateOrdersResponse, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		spotMarkets          = make(map[common.Hash]*v2.SpotMarket)
		derivativeMarkets    = make(map[common.Hash]*v2.DerivativeMarket)
		binaryOptionsMarkets = make(map[common.Hash]*v2.BinaryOptionsMarket)

		spotCancelSuccesses            = make([]bool, len(spotOrdersToCancel))
		derivativeCancelSuccesses      = make([]bool, len(derivativeOrdersToCancel))
		binaryOptionsCancelSuccesses   = make([]bool, len(binaryOptionsOrdersToCancel))
		spotOrderHashes                = make([]string, len(spotOrdersToCreate))
		createdSpotOrdersCids          = make([]string, 0)
		failedSpotOrdersCids           = make([]string, 0)
		derivativeOrderHashes          = make([]string, len(derivativeOrdersToCreate))
		createdDerivativeOrdersCids    = make([]string, 0)
		failedDerivativeOrdersCids     = make([]string, 0)
		binaryOptionsOrderHashes       = make([]string, len(binaryOptionsOrdersToCreate))
		createdBinaryOptionsOrdersCids = make([]string, 0)
		failedBinaryOptionsOrdersCids  = make([]string, 0)
	)

	//  Derive the subaccountID.
	subaccountIDForCancelAll := types.MustGetSubaccountIDOrDeriveFromNonce(sender, subaccountId)

	// NOTE: if the subaccountID is empty, subaccountIDForCancelAll will be the default subaccount, so we must check
	// that its initial value is not empty
	shouldExecuteCancelAlls := subaccountId != ""

	if shouldExecuteCancelAlls {
		k.processCancelAllSpotOrders(ctx, spotMarketIDsToCancelAll, subaccountIDForCancelAll, spotMarkets)
		k.processCancelAllDerivativeOrders(ctx, derivativeMarketIDsToCancelAll, subaccountIDForCancelAll, derivativeMarkets)
		k.processCancelAllBinaryOptionsOrders(ctx, binaryOptionsMarketIDsToCancelAll, subaccountIDForCancelAll, binaryOptionsMarkets)
	}

	k.processCancelSpotOrders(ctx, sender, spotOrdersToCancel, spotCancelSuccesses, spotMarkets)
	k.processCancelDerivativeOrders(ctx, sender, derivativeOrdersToCancel, derivativeCancelSuccesses, derivativeMarkets)
	k.processCancelBinaryOptionsOrders(ctx, sender, binaryOptionsOrdersToCancel, binaryOptionsCancelSuccesses, binaryOptionsMarkets)

	orderFailEvent := v2.EventOrderFail{
		Account: sender.Bytes(),
		Hashes:  make([][]byte, 0),
		Flags:   make([]uint32, 0),
		Cids:    make([]string, 0),
	}

	k.processCreateSpotOrders(
		ctx, sender, spotOrdersToCreate, spotOrderHashes, &orderFailEvent, &createdSpotOrdersCids, &failedSpotOrdersCids, spotMarkets,
	)

	markPrices := make(map[common.Hash]math.LegacyDec)
	k.processCreateDerivativeOrders(
		ctx,
		sender,
		derivativeOrdersToCreate,
		derivativeOrderHashes,
		&orderFailEvent,
		&createdDerivativeOrdersCids,
		&failedDerivativeOrdersCids,
		derivativeMarkets,
		markPrices,
	)
	k.processCreateBinaryOptionsOrders(
		ctx,
		sender,
		binaryOptionsOrdersToCreate,
		binaryOptionsOrderHashes,
		&orderFailEvent,
		&createdBinaryOptionsOrdersCids,
		&failedBinaryOptionsOrdersCids,
		binaryOptionsMarkets,
	)

	if !orderFailEvent.IsEmpty() {
		k.EmitEvent(ctx, &orderFailEvent)
	}

	return &v2.MsgBatchUpdateOrdersResponse{
		SpotCancelSuccess:              spotCancelSuccesses,
		DerivativeCancelSuccess:        derivativeCancelSuccesses,
		SpotOrderHashes:                spotOrderHashes,
		DerivativeOrderHashes:          derivativeOrderHashes,
		BinaryOptionsCancelSuccess:     binaryOptionsCancelSuccesses,
		BinaryOptionsOrderHashes:       binaryOptionsOrderHashes,
		CreatedSpotOrdersCids:          createdSpotOrdersCids,
		FailedSpotOrdersCids:           failedSpotOrdersCids,
		CreatedDerivativeOrdersCids:    createdDerivativeOrdersCids,
		FailedDerivativeOrdersCids:     failedDerivativeOrdersCids,
		CreatedBinaryOptionsOrdersCids: createdBinaryOptionsOrdersCids,
		FailedBinaryOptionsOrdersCids:  failedBinaryOptionsOrdersCids,
	}, nil
}

//nolint:revive // this is fine
func (k *Keeper) FixedGasBatchUpdateOrders(
	c context.Context,
	msg *v2.MsgBatchUpdateOrders,
) (*v2.MsgBatchUpdateOrdersResponse, error) {
	//	no clever method shadowing here

	cc, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(cc)
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	var (
		subaccountId         = msg.SubaccountId
		spotMarkets          = make(map[common.Hash]*v2.SpotMarket)
		derivativeMarkets    = make(map[common.Hash]*v2.DerivativeMarket)
		binaryOptionsMarkets = make(map[common.Hash]*v2.BinaryOptionsMarket)

		spotCancelSuccesses          = make([]bool, len(msg.SpotOrdersToCancel))
		derivativeCancelSuccesses    = make([]bool, len(msg.DerivativeOrdersToCancel))
		binaryOptionsCancelSuccesses = make([]bool, len(msg.BinaryOptionsOrdersToCancel))
		spotOrderHashes              = make([]string, len(msg.SpotOrdersToCreate))
		derivativeOrderHashes        = make([]string, len(msg.DerivativeOrdersToCreate))
		binaryOptionsOrderHashes     = make([]string, len(msg.BinaryOptionsOrdersToCreate))

		createdSpotOrdersCids          = make([]string, 0)
		failedSpotOrdersCids           = make([]string, 0)
		createdDerivativeOrdersCids    = make([]string, 0)
		failedDerivativeOrdersCids     = make([]string, 0)
		createdBinaryOptionsOrdersCids = make([]string, 0)
		failedBinaryOptionsOrdersCids  = make([]string, 0)
	)

	// reference the gas meter early to consume gas later on in loop iterations
	gasMeter := ctx.GasMeter()
	gasConsumedBefore := gasMeter.GasConsumed()

	defer func() {
		totalGas := gasMeter.GasConsumed()
		k.Logger(ctx).Info("MsgBatchUpdateOrders",
			"gas_ante", gasConsumedBefore,
			"gas_msg", totalGas-gasConsumedBefore,
			"gas_total", totalGas,
			"sender", msg.Sender,
		)
	}()

	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	/**	1. Cancel all **/
	// NOTE: provided subaccountID indicates cancelling all orders in a market for given market IDs
	if isCancelAll := subaccountId != ""; isCancelAll {
		//  Derive the subaccountID.
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, subaccountId)

		/**	1. a) Cancel all spot limit orders in markets **/
		for _, spotMarketIdToCancelAll := range msg.SpotMarketIdsToCancelAll {
			marketID := common.HexToHash(spotMarketIdToCancelAll)
			market := k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				continue
			}
			spotMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug("failed to cancel all spot limit orders", "marketID", marketID.Hex())
				continue
			}

			// k.CancelAllSpotLimitOrders(ctx, market, subaccountID, marketID)
			// get all orders to cancel
			var (
				restingBuyOrders = k.GetAllSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				restingSellOrders = k.GetAllSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					subaccountID,
				)
				transientBuyOrders = k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				transientSellOrders = k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					subaccountID,
				)
			)

			// consume gas
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(restingBuyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(restingSellOrders)), "")
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(transientBuyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(transientSellOrders)), "")

			// cancel orders
			for idx := range restingBuyOrders {
				k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, true, restingBuyOrders[idx])
			}

			for idx := range restingSellOrders {
				k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, false, restingSellOrders[idx])
			}

			for idx := range transientBuyOrders {
				k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, transientBuyOrders[idx])
			}

			for idx := range transientSellOrders {
				k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, transientSellOrders[idx])
			}
		}

		/**	1. b) Cancel all derivative limit orders in markets **/
		for _, derivativeMarketIdToCancelAll := range msg.DerivativeMarketIdsToCancelAll {
			marketID := common.HexToHash(derivativeMarketIdToCancelAll)
			market := k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel all derivative limit orders for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			derivativeMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug(
					"failed to cancel all derivative limit orders for market whose status doesnt support cancellations",
					"marketID",
					marketID.Hex(),
				)
				continue
			}

			var (
				restingBuyOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				restingSellOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false, subaccountID,
				)
				buyOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					true,
				)
				sellOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					false,
				)
				higherMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					true,
					subaccountID,
				)
				lowerMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					true,
					subaccountID,
				)
				higherLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					false,
					subaccountID,
				)
				lowerLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					false,
					subaccountID,
				)
			)

			// consume gas
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(restingBuyOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(restingSellOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(buyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(sellOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(higherMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(lowerMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(higherLimitOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(lowerLimitOrders)), "")

			for _, hash := range restingBuyOrderHashes {
				isBuy := true

				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range restingSellOrderHashes {
				isBuy := false
				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, buyOrder := range buyOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
					orderHash := common.BytesToHash(buyOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for buyOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					k.EmitEvent(ctx, v2.NewEventOrderCancelFail(
						marketID,
						subaccountID,
						orderHash.Hex(),
						buyOrder.Cid(),
						err,
					))
				}
			}

			for _, sellOrder := range sellOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
					orderHash := common.BytesToHash(sellOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for sellOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					k.EmitEvent(ctx, v2.NewEventOrderCancelFail(
						marketID,
						subaccountID,
						orderHash.Hex(),
						sellOrder.Cid(),
						err,
					))
				}
			}

			for _, hash := range higherMarketOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerMarketOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range higherLimitOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerLimitOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}
		}

		/**	1. c) Cancel all bo limit orders in markets **/
		for _, binaryOptionsMarketIdToCancelAll := range msg.BinaryOptionsMarketIdsToCancelAll {
			marketID := common.HexToHash(binaryOptionsMarketIdToCancelAll)
			market := k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel all binary options limit orders for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			binaryOptionsMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug(
					"failed to cancel all binary options limit orders for market whose status doesnt support cancellations",
					"marketID",
					marketID.Hex(),
				)
				continue
			}

			var (
				restingBuyOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				restingSellOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					subaccountID,
				)
				buyOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					true,
				)
				sellOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					false,
				)
				higherMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					true,
					subaccountID,
				)
				lowerMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					true,
					subaccountID,
				)
				higherLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					false,
					subaccountID,
				)
				lowerLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					false,
					subaccountID,
				)
			)

			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(restingBuyOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(restingSellOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(buyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(sellOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(higherMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(lowerMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(higherLimitOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(lowerLimitOrders)), "")

			for _, hash := range restingBuyOrderHashes {
				isBuy := true
				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range restingSellOrderHashes {
				isBuy := false
				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, buyOrder := range buyOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
					orderHash := common.BytesToHash(buyOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for buyOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					k.EmitEvent(
						ctx,
						v2.NewEventOrderCancelFail(
							marketID,
							subaccountID,
							orderHash.Hex(),
							buyOrder.Cid(),
							err,
						))
				}
			}

			for _, sellOrder := range sellOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
					orderHash := common.BytesToHash(sellOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for sellOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					k.EmitEvent(ctx, v2.NewEventOrderCancelFail(
						marketID,
						subaccountID,
						orderHash.Hex(),
						sellOrder.Cid(),
						err,
					))
				}
			}

			for _, hash := range higherMarketOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerMarketOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range higherLimitOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerLimitOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}
		}
	}

	/**	2. Cancel all spot limit orders **/
	for idx, spotOrderToCancel := range msg.SpotOrdersToCancel {
		marketID := common.HexToHash(spotOrderToCancel.MarketId)

		var market *v2.SpotMarket
		if m, ok := spotMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel spot limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			spotMarkets[marketID] = market
		}

		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, spotOrderToCancel.SubaccountId)

		err := k.cancelSpotLimitOrderWithIdentifier(ctx, subaccountID, spotOrderToCancel.GetIdentifier(), market, marketID)

		if err == nil {
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas, "")
			spotCancelSuccesses[idx] = true
		} else {
			ev := v2.NewEventOrderCancelFail(
				marketID,
				subaccountID,
				spotOrderToCancel.GetOrderHash(),
				spotOrderToCancel.GetCid(),
				err,
			)
			k.EmitEvent(ctx, ev)
		}
	}

	/**	3. Cancel all derivative limit orders **/
	for idx, derivativeOrderToCancel := range msg.DerivativeOrdersToCancel {
		marketID := common.HexToHash(derivativeOrderToCancel.MarketId)

		var market *v2.DerivativeMarket
		if m, ok := derivativeMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel derivative limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			derivativeMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, derivativeOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(
			ctx,
			subaccountID,
			derivativeOrderToCancel.GetIdentifier(),
			market,
			marketID,
			derivativeOrderToCancel.OrderMask,
		)

		if err == nil {
			derivativeCancelSuccesses[idx] = true
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas, "")
		} else {
			ev := v2.NewEventOrderCancelFail(
				marketID,
				subaccountID,
				derivativeOrderToCancel.GetOrderHash(),
				derivativeOrderToCancel.GetCid(),
				err,
			)
			k.EmitEvent(ctx, ev)
		}
	}

	/**	4. Cancel all bo limit orders **/
	for idx, binaryOptionsOrderToCancel := range msg.BinaryOptionsOrdersToCancel {
		marketID := common.HexToHash(binaryOptionsOrderToCancel.MarketId)

		var market *v2.BinaryOptionsMarket
		if m, ok := binaryOptionsMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel binary options limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			binaryOptionsMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, binaryOptionsOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(
			ctx,
			subaccountID,
			binaryOptionsOrderToCancel.GetIdentifier(),
			market,
			marketID,
			binaryOptionsOrderToCancel.OrderMask,
		)

		if err == nil {
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas, "")
			binaryOptionsCancelSuccesses[idx] = true
		} else {
			ev := v2.NewEventOrderCancelFail(
				marketID,
				subaccountID,
				binaryOptionsOrderToCancel.GetOrderHash(),
				binaryOptionsOrderToCancel.GetCid(),
				err,
			)
			k.EmitEvent(ctx, ev)
		}
	}

	orderFailEvent := v2.EventOrderFail{
		Account: sender.Bytes(),
		Hashes:  make([][]byte, 0),
		Flags:   make([]uint32, 0),
		Cids:    make([]string, 0),
	}

	/**	5. Create spot limit orders **/
	for idx, spotOrder := range msg.SpotOrdersToCreate {
		marketID := common.HexToHash(spotOrder.MarketId)
		var market *v2.SpotMarket
		if m, ok := spotMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to create spot limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			spotMarkets[marketID] = market
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug(
				"failed to create spot limit order for non-active market",
				"marketID",
				marketID.Hex(),
			)
			continue
		}

		var gasToConsume uint64
		if spotOrder.OrderType == v2.OrderType_BUY_PO || spotOrder.OrderType == v2.OrderType_SELL_PO {
			gasToConsume = MsgCreateSpotLimitPostOnlyOrderGas
		} else {
			gasToConsume = MsgCreateSpotLimitOrderGas
		}

		if orderHash, err := k.createSpotLimitOrder(ctx, sender, spotOrder, market); err != nil {
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				spotOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, spotOrder.Cid(), sdkerror.ABCICode())
				failedSpotOrdersCids = append(failedSpotOrdersCids, spotOrder.Cid())
			}
		} else {
			gasMeter.ConsumeGas(gasToConsume, "")
			spotOrderHashes[idx] = orderHash.Hex()
			createdSpotOrdersCids = append(createdSpotOrdersCids, spotOrder.Cid())
		}
	}

	markPrices := make(map[common.Hash]math.LegacyDec)

	/**	6. Create derivative limit orders **/
	for idx, derivativeOrder := range msg.DerivativeOrdersToCreate {
		marketID := derivativeOrder.MarketID()

		var market *v2.DerivativeMarket
		var markPrice math.LegacyDec
		if m, ok := derivativeMarkets[marketID]; ok {
			market = m
		} else {
			market, markPrice = k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to create derivative limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			derivativeMarkets[marketID] = market
			markPrices[marketID] = markPrice
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug(
				"failed to create derivative limit orders for non-active market",
				"marketID",
				marketID.Hex(),
			)
			continue
		}

		if _, ok := markPrices[marketID]; !ok {
			price, err := k.GetDerivativeMarketPrice(
				ctx,
				market.OracleBase,
				market.OracleQuote,
				market.OracleScaleFactor,
				market.OracleType,
			)
			if err != nil {
				k.Logger(ctx).Debug(
					"failed to create derivative limit order for market with no mark price",
					"marketID",
					marketID.Hex(),
				)
				metrics.ReportFuncError(k.svcTags)
				continue
			}
			markPrices[marketID] = *price
		}
		markPrice = markPrices[marketID]

		var gasToConsume uint64
		if derivativeOrder.OrderType == v2.OrderType_BUY_PO || derivativeOrder.OrderType == v2.OrderType_SELL_PO {
			gasToConsume = MsgCreateDerivativeLimitPostOnlyOrderGas
		} else {
			gasToConsume = MsgCreateDerivativeLimitOrderGas
		}

		if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, derivativeOrder, market, markPrice); err != nil {
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				derivativeOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, derivativeOrder.Cid(), sdkerror.ABCICode())
				failedDerivativeOrdersCids = append(failedDerivativeOrdersCids, derivativeOrder.Cid())
			}
		} else {
			gasMeter.ConsumeGas(gasToConsume, "")
			derivativeOrderHashes[idx] = orderHash.Hex()
			createdDerivativeOrdersCids = append(createdDerivativeOrdersCids, derivativeOrder.Cid())
		}
	}

	/**	7. Create bo limit orders **/
	for idx, order := range msg.BinaryOptionsOrdersToCreate {
		marketID := order.MarketID()

		var market *v2.BinaryOptionsMarket
		if m, ok := binaryOptionsMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetBinaryOptionsMarket(ctx, marketID, true)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to create binary options limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			binaryOptionsMarkets[marketID] = market
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug(
				"failed to create binary options limit orders for non-active market",
				"marketID",
				marketID.Hex(),
			)
			continue
		}

		var gasToConsume uint64
		switch order.OrderType {
		case v2.OrderType_BUY_PO, v2.OrderType_SELL_PO:
			gasToConsume = MsgCreateBinaryOptionsLimitPostOnlyOrderGas
		default:
			gasToConsume = MsgCreateBinaryOptionsLimitOrderGas
		}

		if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, order, market, math.LegacyDec{}); err != nil {
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				binaryOptionsOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, order.Cid(), sdkerror.ABCICode())
				failedBinaryOptionsOrdersCids = append(failedBinaryOptionsOrdersCids, order.Cid())
			}
		} else {
			gasMeter.ConsumeGas(gasToConsume, "")
			binaryOptionsOrderHashes[idx] = orderHash.Hex()
			createdBinaryOptionsOrdersCids = append(createdBinaryOptionsOrdersCids, order.Cid())
		}
	}

	if !orderFailEvent.IsEmpty() {
		k.EmitEvent(ctx, &orderFailEvent)
	}

	return &v2.MsgBatchUpdateOrdersResponse{
		SpotCancelSuccess:              spotCancelSuccesses,
		DerivativeCancelSuccess:        derivativeCancelSuccesses,
		SpotOrderHashes:                spotOrderHashes,
		DerivativeOrderHashes:          derivativeOrderHashes,
		BinaryOptionsCancelSuccess:     binaryOptionsCancelSuccesses,
		BinaryOptionsOrderHashes:       binaryOptionsOrderHashes,
		CreatedSpotOrdersCids:          createdSpotOrdersCids,
		FailedSpotOrdersCids:           failedSpotOrdersCids,
		CreatedDerivativeOrdersCids:    createdDerivativeOrdersCids,
		FailedDerivativeOrdersCids:     failedDerivativeOrdersCids,
		CreatedBinaryOptionsOrdersCids: createdBinaryOptionsOrdersCids,
		FailedBinaryOptionsOrdersCids:  failedBinaryOptionsOrdersCids,
	}, nil
}

// ProcessExpiredDOrders processes all expired orders at the current block height
func (k *Keeper) ProcessExpiredOrders(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				ctx.Logger().Error("BeginBlocker (ProcessExpiredOrders) panicked with an error: ", e)
				ctx.Logger().Error(string(debug.Stack()))
			} else {
				ctx.Logger().Error("BeginBlocker (ProcessExpiredOrders) panicked with a msg: ", r)
			}
		}
	}()

	blockHeight := ctx.BlockHeight()
	marketIDs := k.GetMarketsWithOrderExpirations(ctx, blockHeight)

	marketFinder := NewCachedMarketFinder(k)

	for _, marketID := range marketIDs {
		market, err := marketFinder.FindMarket(ctx, marketID.Hex())
		if err != nil {
			ctx.Logger().Error("failed to find market with GTB orders", "error", err, "marketID", marketID)
			continue
		}
		k.processMarketExpiredOrders(ctx, market, blockHeight)
	}
}

func (k *Keeper) processMarketExpiredOrders(ctx sdk.Context, market MarketInterface, blockHeight int64) {
	defer k.DeleteMarketWithOrderExpirations(ctx, market.MarketID(), blockHeight)

	orders, err := k.GetOrdersByExpiration(ctx, market.MarketID(), blockHeight)
	if err != nil {
		ctx.Logger().Error("failed to get expired orders", "error", err, "marketID", market.MarketID())
		return
	}

	if len(orders) == 0 {
		return
	}

	for _, order := range orders {
		spotMarket, ok := market.(*v2.SpotMarket)
		if ok {
			if err := k.cancelSpotLimitOrderWithIdentifier(
				ctx,
				common.HexToHash(order.SubaccountId),
				order.GetIdentifier(),
				spotMarket,
				market.MarketID(),
			); err != nil {
				k.EmitEvent(ctx, v2.NewEventOrderCancelFail(
					market.MarketID(),
					common.HexToHash(order.SubaccountId),
					order.OrderHash,
					order.Cid,
					err,
				))
			}
		} else {
			if err := k.cancelDerivativeOrder(
				ctx,
				common.HexToHash(order.SubaccountId),
				order.GetIdentifier(),
				market,
				market.MarketID(),
				int32(v2.OrderMask_ANY),
			); err != nil {
				k.EmitEvent(ctx, v2.NewEventOrderCancelFail(
					market.MarketID(),
					common.HexToHash(order.SubaccountId),
					order.OrderHash,
					order.Cid,
					err,
				))
			}
		}

		k.DeleteOrderExpiration(ctx, market.MarketID(), blockHeight, common.HexToHash(order.OrderHash))
	}
}

func (k *Keeper) processCancelAllSpotOrders(
	ctx sdk.Context,
	spotMarketIDsToCancelAll []string,
	subaccountID common.Hash,
	spotMarkets map[common.Hash]*v2.SpotMarket,
) {
	for _, spotMarketIdToCancelAll := range spotMarketIDsToCancelAll {
		marketID := common.HexToHash(spotMarketIdToCancelAll)
		market := k.GetSpotMarketByID(ctx, marketID)
		if market == nil {
			continue
		}
		spotMarkets[marketID] = market

		if !market.StatusSupportsOrderCancellations() {
			k.Logger(ctx).Debug("failed to cancel all spot limit orders", "marketID", marketID.Hex())
			continue
		}

		k.CancelAllSpotLimitOrders(ctx, market, subaccountID, marketID)
	}
}

func (k *Keeper) processCancelAllDerivativeOrders(
	ctx sdk.Context,
	derivativeMarketIDsToCancelAll []string,
	subaccountID common.Hash,
	derivativeMarkets map[common.Hash]*v2.DerivativeMarket,
) {
	for _, derivativeMarketIdToCancelAll := range derivativeMarketIDsToCancelAll {
		marketID := common.HexToHash(derivativeMarketIdToCancelAll)
		market := k.GetDerivativeMarketByID(ctx, marketID)
		if market == nil {
			k.Logger(ctx).Debug("failed to cancel all derivative limit orders for non-existent market", "marketID", marketID.Hex())
			continue
		}
		derivativeMarkets[marketID] = market

		if !market.StatusSupportsOrderCancellations() {
			k.Logger(ctx).Debug(
				"failed to cancel all derivative limit orders for market whose status doesnt support cancellations",
				"marketID", marketID.Hex(),
			)
			continue
		}

		k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountID, true, true)
		k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountID)
		k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountID)
	}
}

func (k *Keeper) processCancelAllBinaryOptionsOrders(
	ctx sdk.Context,
	binaryOptionsMarketIDsToCancelAll []string,
	subaccountID common.Hash,
	binaryOptionsMarkets map[common.Hash]*v2.BinaryOptionsMarket,
) {
	for _, binaryOptionsMarketIdToCancelAll := range binaryOptionsMarketIDsToCancelAll {
		marketID := common.HexToHash(binaryOptionsMarketIdToCancelAll)
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)
		if market == nil {
			k.Logger(ctx).Debug("failed to cancel all binary options limit orders for non-existent market", "marketID", marketID.Hex())
			continue
		}
		binaryOptionsMarkets[marketID] = market

		if !market.StatusSupportsOrderCancellations() {
			k.Logger(ctx).Debug(
				"failed to cancel all binary options limit orders for market whose status doesnt support cancellations",
				"marketID", marketID.Hex(),
			)
			continue
		}

		k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountID, true, true)
		k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountID)
		k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountID)
	}
}

func (k *Keeper) processCancelSpotOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	spotOrdersToCancel []*v2.OrderData,
	spotCancelSuccesses []bool,
	spotMarkets map[common.Hash]*v2.SpotMarket,
) {
	for idx, spotOrderToCancel := range spotOrdersToCancel {
		marketID := common.HexToHash(spotOrderToCancel.MarketId)

		var market *v2.SpotMarket
		if m, ok := spotMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel spot limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			spotMarkets[marketID] = market
		}

		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, spotOrderToCancel.SubaccountId)

		err := k.cancelSpotLimitOrderWithIdentifier(ctx, subaccountID, spotOrderToCancel.GetIdentifier(), market, marketID)

		if err == nil {
			spotCancelSuccesses[idx] = true
		} else {
			ev := v2.NewEventOrderCancelFail(marketID, subaccountID, spotOrderToCancel.GetOrderHash(), spotOrderToCancel.GetCid(), err)
			k.EmitEvent(ctx, ev)
		}
	}
}

func (k *Keeper) processCancelDerivativeOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrdersToCancel []*v2.OrderData,
	derivativeCancelSuccesses []bool,
	derivativeMarkets map[common.Hash]*v2.DerivativeMarket,
) {
	for idx, derivativeOrderToCancel := range derivativeOrdersToCancel {
		marketID := common.HexToHash(derivativeOrderToCancel.MarketId)

		var market *v2.DerivativeMarket
		if m, ok := derivativeMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel derivative limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			derivativeMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, derivativeOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(
			ctx, subaccountID, derivativeOrderToCancel.GetIdentifier(), market, marketID, derivativeOrderToCancel.OrderMask,
		)

		if err == nil {
			derivativeCancelSuccesses[idx] = true
		} else {
			ev := v2.NewEventOrderCancelFail(marketID, subaccountID, derivativeOrderToCancel.GetOrderHash(), derivativeOrderToCancel.GetCid(), err)
			k.EmitEvent(ctx, ev)
		}
	}
}

//revive:disable:cognitive-complexity // The complexity is acceptable and creating more helper functions would make the code less readable
func (k *Keeper) processCancelBinaryOptionsOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	binaryOptionsOrdersToCancel []*v2.OrderData,
	binaryOptionsCancelSuccesses []bool,
	binaryOptionsMarkets map[common.Hash]*v2.BinaryOptionsMarket,
) {
	for idx, binaryOptionsOrderToCancel := range binaryOptionsOrdersToCancel {
		marketID := common.HexToHash(binaryOptionsOrderToCancel.MarketId)

		var market *v2.BinaryOptionsMarket
		if m, ok := binaryOptionsMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel binary options limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			binaryOptionsMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, binaryOptionsOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(
			ctx, subaccountID, binaryOptionsOrderToCancel.GetIdentifier(), market, marketID, binaryOptionsOrderToCancel.OrderMask,
		)

		if err == nil {
			binaryOptionsCancelSuccesses[idx] = true
		} else {
			ev := v2.NewEventOrderCancelFail(
				marketID, subaccountID, binaryOptionsOrderToCancel.GetOrderHash(), binaryOptionsOrderToCancel.GetCid(), err,
			)
			k.EmitEvent(ctx, ev)
		}
	}
}

func (k *Keeper) processCreateSpotOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	spotOrdersToCreate []*v2.SpotOrder,
	spotOrderHashes []string,
	orderFailEvent *v2.EventOrderFail,
	createdSpotOrdersCids *[]string,
	failedSpotOrdersCids *[]string,
	spotMarkets map[common.Hash]*v2.SpotMarket,
) {
	for idx, spotOrder := range spotOrdersToCreate {
		marketID := common.HexToHash(spotOrder.MarketId)
		market := k.getSpotMarketForOrder(ctx, marketID, spotMarkets)
		if market == nil {
			continue
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug("failed to create spot limit order for non-active market", "marketID", marketID.Hex())
			continue
		}

		k.processSpotOrderCreation(
			ctx, sender, spotOrder, market, idx, spotOrderHashes, orderFailEvent, createdSpotOrdersCids, failedSpotOrdersCids,
		)
	}
}

func (k *Keeper) getSpotMarketForOrder(
	ctx sdk.Context,
	marketID common.Hash,
	spotMarkets map[common.Hash]*v2.SpotMarket,
) *v2.SpotMarket {
	if m, ok := spotMarkets[marketID]; ok {
		return m
	}

	market := k.GetSpotMarketByID(ctx, marketID)
	if market == nil {
		k.Logger(ctx).Debug("failed to create spot limit order for non-existent market", "marketID", marketID.Hex())
		return nil
	}

	spotMarkets[marketID] = market
	return market
}

func (k *Keeper) processSpotOrderCreation(
	ctx sdk.Context,
	sender sdk.AccAddress,
	spotOrder *v2.SpotOrder,
	market *v2.SpotMarket,
	idx int,
	spotOrderHashes []string,
	orderFailEvent *v2.EventOrderFail,
	createdSpotOrdersCids *[]string,
	failedSpotOrdersCids *[]string,
) {
	if orderHash, err := k.createSpotLimitOrder(ctx, sender, spotOrder, market); err != nil {
		sdkerror := &sdkerrors.Error{}
		if errors.As(err, &sdkerror) {
			spotOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
			orderFailEvent.AddOrderFail(orderHash, spotOrder.Cid(), sdkerror.ABCICode())
			*failedSpotOrdersCids = append(*failedSpotOrdersCids, spotOrder.Cid())
		}
	} else {
		spotOrderHashes[idx] = orderHash.Hex()
		*createdSpotOrdersCids = append(*createdSpotOrdersCids, spotOrder.Cid())
	}
}

func (k *Keeper) processCreateDerivativeOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrdersToCreate []*v2.DerivativeOrder,
	derivativeOrderHashes []string,
	orderFailEvent *v2.EventOrderFail,
	createdDerivativeOrdersCids *[]string,
	failedDerivativeOrdersCids *[]string,
	derivativeMarkets map[common.Hash]*v2.DerivativeMarket,
	markPrices map[common.Hash]math.LegacyDec,
) {
	for idx, derivativeOrder := range derivativeOrdersToCreate {
		marketID := derivativeOrder.MarketID()

		market, markPrice := k.getDerivativeMarketForOrder(ctx, marketID, derivativeMarkets, markPrices)
		if market == nil {
			continue
		}

		k.processDerivativeOrderCreation(
			ctx, sender, derivativeOrder, market, markPrice, idx,
			derivativeOrderHashes, orderFailEvent,
			createdDerivativeOrdersCids, failedDerivativeOrdersCids,
		)
	}
}

func (k *Keeper) getDerivativeMarketForOrder(
	ctx sdk.Context,
	marketID common.Hash,
	derivativeMarkets map[common.Hash]*v2.DerivativeMarket,
	markPrices map[common.Hash]math.LegacyDec,
) (*v2.DerivativeMarket, math.LegacyDec) {
	var market *v2.DerivativeMarket
	var markPrice math.LegacyDec

	if m, ok := derivativeMarkets[marketID]; ok {
		market = m
	} else {
		market, markPrice = k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
		if market == nil {
			k.Logger(ctx).Debug("failed to create derivative limit order for non-existent market", "marketID", marketID.Hex())
			return nil, math.LegacyDec{}
		}
		derivativeMarkets[marketID] = market
		markPrices[marketID] = markPrice
	}

	if !market.IsActive() {
		k.Logger(ctx).Debug("failed to create derivative limit orders for non-active market", "marketID", marketID.Hex())
		return nil, math.LegacyDec{}
	}

	if _, ok := markPrices[marketID]; !ok {
		price, err := k.GetDerivativeMarketPrice(
			ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType,
		)
		if err != nil {
			k.Logger(ctx).Debug("failed to create derivative limit order for market with no mark price", "marketID", marketID.Hex())
			metrics.ReportFuncError(k.svcTags)
			return nil, math.LegacyDec{}
		}
		markPrices[marketID] = *price
	}

	return market, markPrices[marketID]
}

func (k *Keeper) processDerivativeOrderCreation(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	idx int,
	derivativeOrderHashes []string,
	orderFailEvent *v2.EventOrderFail,
	createdDerivativeOrdersCids *[]string,
	failedDerivativeOrdersCids *[]string,
) {
	if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, derivativeOrder, market, markPrice); err != nil {
		sdkerror := &sdkerrors.Error{}
		if errors.As(err, &sdkerror) {
			derivativeOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
			orderFailEvent.AddOrderFail(orderHash, derivativeOrder.Cid(), sdkerror.ABCICode())
			*failedDerivativeOrdersCids = append(*failedDerivativeOrdersCids, derivativeOrder.Cid())
		}
	} else {
		derivativeOrderHashes[idx] = orderHash.Hex()
		*createdDerivativeOrdersCids = append(*createdDerivativeOrdersCids, derivativeOrder.Cid())
	}
}

func (k *Keeper) processCreateBinaryOptionsOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	binaryOptionsOrdersToCreate []*v2.DerivativeOrder,
	binaryOptionsOrderHashes []string,
	orderFailEvent *v2.EventOrderFail,
	createdBinaryOptionsOrdersCids *[]string,
	failedBinaryOptionsOrdersCids *[]string,
	binaryOptionsMarkets map[common.Hash]*v2.BinaryOptionsMarket,
) {
	for idx, order := range binaryOptionsOrdersToCreate {
		marketID := order.MarketID()

		market := k.getBinaryOptionsMarketForOrder(ctx, marketID, binaryOptionsMarkets)
		if market == nil {
			continue
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug("failed to create binary options limit orders for non-active market", "marketID", marketID.Hex())
			continue
		}

		k.processBinaryOptionsOrderCreation(
			ctx,
			sender,
			order,
			market,
			idx,
			binaryOptionsOrderHashes,
			orderFailEvent,
			createdBinaryOptionsOrdersCids,
			failedBinaryOptionsOrdersCids,
		)
	}
}

func (k *Keeper) getBinaryOptionsMarketForOrder(
	ctx sdk.Context,
	marketID common.Hash,
	binaryOptionsMarkets map[common.Hash]*v2.BinaryOptionsMarket,
) *v2.BinaryOptionsMarket {
	if m, ok := binaryOptionsMarkets[marketID]; ok {
		return m
	}

	market := k.GetBinaryOptionsMarket(ctx, marketID, true)
	if market == nil {
		k.Logger(ctx).Debug("failed to create binary options limit order for non-existent market", "marketID", marketID.Hex())
		return nil
	}

	binaryOptionsMarkets[marketID] = market
	return market
}

func (k *Keeper) processBinaryOptionsOrderCreation(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order *v2.DerivativeOrder,
	market *v2.BinaryOptionsMarket,
	idx int,
	binaryOptionsOrderHashes []string,
	orderFailEvent *v2.EventOrderFail,
	createdBinaryOptionsOrdersCids *[]string,
	failedBinaryOptionsOrdersCids *[]string,
) {
	if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, order, market, math.LegacyDec{}); err != nil {
		sdkerror := &sdkerrors.Error{}
		if errors.As(err, &sdkerror) {
			binaryOptionsOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
			orderFailEvent.AddOrderFail(orderHash, order.Cid(), sdkerror.ABCICode())
			*failedBinaryOptionsOrdersCids = append(*failedBinaryOptionsOrdersCids, order.Cid())
		}
	} else {
		binaryOptionsOrderHashes[idx] = orderHash.Hex()
		*createdBinaryOptionsOrdersCids = append(*createdBinaryOptionsOrdersCids, order.Cid())
	}
}

func (k *Keeper) IsGovernanceAuthorityAddress(address string) bool {
	return address == k.authority
}

func (k *Keeper) getStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) getTransientStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.TransientStore(k.tStoreKey)
}

func (k *Keeper) IsAdmin(ctx sdk.Context, addr string) bool {
	for _, adminAddress := range k.GetParams(ctx).ExchangeAdmins {
		if adminAddress == addr {
			return true
		}
	}
	return false
}

func (k *Keeper) handleBatchCommunityPoolSpendProposal(ctx sdk.Context, p *v2.BatchCommunityPoolSpendProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, proposal := range p.Proposals {
		recipient, addrErr := sdk.AccAddressFromBech32(proposal.Recipient)
		if addrErr != nil {
			return addrErr
		}

		err := k.DistributionKeeper.DistributeFromFeePool(ctx, proposal.Amount, recipient)
		if err != nil {
			return err
		}

		ctx.Logger().Info(
			"transferred from the community pool to recipient",
			"amount", proposal.Amount.String(),
			"recipient", proposal.Recipient,
		)
	}

	return nil
}

func (k *Keeper) handleDenomMinNotionalProposal(
	ctx sdk.Context,
	p *v2.DenomMinNotionalProposal,
) {
	for _, denomMinNotional := range p.DenomMinNotionals {
		k.SetMinNotionalForDenom(ctx, denomMinNotional.Denom, denomMinNotional.MinNotional)
	}
}

func (k *Keeper) IsFixedGasEnabled() bool {
	return k.fixedGas
}

func (k *Keeper) SetFixedGasEnabled(enabled bool) {
	k.fixedGas = enabled
}
