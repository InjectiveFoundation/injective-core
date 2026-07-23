package keeper

import (
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"cosmossdk.io/collections"
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/base"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/binaryoptions"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/derivative"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/feediscounts"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/marketfinder"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/proposals"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/rewards"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/spot"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/subaccount"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/utils"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/wasm"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/risk"
	_ "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/risk/cross"    // registers cross-margin model factory
	_ "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/risk/isolated" // registers isolated-margin model factory
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// Keeper of this module maintains collections of exchange.
type Keeper struct {
	DistributionKeeper distrkeeper.Keeper
	AccountKeeper      authkeeper.AccountKeeper // todo: revisit
	OracleKeeper       types.OracleKeeper       // todo: revisit

	*base.BaseKeeper
	*subaccount.SubaccountKeeper
	*binaryoptions.BinaryOptionsKeeper
	*derivative.DerivativeKeeper
	*spot.SpotKeeper
	*feediscounts.FeeDiscountsKeeper
	*rewards.TradingKeeper
	*wasm.WasmKeeper
	*proposals.ProposalKeeper

	bankKeeper           bankkeeper.Keeper
	govKeeper            govkeeper.Keeper
	wasmViewKeeper       types.WasmViewKeeper
	wasmxExecutionKeeper types.WasmxExecutionKeeper
	DowntimeKeeper       types.DowntimeKeeper
	permissionsKeeper    types.PermissionsKeeper
	StakingKeeper        types.StakingKeeper

	authority string

	// cached value from params (false by default)
	fixedGas bool

	// riskEngine handles risk-sensitive admission, matching-time pruning, and liquidation checks.
	riskEngine *risk.Engine
}

// NewKeeper creates new instances of the exchange Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	tStoreKey storetypes.StoreKey,
	objectStoreKey storetypes.StoreKey,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	ok types.OracleKeeper,
	ik types.InsuranceKeeper,
	dk distrkeeper.Keeper,
	sk types.StakingKeeper,
	downtimeK types.DowntimeKeeper,
	pk types.PermissionsKeeper,
	authority string,
) *Keeper {
	var (
		b            = base.NewBaseKeeper(cdc, storeKey, tStoreKey, objectStoreKey)
		riskEngine   = risk.New(b)
		subacc       = subaccount.New(b, ak, bk, pk)
		feeDiscounts = feediscounts.New(b, sk)
		trade        = rewards.New(b, bk, feeDiscounts, dk)
		derv         = derivative.New(b, riskEngine, subacc, ok, feeDiscounts, bk, ik, trade, pk)
	)

	// Wire cross-margin snapshot dependencies into the risk engine.
	riskEngine.SetCrossMarginDeps(riskCrossDeps{
		base:       b,
		derivative: derv,
	})
	riskEngine.SetParamsProvider(b)

	// Fail fast if risk engine wiring is incomplete.
	if err := riskEngine.EnsureWired(); err != nil {
		panic(err)
	}

	return &Keeper{
		BaseKeeper:          b,
		SubaccountKeeper:    subacc,
		BinaryOptionsKeeper: binaryoptions.New(b, derv, subacc, ok, ak, trade, feeDiscounts),
		DerivativeKeeper:    derv,
		SpotKeeper:          spot.New(b, riskEngine, bk, subacc, trade, feeDiscounts),
		FeeDiscountsKeeper:  feeDiscounts,
		TradingKeeper:       trade,

		DowntimeKeeper:     downtimeK,
		permissionsKeeper:  pk,
		AccountKeeper:      ak,
		DistributionKeeper: dk,
		OracleKeeper:       ok,
		bankKeeper:         bk,
		StakingKeeper:      sk,
		authority:          authority,
		fixedGas:           false,
		riskEngine:         riskEngine,
	}
}

func (k *Keeper) RiskEngine() risk.ReadOnlyEngine {
	return k.riskEngine
}

func (k *Keeper) SetGovKeeper(gk govkeeper.Keeper) {
	k.govKeeper = gk

	// now we can set proposal keeper
	k.ProposalKeeper = proposals.New(
		k.BaseKeeper,
		k.AccountKeeper,
		k.OracleKeeper,
		k.govKeeper,
		k.DistributionKeeper,
		k.BinaryOptionsKeeper,
		k.DerivativeKeeper,
		k.SpotKeeper,
		k.TradingKeeper,
		k.SubaccountKeeper,
		k.FeeDiscountsKeeper,
	)
}

func (k *Keeper) EmitEvent(ctx sdk.Context, event proto.Message) {
	events.Emit(ctx, k.BaseKeeper, event)
}

func (k *Keeper) SetWasmKeepers(
	wk wasmkeeper.Keeper,
	wxk types.WasmxExecutionKeeper,
) {
	k.wasmViewKeeper = types.WasmViewKeeper(wk)
	k.wasmxExecutionKeeper = wxk
	k.DerivativeKeeper = k.SetWasm(k.wasmViewKeeper)

	// now we can set wasm keeper
	k.WasmKeeper = wasm.New(
		k.BaseKeeper,
		k.bankKeeper,
		k.SubaccountKeeper,
		k.DerivativeKeeper,
		k.wasmViewKeeper,
		k.wasmxExecutionKeeper,
	)
}

func (k *Keeper) SetPermissionsKeeper(pk types.PermissionsKeeper) {
	k.permissionsKeeper = pk
	k.SubaccountKeeper.SetPermissionsKeeper(pk)
	k.DerivativeKeeper.SetPermissionsKeeper(pk)
}

// CreateModuleAccount creates a module account with minter and burning capabilities
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	//revive:disable:unchecked-type-assertion // we know the type is correct
	moduleAcc := (k.AccountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI)
	k.AccountKeeper.SetModuleAccount(ctx, moduleAcc)
}

// GetAllDerivativeAndBinaryOptionsLimitOrderbook returns all orderbooks for all derivative markets.
func (k *Keeper) GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx sdk.Context) []v2.DerivativeOrderBook {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllDerivativeAndBinaryOptionsLimitOrderbook")()

	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)
	binaryOptionsMarkets := k.GetAllBinaryOptionsMarkets(ctx)

	markets := make([]v2.DerivativeMarketI, 0, len(derivativeMarkets)+len(binaryOptionsMarkets))
	for _, m := range derivativeMarkets {
		markets = append(markets, m)
	}

	for _, m := range binaryOptionsMarkets {
		markets = append(markets, m)
	}

	orderbook := make([]v2.DerivativeOrderBook, 0, len(markets)*2)
	for _, market := range markets {
		marketID := market.MarketID()
		orderbook = append(orderbook, v2.DerivativeOrderBook{
			MarketId:  marketID.Hex(),
			IsBuySide: true,
			Orders:    k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true),
		},
			v2.DerivativeOrderBook{
				MarketId:  marketID.Hex(),
				IsBuySide: false,
				Orders:    k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false),
			})
	}

	return orderbook
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
	spotLimitOrdersToCreate []*v2.SpotOrder,
	derivativeLimitOrdersToCreate []*v2.DerivativeOrder,
	binaryOptionsLimitOrdersToCreate []*v2.DerivativeOrder,
	spotMarketOrdersToCreate []*v2.SpotOrder,
	derivativeMarketOrdersToCreate []*v2.DerivativeOrder,
	binaryOptionsMarketOrdersToCreate []*v2.DerivativeOrder,
) (*v2.MsgBatchUpdateOrdersResponse, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExecuteBatchUpdateOrders")()

	var (
		spotMarkets          = make(map[common.Hash]*v2.SpotMarket)
		derivativeMarkets    = make(map[common.Hash]*v2.DerivativeMarket)
		binaryOptionsMarkets = make(map[common.Hash]*v2.BinaryOptionsMarket)

		spotCancelSuccesses                  = make([]bool, len(spotOrdersToCancel))
		derivativeCancelSuccesses            = make([]bool, len(derivativeOrdersToCancel))
		binaryOptionsCancelSuccesses         = make([]bool, len(binaryOptionsOrdersToCancel))
		spotLimitOrderHashes                 = make([]string, len(spotLimitOrdersToCreate))
		createdSpotLimitOrdersCids           = make([]string, 0)
		failedSpotLimitOrdersCids            = make([]string, 0)
		spotMarketOrderHashes                = make([]string, len(spotMarketOrdersToCreate))
		createdSpotMarketOrdersCids          = make([]string, 0)
		failedSpotMarketOrdersCids           = make([]string, 0)
		derivativeOrderHashes                = make([]string, len(derivativeLimitOrdersToCreate))
		createdDerivativeOrdersCids          = make([]string, 0)
		failedDerivativeOrdersCids           = make([]string, 0)
		derivativeMarketOrderHashes          = make([]string, len(derivativeMarketOrdersToCreate))
		createdDerivativeMarketOrdersCids    = make([]string, 0)
		failedDerivativeMarketOrdersCids     = make([]string, 0)
		binaryOptionsOrderHashes             = make([]string, len(binaryOptionsLimitOrdersToCreate))
		createdBinaryOptionsOrdersCids       = make([]string, 0)
		failedBinaryOptionsOrdersCids        = make([]string, 0)
		binaryOptionsMarketOrderHashes       = make([]string, len(binaryOptionsMarketOrdersToCreate))
		createdBinaryOptionsMarketOrdersCids = make([]string, 0)
		failedBinaryOptionsMarketOrdersCids  = make([]string, 0)
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
		ctx,
		sender,
		spotMarketOrdersToCreate,
		spotMarketOrderHashes,
		&createdSpotMarketOrdersCids,
		&failedSpotMarketOrdersCids,
		&orderFailEvent,
		spotMarkets,
		k.createSpotMarketOrder,
	)
	k.processCreateSpotOrders(
		ctx,
		sender,
		spotLimitOrdersToCreate,
		spotLimitOrderHashes,
		&createdSpotLimitOrdersCids,
		&failedSpotLimitOrdersCids,
		&orderFailEvent,
		spotMarkets,
		k.CreateSpotLimitOrder,
	)

	markPrices := make(map[common.Hash]math.LegacyDec)
	k.processCreateDerivativeOrders(
		ctx,
		sender,
		derivativeMarketOrdersToCreate,
		derivativeMarketOrderHashes,
		&orderFailEvent,
		&createdDerivativeMarketOrdersCids,
		&failedDerivativeMarketOrdersCids,
		derivativeMarkets,
		markPrices,
		k.createDerivativeMarketOrderWithoutResultsForAtomicExecution,
	)
	k.processCreateDerivativeOrders(
		ctx,
		sender,
		derivativeLimitOrdersToCreate,
		derivativeOrderHashes,
		&orderFailEvent,
		&createdDerivativeOrdersCids,
		&failedDerivativeOrdersCids,
		derivativeMarkets,
		markPrices,
		k.CreateDerivativeLimitOrder,
	)
	k.processCreateBinaryOptionsOrders(
		ctx,
		sender,
		binaryOptionsMarketOrdersToCreate,
		binaryOptionsMarketOrderHashes,
		&orderFailEvent,
		&createdBinaryOptionsMarketOrdersCids,
		&failedBinaryOptionsMarketOrdersCids,
		binaryOptionsMarkets,
		k.CreateBinaryOptionsMarketOrder,
	)
	k.processCreateBinaryOptionsOrders(
		ctx,
		sender,
		binaryOptionsLimitOrdersToCreate,
		binaryOptionsOrderHashes,
		&orderFailEvent,
		&createdBinaryOptionsOrdersCids,
		&failedBinaryOptionsOrdersCids,
		binaryOptionsMarkets,
		k.CreateDerivativeLimitOrder,
	)

	if !orderFailEvent.IsEmpty() {
		k.EmitEvent(ctx, &orderFailEvent)
	}

	return &v2.MsgBatchUpdateOrdersResponse{
		SpotCancelSuccess:                    spotCancelSuccesses,
		DerivativeCancelSuccess:              derivativeCancelSuccesses,
		SpotOrderHashes:                      spotLimitOrderHashes,
		DerivativeOrderHashes:                derivativeOrderHashes,
		BinaryOptionsCancelSuccess:           binaryOptionsCancelSuccesses,
		BinaryOptionsOrderHashes:             binaryOptionsOrderHashes,
		CreatedSpotOrdersCids:                createdSpotLimitOrdersCids,
		FailedSpotOrdersCids:                 failedSpotLimitOrdersCids,
		CreatedDerivativeOrdersCids:          createdDerivativeOrdersCids,
		FailedDerivativeOrdersCids:           failedDerivativeOrdersCids,
		CreatedBinaryOptionsOrdersCids:       createdBinaryOptionsOrdersCids,
		FailedBinaryOptionsOrdersCids:        failedBinaryOptionsOrdersCids,
		SpotMarketOrderHashes:                spotMarketOrderHashes,
		CreatedSpotMarketOrdersCids:          createdSpotMarketOrdersCids,
		FailedSpotMarketOrdersCids:           failedSpotMarketOrdersCids,
		DerivativeMarketOrderHashes:          derivativeMarketOrderHashes,
		CreatedDerivativeMarketOrdersCids:    createdDerivativeMarketOrdersCids,
		FailedDerivativeMarketOrdersCids:     failedDerivativeMarketOrdersCids,
		BinaryOptionsMarketOrderHashes:       binaryOptionsMarketOrderHashes,
		CreatedBinaryOptionsMarketOrdersCids: createdBinaryOptionsMarketOrdersCids,
		FailedBinaryOptionsMarketOrdersCids:  failedBinaryOptionsMarketOrdersCids,
	}, nil
}

// ProcessExpiredDOrders processes all expired orders at the current block height
func (k *Keeper) ProcessExpiredOrders(ctx sdk.Context) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessExpiredOrders")()

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

	marketFinder := marketfinder.New(k.BaseKeeper)

	for _, marketID := range marketIDs {
		market, err := marketFinder.FindMarket(ctx, marketID.Hex())
		if err != nil {
			ctx.Logger().Error("failed to find market with GTB orders", "error", err, "marketID", marketID)
			continue
		}
		k.processMarketExpiredOrders(ctx, market, blockHeight)
	}
}

func (k *Keeper) processMarketExpiredOrders(ctx sdk.Context, market v2.MarketI, blockHeight int64) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processMarketExpiredOrders")()

	k.IterateOrderExpirationEntries(ctx, market.MarketID(), blockHeight, func(orderHashKey []byte, value []byte) bool {
		order, err := k.UnmarshalOrderData(value)
		if err != nil {
			ctx.Logger().Error(
				"failed to decode expired order, deleting malformed expiration entry",
				"error", err,
				"marketID", market.MarketID(),
				"blockHeight", blockHeight,
				"orderHashKey", fmt.Sprintf("0x%x", orderHashKey),
			)
			k.DeleteOrderExpirationByKey(ctx, market.MarketID(), blockHeight, orderHashKey)
			return false
		}

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
			if err := k.CancelDerivativeOrder(
				ctx,
				common.HexToHash(order.SubaccountId),
				order.GetIdentifier(),
				market.(v2.DerivativeMarketI),
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

		k.DeleteOrderExpirationByKey(ctx, market.MarketID(), blockHeight, orderHashKey)
		return false
	})

	k.DeleteMarketWithOrderExpirations(ctx, market.MarketID(), blockHeight)
}

func (k *Keeper) processCancelAllSpotOrders(
	ctx sdk.Context,
	spotMarketIDsToCancelAll []string,
	subaccountID common.Hash,
	spotMarkets map[common.Hash]*v2.SpotMarket,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processCancelAllSpotOrders")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "processCancelAllDerivativeOrders")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "processCancelAllBinaryOptionsOrders")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "processCancelSpotOrders")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "processCancelDerivativeOrders")()

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

		err := k.CancelDerivativeOrder(
			ctx, subaccountID, derivativeOrderToCancel.GetIdentifier(), market, marketID, derivativeOrderToCancel.OrderMask,
		)

		if err == nil {
			derivativeCancelSuccesses[idx] = true
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
}

//revive:disable:cognitive-complexity // The complexity is acceptable and creating more helper functions would make the code less readable
func (k *Keeper) processCancelBinaryOptionsOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	binaryOptionsOrdersToCancel []*v2.OrderData,
	binaryOptionsCancelSuccesses []bool,
	binaryOptionsMarkets map[common.Hash]*v2.BinaryOptionsMarket,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processCancelBinaryOptionsOrders")()

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

		err := k.CancelDerivativeOrder(
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
	createdSpotOrdersCids *[]string,
	failedSpotOrdersCids *[]string,
	orderFailEvent *v2.EventOrderFail,
	spotMarkets map[common.Hash]*v2.SpotMarket,
	orderCreator func(ctx sdk.Context, sender sdk.AccAddress, order *v2.SpotOrder, market *v2.SpotMarket) (common.Hash, error),
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processCreateSpotOrders")()

	for idx, spotOrder := range spotOrdersToCreate {
		marketID := common.HexToHash(spotOrder.MarketId)
		market := k.getSpotMarketForOrder(ctx, marketID, spotMarkets)
		if market == nil {
			continue
		}
		if !market.IsActive() {
			k.Logger(ctx).Debug("failed to create spot order for non-active market", "marketID", marketID.Hex())
			continue
		}

		k.processSpotOrderCreation(
			ctx,
			sender,
			spotOrder,
			market,
			idx,
			spotOrderHashes,
			orderFailEvent,
			createdSpotOrdersCids,
			failedSpotOrdersCids,
			orderCreator,
		)
	}
}

func (k *Keeper) getSpotMarketForOrder(
	ctx sdk.Context,
	marketID common.Hash,
	spotMarkets map[common.Hash]*v2.SpotMarket,
) *v2.SpotMarket {
	defer k.Meter(ctx).FuncTiming(&ctx, "getSpotMarketForOrder")()

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
	orderCreator func(ctx sdk.Context, sender sdk.AccAddress, order *v2.SpotOrder, market *v2.SpotMarket) (common.Hash, error),
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processSpotOrderCreation")()

	if orderHash, err := orderCreator(ctx, sender, spotOrder, market); err != nil {
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
	orderCreator func(
		ctx sdk.Context, sender sdk.AccAddress, order *v2.DerivativeOrder, market v2.DerivativeMarketI, markPrice math.LegacyDec,
	) (common.Hash, error),
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processCreateDerivativeOrders")()

	for idx, derivativeOrder := range derivativeOrdersToCreate {
		marketID := derivativeOrder.MarketID()

		market, markPrice := k.getDerivativeMarketForOrder(ctx, marketID, derivativeMarkets, markPrices)
		if market == nil {
			continue
		}

		k.processDerivativeOrderCreation(
			ctx,
			sender,
			derivativeOrder,
			market,
			markPrice,
			idx,
			derivativeOrderHashes,
			orderFailEvent,
			createdDerivativeOrdersCids,
			failedDerivativeOrdersCids,
			orderCreator,
		)
	}
}

func (k *Keeper) getDerivativeMarketForOrder(
	ctx sdk.Context,
	marketID common.Hash,
	derivativeMarkets map[common.Hash]*v2.DerivativeMarket,
	markPrices map[common.Hash]math.LegacyDec,
) (*v2.DerivativeMarket, math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "getDerivativeMarketForOrder")()

	var market *v2.DerivativeMarket
	var markPrice math.LegacyDec

	if m, ok := derivativeMarkets[marketID]; ok {
		market = m
	} else {
		market, markPrice = k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
		if market == nil {
			k.Logger(ctx).Debug("failed to create derivative order for non-existent market", "marketID", marketID.Hex())
			return nil, math.LegacyDec{}
		}
		derivativeMarkets[marketID] = market
		markPrices[marketID] = markPrice
	}

	if !market.IsActive() {
		k.Logger(ctx).Debug("failed to create derivative orders for non-active market", "marketID", marketID.Hex())
		return nil, math.LegacyDec{}
	}

	if _, ok := markPrices[marketID]; !ok {
		price, err := k.GetDerivativeMarketPrice(
			ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType,
		)
		if err != nil {
			k.Logger(ctx).Debug("failed to create derivative order for market with no mark price", "marketID", marketID.Hex())

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
	orderCreator func(
		ctx sdk.Context, sender sdk.AccAddress, order *v2.DerivativeOrder, market v2.DerivativeMarketI, markPrice math.LegacyDec,
	) (common.Hash, error),
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processDerivativeOrderCreation")()

	if orderHash, err := orderCreator(ctx, sender, derivativeOrder, market, markPrice); err != nil {
		// Evict cross-margin snapshot cache for this subaccount on failure.
		// ReserveDerivativeOrderMargin (inside EnsureValidDerivativeOrder) may have updated
		// the cached OLR before the order failed at a later stage (e.g. atomic access check,
		// atomic execution). Without eviction, subsequent orders in the same batch would see
		// an inflated OLR and may be incorrectly rejected. This is a no-op for isolated
		// margin accounts or when there is no cached snapshot.
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, derivativeOrder.OrderInfo.SubaccountId)
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)

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
	orderCreator func(
		ctx sdk.Context, sender sdk.AccAddress, order *v2.DerivativeOrder, market v2.DerivativeMarketI, markPrice math.LegacyDec,
	) (common.Hash, error),
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processCreateBinaryOptionsOrders")()

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
			orderCreator,
		)
	}
}

func (k *Keeper) getBinaryOptionsMarketForOrder(
	ctx sdk.Context,
	marketID common.Hash,
	binaryOptionsMarkets map[common.Hash]*v2.BinaryOptionsMarket,
) *v2.BinaryOptionsMarket {
	defer k.Meter(ctx).FuncTiming(&ctx, "getBinaryOptionsMarketForOrder")()

	if m, ok := binaryOptionsMarkets[marketID]; ok {
		return m
	}

	market := k.GetBinaryOptionsMarket(ctx, marketID, true)
	if market == nil {
		k.Logger(ctx).Debug("failed to create binary options order for non-existent market", "marketID", marketID.Hex())
		return nil
	}

	if !market.IsActive() {
		k.Logger(ctx).Debug("failed to create binary options order for non-active market", "marketID", marketID.Hex())
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
	orderCreator func(
		ctx sdk.Context, sender sdk.AccAddress, order *v2.DerivativeOrder, market v2.DerivativeMarketI, markPrice math.LegacyDec,
	) (common.Hash, error),
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processBinaryOptionsOrderCreation")()

	if orderHash, err := orderCreator(ctx, sender, order, market, math.LegacyDec{}); err != nil {
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

func (k *Keeper) createDerivativeMarketOrderWithoutResultsForAtomicExecution(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market v2.DerivativeMarketI,
	markPrice math.LegacyDec,
) (orderHash common.Hash, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "createDerivativeMarketOrderWithoutResultsForAtomicExecution")()

	orderHash, _, err = k.CreateDerivativeMarketOrder(ctx, sender, derivativeOrder, market, markPrice)
	return orderHash, err
}

func (k *Keeper) IsGovernanceAuthorityAddress(address string) bool {
	return address == k.authority
}

func (k *Keeper) IsAdmin(ctx sdk.Context, addr string) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "IsAdmin")()

	for _, adminAddress := range k.GetCachedParams(ctx).ExchangeAdmins {
		if adminAddress == addr {
			return true
		}
	}
	return false
}

func (k *Keeper) IsFixedGasEnabled() bool {
	return k.fixedGas
}

func (k *Keeper) SetFixedGasEnabled(enabled bool) {
	k.fixedGas = enabled
}

// GetAllPerpetualMarketFundingStates returns all perpetual market funding states
func (k *Keeper) GetAllPerpetualMarketFundingStates(ctx sdk.Context) []v2.PerpetualMarketFundingState {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllPerpetualMarketFundingStates")()

	fundingStates := make([]v2.PerpetualMarketFundingState, 0)
	k.IteratePerpetualMarketFundings(ctx, func(p *v2.PerpetualMarketFunding, marketID common.Hash) (stop bool) {
		fundingState := v2.PerpetualMarketFundingState{
			MarketId: marketID.Hex(),
			Funding:  p,
		}
		fundingStates = append(fundingStates, fundingState)
		return false
	})

	return fundingStates
}

func (k *Keeper) checkDenomMinNotional(ctx sdk.Context, sender sdk.AccAddress, denom string, minNotional math.LegacyDec) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "checkDenomMinNotional")()
	// governance and exchange admins can set any min notional values
	if sender.String() == k.authority {
		return nil
	}

	if k.IsAdmin(ctx, sender.String()) {
		return nil
	}

	if !k.HasMinNotionalForDenom(ctx, denom) {
		return types.ErrInvalidNotional.Wrapf("min notional for %s does not exist", denom)
	}

	denomMinNotional := k.GetMinNotionalForDenom(ctx, denom)
	if minNotional.LT(denomMinNotional) {
		return types.ErrInvalidNotional.Wrapf("must be GTE %s", denomMinNotional)
	}

	return nil
}

func (k *Keeper) checkIfMarketLaunchProposalExist(
	ctx sdk.Context,
	marketID common.Hash,
	proposalTypes ...string,
) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "checkIfMarketLaunchProposalExist")()

	exists := false
	params, _ := k.govKeeper.Params.Get(ctx)
	// Note: we do 10 * voting period to iterate all active proposals safely
	endTime := ctx.BlockTime().Add(10 * (*params.VotingPeriod))
	rng := collections.NewPrefixUntilPairRange[time.Time, uint64](endTime)
	_ = k.govKeeper.ActiveProposalsQueue.Walk(ctx, rng, func(key collections.Pair[time.Time, uint64], _ uint64) (bool, error) {
		p, err := k.govKeeper.Proposals.Get(ctx, key.K2())
		if err != nil {
			return false, err
		}

		exists = utils.ProposalAlreadyExists(p, marketID, proposalTypes...)
		return exists, nil
	})

	return exists
}

func (k *Keeper) GetMarketType(ctx sdk.Context, marketID common.Hash, isEnabled bool) (*types.MarketType, error) { //nolint:revive // ok
	defer k.Meter(ctx).FuncTiming(&ctx, "GetMarketType")()

	if k.HasSpotMarket(ctx, marketID, isEnabled) {
		tp := types.MarketType_Spot
		return &tp, nil
	}

	if k.HasDerivativeMarket(ctx, marketID, isEnabled) {
		derivativeMarket := k.GetDerivativeMarket(ctx, marketID, isEnabled)
		tp := derivativeMarket.GetMarketType()
		return &tp, nil
	}

	if k.HasBinaryOptionsMarket(ctx, marketID, isEnabled) {
		tp := types.MarketType_BinaryOption
		return &tp, nil
	}

	return nil, types.ErrMarketInvalid.Wrapf("Market with id: %v doesn't exist or is not active", marketID)
}
