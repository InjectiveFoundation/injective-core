package wasm

import (
	"bytes"
	"encoding/json"
	"slices"
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/base"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/derivative"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/subaccount"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/risk"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

type WasmKeeper struct { //nolint:revive // ok
	*base.BaseKeeper

	bank       bankkeeper.Keeper
	subaccount *subaccount.SubaccountKeeper
	derivative *derivative.DerivativeKeeper
	wasmv      types.WasmViewKeeper
	wasmx      types.WasmxExecutionKeeper
}

func New(
	b *base.BaseKeeper,
	bk bankkeeper.Keeper,
	sk *subaccount.SubaccountKeeper,
	d *derivative.DerivativeKeeper,
	wv types.WasmViewKeeper,
	wx types.WasmxExecutionKeeper,
) *WasmKeeper {
	return &WasmKeeper{
		BaseKeeper: b,
		bank:       bk,
		subaccount: sk,
		derivative: d,
		wasmv:      wv,
		wasmx:      wx,
	}
}

func (k WasmKeeper) PrivilegedExecuteContractWithVersion(
	ctx sdk.Context,
	msg *v2.MsgPrivilegedExecuteContract,
	exchangeTypeVersion types.ExchangeTypeVersion,
) (*v2.MsgPrivilegedExecuteContractResponse, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "PrivilegedExecuteContractWithVersion")()

	k.Logger(ctx).Debug("=============== ⭐️ [Start] PrivilegedExecuteContract ⭐️ ===============")

	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	contract, _ := sdk.AccAddressFromBech32(msg.ContractAddress)

	fundsBefore, totalFunds, err := k.handleFundsTransfer(ctx, msg, sender, contract)
	if err != nil {
		return nil, err
	}

	err = k.executeContractAndHandleAction(ctx, contract, sender, totalFunds, msg.Data, exchangeTypeVersion)
	if err != nil {
		return nil, err
	}

	filteredFundsDiff := k.calculateFundsDifference(ctx, sender, fundsBefore)

	k.Logger(ctx).Debug("=============== 🛏️ [End] Exec 🛏️ ===============")
	return &v2.MsgPrivilegedExecuteContractResponse{FundsDiff: filteredFundsDiff}, nil
}

func (k WasmKeeper) handleFundsTransfer(
	ctx sdk.Context,
	msg *v2.MsgPrivilegedExecuteContract,
	sender,
	contract sdk.AccAddress,
) (fundsBefore, totalFunds sdk.Coins, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleFundsTransfer")()

	fundsBefore = sdk.Coins(make([]sdk.Coin, 0, len(msg.Funds)))
	totalFunds = sdk.Coins{}

	// Enforce sender has sufficient funds for execution
	if !msg.HasEmptyFunds() {
		coins, err := sdk.ParseCoinsNormalized(msg.Funds)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse coins %s", msg.Funds)
		}

		for _, coin := range coins {
			coinBefore := k.bank.GetBalance(ctx, sender, coin.Denom)
			fundsBefore = fundsBefore.Add(coinBefore)
		}

		// No need to check if receiver is a blocked address because it could never be a module account
		if err := k.bank.SendCoins(ctx, sender, contract, coins); err != nil {
			return nil, nil, errors.Wrap(err, "failed to send coins")
		}
		totalFunds = coins
	}

	return fundsBefore, totalFunds, nil
}

func (k WasmKeeper) executeContractAndHandleAction(
	ctx sdk.Context, contract, sender sdk.AccAddress, totalFunds sdk.Coins, data string, exchangeTypeVersion types.ExchangeTypeVersion,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "executeContractAndHandleAction")()

	execMsg, err := wasmxtypes.NewInjectiveExecMsg(sender, data)
	if err != nil {
		return errors.Wrap(err, "failed to create exec msg")
	}

	res, err := k.wasmx.InjectiveExec(ctx, contract, totalFunds, execMsg)
	if err != nil {
		return errors.Wrap(err, "failed to execute msg")
	}

	action, err := types.ParseRequest(res)
	if err != nil {
		return errors.Wrap(err, "failed to execute msg")
	}

	if action != nil {
		err = k.HandlePrivilegedAction(ctx, contract, sender, action, exchangeTypeVersion)
		if err != nil {
			return errors.Wrap(err, "failed to execute msg")
		}
	}

	return nil
}

func (k WasmKeeper) HandlePrivilegedAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action types.InjectiveAction,
	exchangeTypeVersion types.ExchangeTypeVersion,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "HandlePrivilegedAction")()

	switch t := action.(type) {
	case *types.SyntheticTradeAction:
		return k.handleSyntheticTradePrivilegedAction(ctx, contractAddress, origin, t, exchangeTypeVersion)
	case *types.PositionTransfer:
		return k.HandlePositionTransferAction(ctx, contractAddress, origin, t)
	default:
		return types.ErrUnsupportedAction
	}
}

func (k WasmKeeper) handleSyntheticTradePrivilegedAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action *types.SyntheticTradeAction,
	exchangeTypeVersion types.ExchangeTypeVersion,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleSyntheticTradePrivilegedAction")()

	if exchangeTypeVersion == types.ExchangeTypeVersionV1 {
		newContractTrades, err := k.ConvertSyntheticTradesV1ToV2(ctx, action.ContractTrades)
		if err != nil {
			return err
		}

		newUserTrades, err := k.ConvertSyntheticTradesV1ToV2(ctx, action.UserTrades)
		if err != nil {
			return err
		}

		action.ContractTrades = newContractTrades
		action.UserTrades = newUserTrades
	}

	return k.HandleSyntheticTradeAction(ctx, contractAddress, origin, action)
}

func (k WasmKeeper) ConvertSyntheticTradesV1ToV2(
	ctx sdk.Context,
	trades []*types.SyntheticTrade,
) ([]*types.SyntheticTrade, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ConvertSyntheticTradesV1ToV2")()

	v2Trades := make([]*types.SyntheticTrade, 0, len(trades))
	for _, trade := range trades {
		derivativeMarket := k.derivative.GetDerivativeMarketByID(ctx, trade.MarketID)
		if derivativeMarket == nil {
			return nil, errors.Wrap(types.ErrDerivativeMarketNotFound, "failed to convert trade type to v2")
		}

		v2Trades = append(v2Trades, &types.SyntheticTrade{
			MarketID:     trade.MarketID,
			SubaccountID: trade.SubaccountID,
			IsBuy:        trade.IsBuy,
			Quantity:     trade.Quantity,
			Price:        derivativeMarket.PriceFromChainFormat(trade.Price),
			Margin:       derivativeMarket.NotionalFromChainFormat(trade.Margin),
		})
	}

	return v2Trades, nil
}

type capState struct {
	openNotionalCap   v2.OpenNotionalCap
	currOpenNotional  math.LegacyDec
	addedOpenNotional math.LegacyDec
	openInterestDelta math.LegacyDec
	posQty            map[common.Hash]math.LegacyDec
}

func (cs *capState) initSignedQty(subID common.Hash, pos *v2.Position) {
	if pos == nil || pos.Quantity.IsZero() {
		cs.posQty[subID] = math.LegacyZeroDec()
		return
	}

	if pos.IsLong {
		qty := pos.Quantity
		cs.posQty[subID] = qty
		return
	}

	neg := pos.Quantity.Neg()
	cs.posQty[subID] = neg
}

func (k WasmKeeper) HandlePositionTransferAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action *types.PositionTransfer,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "HandlePositionTransferAction")()

	if k.IsPostOnlyMode(ctx) {
		return types.ErrPostOnlyMode.Wrap("position transfers are not allowed in post-only mode")
	}

	// Check cross-margin emergency pause for both sides of the transfer.
	if err := k.derivative.RiskEngine().CheckCrossMarginEmergencyPause(ctx, action.SourceSubaccountID); err != nil {
		return err
	}
	if err := k.derivative.RiskEngine().CheckCrossMarginEmergencyPause(ctx, action.DestinationSubaccountID); err != nil {
		return err
	}

	m := k.derivative.GetDerivativeMarketInfo(ctx, action.MarketID, true)

	var (
		market    = m.Market
		markPrice = m.MarkPrice
		funding   = m.Funding
	)

	if err := ensureActiveDerivativeMarket(market, markPrice, action.MarketID); err != nil {
		return err
	}

	sourcePosition := k.GetPosition(ctx, action.MarketID, action.SourceSubaccountID)
	destinationPosition := k.GetPosition(ctx, action.MarketID, action.DestinationSubaccountID)

	// For cross-margin destinations, enforce eligibility (enabled denom, market type, active
	// market cap) unless the transfer strictly reduces the destination's exposure. A transfer
	// is strictly reducing when the destination has an opposite-side position AND the transferred
	// quantity doesn't exceed it (no flip into new net exposure). This preserves wind-down paths
	// while preventing position-flipping bypasses.
	isStrictlyReducingForDest := destinationPosition != nil && !destinationPosition.Quantity.IsZero() &&
		sourcePosition != nil && sourcePosition.IsLong != destinationPosition.IsLong &&
		action.Quantity.LTE(destinationPosition.Quantity)
	if !isStrictlyReducingForDest {
		if err := k.ensureCrossMarginEligibility(ctx, action.DestinationSubaccountID, market, action.MarketID); err != nil {
			return err
		}
	}

	// Per-position margin checks must be skipped for cross-margin sides (position.Margin
	// is accounting state — pool-level equity covers the position) but preserved for
	// isolated sides in mixed-mode transfers.
	sourceProfile, _ := k.derivative.RiskEngine().EffectiveProfile(ctx, action.SourceSubaccountID)
	destProfile, _ := k.derivative.RiskEngine().EffectiveProfile(ctx, action.DestinationSubaccountID)
	sourceIsCross := sourceProfile != nil && sourceProfile.Mode == v2.RiskMode_RISK_MODE_CROSS
	destIsCross := destProfile != nil && destProfile.Mode == v2.RiskMode_RISK_MODE_CROSS

	destinationPosition, err := preparePositionTransfer(
		contractAddress,
		origin,
		action,
		sourcePosition,
		destinationPosition,
		funding,
		market,
		markPrice,
		sourceIsCross,
		destIsCross,
	)
	if err != nil {
		return err
	}

	oiDelta := calcPositionTransferOpenInterestDelta(destinationPosition, sourcePosition, action.Quantity)

	executionPrice := sourcePosition.EntryPrice
	sourceMarginBefore := sourcePosition.Margin

	isSourceLongBefore := DirLong
	if !sourcePosition.IsLong {
		isSourceLongBefore = DirShort
	}
	isDestinationLongBefore := DirLong
	if !destinationPosition.IsLong {
		isDestinationLongBefore = DirShort
	}

	// Ignore payouts when applying position delta in source position, because margin + PNL is accounted for in destination position
	payout, closeExecutionMargin := applyPositionTransferDeltas(sourcePosition, destinationPosition, action.Quantity, executionPrice, sourceMarginBefore)

	receiverTradingFee := markPrice.Mul(action.Quantity).Mul(market.TakerFeeRate)

	sourceDepositBefore := *k.GetDeposit(ctx, action.SourceSubaccountID, market.QuoteDenom)
	destinationDepositBefore := *k.GetDeposit(ctx, action.DestinationSubaccountID, market.QuoteDenom)

	actionCtx, writeAction := ctx.CacheContext()
	ctx = actionCtx

	if err := k.applyPositionTransferMarketBalanceDelta(ctx, action.MarketID, market, payout, closeExecutionMargin); err != nil {
		return err
	}

	k.derivative.SavePosition(ctx, action.MarketID, action.SourceSubaccountID, sourcePosition)
	k.derivative.SavePosition(ctx, action.MarketID, action.DestinationSubaccountID, destinationPosition)

	k.applyPositionTransferDeposits(ctx, action, market, payout, closeExecutionMargin, receiverTradingFee)

	// Evict cached cross-pool snapshots for both sides — positions and deposits changed.
	k.derivative.RiskEngine().EvictCrossPoolSnapshotCache(ctx, action.SourceSubaccountID)
	k.derivative.RiskEngine().EvictCrossPoolSnapshotCache(ctx, action.DestinationSubaccountID)

	// Verify cross-margin pool health for both sides after the transfer.
	// Losing a profitable position can push the source pool below maintenance;
	// receiving a thin-margin position can push the destination pool below maintenance.
	if err := k.ensureCrossMarginPoolHealth(ctx, action.SourceSubaccountID, market.QuoteDenom, market.QuoteDecimals); err != nil {
		return err
	}
	// Skip destination health check for strictly-reducing transfers. A reducing transfer
	// nets the destination's opposite-side position, analogous to a reduce-only close in
	// the normal FBA path which has no post-execution health requirement. Requiring full
	// health restoration here would block legitimate wind-down flows for distressed pools.
	if !isStrictlyReducingForDest {
		if err := k.ensureCrossMarginPoolHealth(ctx, action.DestinationSubaccountID, market.QuoteDenom, market.QuoteDecimals); err != nil {
			return err
		}
	}

	k.applyOpenInterestDeltaIfNeeded(ctx, action.MarketID, oiDelta)

	if err := k.checkAndResolveReduceOnlyConflicts(ctx, action.MarketID, action.SourceSubaccountID, sourcePosition, !sourcePosition.IsLong); err != nil {
		return err
	}

	if err := k.resolvePositionTransferDestinationReduceOnlyEffects(
		ctx,
		action,
		destinationPosition,
		isSourceLongBefore,
		isDestinationLongBefore,
	); err != nil {
		return err
	}

	if !sourceIsCross {
		sourceDepositAfter := *k.GetDeposit(ctx, action.SourceSubaccountID, market.QuoteDenom)
		if err := ensurePositionTransferDepositNotMoreNegative("source", action.SourceSubaccountID, market.QuoteDenom, sourceDepositBefore, sourceDepositAfter); err != nil {
			return err
		}
	}
	if !destIsCross {
		destinationDepositAfter := *k.GetDeposit(ctx, action.DestinationSubaccountID, market.QuoteDenom)
		if err := ensurePositionTransferDepositNotMoreNegative("destination", action.DestinationSubaccountID, market.QuoteDenom, destinationDepositBefore, destinationDepositAfter); err != nil {
			return err
		}
	}

	events.Emit(ctx, k.BaseKeeper, &v2.EventPositionTransfer{
		MarketId:                action.MarketID.Hex(),
		SourceSubaccountId:      action.SourceSubaccountID.Hex(),
		DestinationSubaccountId: action.DestinationSubaccountID.Hex(),
		Quantity:                action.Quantity,
	})

	writeAction()
	return nil
}

func ensurePositionTransferDepositNotMoreNegative(
	side string,
	subaccountID common.Hash,
	denom string,
	before, after v2.Deposit,
) error {
	if !depositBecameMoreNegative(before, after) {
		return nil
	}

	return errors.Wrapf(
		types.ErrInsufficientDeposit,
		"%s subaccount %s deposit in denom %s became more negative",
		side,
		subaccountID.Hex(),
		denom,
	)
}

func depositBecameMoreNegative(before, after v2.Deposit) bool {
	return balanceBecameMoreNegative(before.AvailableBalance, after.AvailableBalance) ||
		balanceBecameMoreNegative(before.TotalBalance, after.TotalBalance)
}

func balanceBecameMoreNegative(before, after math.LegacyDec) bool {
	return after.IsNegative() && after.LT(before)
}

type Direction uint8

const (
	DirLong Direction = iota
	DirShort
)

func (k WasmKeeper) resolvePositionTransferDestinationReduceOnlyEffects(
	ctx sdk.Context,
	action *types.PositionTransfer,
	destinationPosition *v2.Position,
	sourceDirBefore Direction,
	destDirBefore Direction,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "resolvePositionTransferDestinationReduceOnlyEffects")()

	if sourceDirBefore == destDirBefore {
		return nil
	}

	destWasLong := destDirBefore == DirLong

	// if destination position flipped or is closed, cancel all RO orders
	if destWasLong != destinationPosition.IsLong || destinationPosition.Quantity.IsZero() {
		metadata := k.GetSubaccountOrderbookMetadata(ctx, action.MarketID, action.DestinationSubaccountID, !destWasLong)
		return k.cancelAllReduceOnlyOrders(ctx, action.MarketID, action.DestinationSubaccountID, metadata, !destWasLong)
	}

	// partial closing case
	return k.checkAndResolveReduceOnlyConflicts(ctx, action.MarketID, action.DestinationSubaccountID, destinationPosition, !destinationPosition.IsLong)
}

func (k WasmKeeper) applyOpenInterestDeltaIfNeeded(ctx sdk.Context, marketID common.Hash, oiDelta math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "applyOpenInterestDeltaIfNeeded")()

	if oiDelta.IsZero() {
		return
	}
	k.ApplyOpenInterestDeltaForMarket(ctx, marketID, oiDelta)
}

func (k WasmKeeper) checkAndResolveReduceOnlyConflicts(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	position *v2.Position,
	isReduceOnlyDirectionBuy bool,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "checkAndResolveReduceOnlyConflicts")()

	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isReduceOnlyDirectionBuy)

	if metadata.ReduceOnlyLimitOrderCount == 0 {
		return nil
	}

	if position.Quantity.IsZero() {
		return k.cancelAllReduceOnlyOrders(ctx, marketID, subaccountID, metadata, isReduceOnlyDirectionBuy)
	}

	cumulativeOrderSideQuantity := metadata.AggregateReduceOnlyQuantity.Add(metadata.AggregateVanillaQuantity)

	maxRoQuantityToCancel := cumulativeOrderSideQuantity.Sub(position.Quantity)
	if maxRoQuantityToCancel.IsNegative() || maxRoQuantityToCancel.IsZero() {
		return nil
	}

	subaccountEOBResults := v2.NewSubaccountOrderResults()
	return k.derivative.CancelMinimumReduceOnlyOrders(
		ctx,
		marketID,
		subaccountID,
		metadata,
		isReduceOnlyDirectionBuy,
		position.Quantity,
		subaccountEOBResults,
		nil,
	)
}

func (k WasmKeeper) cancelAllReduceOnlyOrders(
	ctx sdk.Context,
	marketID,
	subaccountID common.Hash,
	metadata *v2.SubaccountOrderbookMetadata,
	isBuy bool,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "cancelAllReduceOnlyOrders")()

	if metadata.ReduceOnlyLimitOrderCount == 0 {
		return nil
	}

	orders := k.subaccount.GetWorstReduceOnlySubaccountOrdersUpToCount(
		ctx,
		marketID,
		subaccountID,
		isBuy,
		&metadata.ReduceOnlyLimitOrderCount,
	)

	return k.derivative.CancelReduceOnlyOrders(ctx, marketID, subaccountID, metadata, isBuy, orders)
}

func (k WasmKeeper) HandleSyntheticTradeAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action *types.SyntheticTradeAction,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "HandleSyntheticTradeAction")()

	if k.IsPostOnlyMode(ctx) {
		return types.ErrPostOnlyMode.Wrap("synthetic trades are not allowed in post-only mode")
	}

	summary, err := action.Summarize()
	if err != nil {
		return err
	}

	// Enforce that subaccountIDs provided match either the contract address or the origin address
	if err := ensureSyntheticTradeParties(contractAddress, origin, summary.ContractAddress, summary.UserAddress); err != nil {
		return err
	}

	return k.processSyntheticTradeAction(ctx, contractAddress, summary.GetMarketIDs(), action)
}

// syntheticPoolKey identifies a cross-margin pool by (subaccount, quoteDenom). Used to
// track which pools need post-batch health checks after synthetic trade execution.
type syntheticPoolKey struct {
	subaccountID common.Hash
	quoteDenom   string
}

type syntheticEventKey struct {
	marketID string
	isBuy    bool
}

type syntheticEventData struct {
	cumulativeFunding *math.LegacyDec
	trades            []*v2.DerivativeTradeLog
}

type syntheticFundingVwapByMarket map[common.Hash]*v2.VwapData

func (k WasmKeeper) processSyntheticTradeAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	marketIDs []common.Hash,
	action *types.SyntheticTradeAction,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "processSyntheticTradeAction")()

	totalMarginAndFees := make(map[string]math.LegacyDec)
	totalFees := make(map[string]math.LegacyDec)
	markets := make(map[common.Hash]*v2.DerivativeMarketInfo)
	caps := make(map[common.Hash]*capState)

	if err := k.initSyntheticTradeState(ctx, marketIDs, markets, totalMarginAndFees, totalFees, caps); err != nil {
		return err
	}

	initialPositions := v2.NewModifiedPositionCache()
	finalPositions := v2.NewModifiedPositionCache()

	trades := append(append([]*types.SyntheticTrade{}, action.UserTrades...), action.ContractTrades...)

	eventGroups := make(map[syntheticEventKey]*syntheticEventData)

	// Track (subaccount, quoteDenom) pools that had at least one non-reducing trade.
	// Only these need a post-batch health check — pools with exclusively reducing trades
	// are analogous to reduce-only orders in the normal FBA path, which have no
	// post-execution health requirement. Keyed per pool so that a non-reducing trade in
	// one quote-denom pool does not cause a reducing-only unwind in another pool to be
	// rejected.
	nonReducingPools := make(map[syntheticPoolKey]struct{})

	for _, trade := range trades {
		result, err := k.applySyntheticTrade(
			ctx,
			markets,
			caps,
			initialPositions,
			finalPositions,
			totalMarginAndFees,
			totalFees,
			trade,
		)
		if err != nil {
			return err
		}

		if !result.isStrictlyReducing {
			m := markets[trade.MarketID]
			nonReducingPools[syntheticPoolKey{subaccountID: trade.SubaccountID, quoteDenom: m.Market.QuoteDenom}] = struct{}{}
		}

		key := syntheticEventKey{marketID: result.marketID, isBuy: result.isBuy}
		if _, exists := eventGroups[key]; !exists {
			eventGroups[key] = &syntheticEventData{
				cumulativeFunding: result.cumulativeFunding,
				trades:            make([]*v2.DerivativeTradeLog, 0),
			}
		}
		eventGroups[key].trades = append(eventGroups[key].trades, result.tradeLog)
	}

	// Transfer funds from the contract to exchange module to pay for the synthetic trades
	coinsToTransfer := buildCoinsToTransfer(totalMarginAndFees)
	if err := k.transferSyntheticTradeFunds(ctx, contractAddress, coinsToTransfer, totalFees); err != nil {
		return err
	}

	// Evict cached cross-pool snapshots for all affected subaccounts — positions and
	// deposits changed. Must happen before the health check so it builds fresh snapshots,
	// and so later same-block operations don't reuse stale cached equity/OLR.
	evictedSubaccounts := make(map[common.Hash]struct{})
	for _, trade := range trades {
		if _, done := evictedSubaccounts[trade.SubaccountID]; !done {
			k.derivative.RiskEngine().EvictCrossPoolSnapshotCache(ctx, trade.SubaccountID)
			evictedSubaccounts[trade.SubaccountID] = struct{}{}
		}
	}

	// Verify cross-margin pool health for pools that had at least one non-reducing trade.
	// Checked post-batch because synthetic trades come in matched pairs (user + contract)
	// and the pool should be validated after both sides are applied. Pools with only
	// reducing trades are skipped — they are unwinding exposure and must be allowed even
	// if the pool remains distressed, matching the normal FBA path.
	if err := k.ensureCrossMarginPoolHealthAfterSyntheticTrades(ctx, trades, markets, nonReducingPools); err != nil {
		return err
	}

	for _, marketID := range marketIDs {
		if err := k.resolveSyntheticTradeROConflictsForMarket(ctx, marketID, initialPositions, finalPositions); err != nil {
			return err
		}

		if cs := caps[marketID]; cs != nil && !cs.openInterestDelta.IsZero() {
			k.ApplyOpenInterestDeltaForMarket(ctx, marketID, cs.openInterestDelta)
		}
	}

	k.persistSyntheticTradeFundingVwap(ctx, markets, action.UserTrades)

	keys := make([]syntheticEventKey, 0, len(eventGroups))
	for key := range eventGroups {
		keys = append(keys, key)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		if keys[i].marketID != keys[j].marketID {
			return keys[i].marketID < keys[j].marketID
		}
		return !keys[i].isBuy && keys[j].isBuy
	})

	for _, key := range keys {
		data := eventGroups[key]
		events.Emit(ctx, k.BaseKeeper, &v2.EventBatchDerivativeExecution{
			MarketId:          key.marketID,
			IsBuy:             key.isBuy,
			IsLiquidation:     false,
			ExecutionType:     v2.ExecutionType_Synthetic,
			Trades:            data.trades,
			CumulativeFunding: data.cumulativeFunding,
		})
	}

	return nil
}

func (k WasmKeeper) persistSyntheticTradeFundingVwap(
	ctx sdk.Context,
	markets map[common.Hash]*v2.DerivativeMarketInfo,
	trades []*types.SyntheticTrade,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "persistSyntheticTradeFundingVwap")()

	vwapByMarket := buildSyntheticFundingVwapByMarket(markets, trades)

	for _, marketID := range sortedSyntheticFundingVwapMarketIDs(vwapByMarket) {
		k.persistSyntheticTradeFundingVwapForMarket(ctx, marketID, markets[marketID], vwapByMarket[marketID])
	}
}

func buildSyntheticFundingVwapByMarket(
	markets map[common.Hash]*v2.DerivativeMarketInfo,
	trades []*types.SyntheticTrade,
) syntheticFundingVwapByMarket {
	vwapByMarket := make(syntheticFundingVwapByMarket)

	for _, trade := range trades {
		marketInfo := markets[trade.MarketID]
		if !shouldPersistSyntheticFundingTrade(marketInfo, trade) {
			continue
		}

		vwapByMarket[trade.MarketID] = applySyntheticFundingTradeExecution(vwapByMarket[trade.MarketID], trade)
	}

	return vwapByMarket
}

func shouldPersistSyntheticFundingTrade(marketInfo *v2.DerivativeMarketInfo, trade *types.SyntheticTrade) bool {
	return marketInfo != nil &&
		marketInfo.Market != nil &&
		marketInfo.Market.IsPerpetual &&
		!trade.Quantity.IsZero()
}

func applySyntheticFundingTradeExecution(vwapData *v2.VwapData, trade *types.SyntheticTrade) *v2.VwapData {
	if vwapData == nil {
		vwapData = v2.NewVwapData()
	}

	return vwapData.ApplyExecution(trade.Price, trade.Quantity)
}

func sortedSyntheticFundingVwapMarketIDs(vwapByMarket syntheticFundingVwapByMarket) []common.Hash {
	marketIDs := make([]common.Hash, 0, len(vwapByMarket))
	for marketID := range vwapByMarket {
		marketIDs = append(marketIDs, marketID)
	}

	slices.SortStableFunc(marketIDs, func(a, b common.Hash) int {
		return bytes.Compare(a.Bytes(), b.Bytes())
	})

	return marketIDs
}

func (k WasmKeeper) persistSyntheticTradeFundingVwapForMarket(
	ctx sdk.Context,
	marketID common.Hash,
	marketInfo *v2.DerivativeMarketInfo,
	vwapData *v2.VwapData,
) {
	if marketInfo == nil || marketInfo.MarkPrice.IsNil() || marketInfo.MarkPrice.IsZero() || vwapData == nil || vwapData.Quantity.IsZero() {
		return
	}

	k.AccumulateSyntheticPerpetualFundingVwap(ctx, marketID, marketInfo.MarkPrice, vwapData.Price, vwapData.Quantity)
}

func ensureSyntheticTradeParties(
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	contractSubaccountAddress sdk.Address,
	userSubaccountAddress sdk.Address,
) error {
	if !contractAddress.Equals(contractSubaccountAddress) || !origin.Equals(userSubaccountAddress) {
		return errors.Wrapf(
			types.ErrBadSubaccountID,
			"subaccountID address %s does not match either contract address %s or origin address %s",
			userSubaccountAddress.String(),
			contractAddress.String(), origin.String(),
		)
	}
	return nil
}

func ensureActiveDerivativeMarket(market *v2.DerivativeMarket, markPrice math.LegacyDec, marketID common.Hash) error {
	if market == nil || markPrice.IsNil() {
		return errors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", marketID.Hex())
	}
	return nil
}

func (k WasmKeeper) ensureSyntheticTradeMarketSupported(ctx sdk.Context, marketID common.Hash) error {
	isEnabled := true
	market := k.derivative.GetDerivativeOrBinaryOptionsMarket(ctx, marketID, &isEnabled)
	if market != nil && market.GetMarketType() == types.MarketType_BinaryOption {
		return errors.Wrapf(types.ErrInvalidTrade, "synthetic trades do not support binary options markets: %s", marketID.Hex())
	}
	return nil
}

func preparePositionTransfer(
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action *types.PositionTransfer,
	sourcePosition *v2.Position,
	destinationPosition *v2.Position,
	funding *v2.PerpetualMarketFunding,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	sourceIsCross, destIsCross bool,
) (*v2.Position, error) {
	if err := ensurePositionTransferParties(contractAddress, origin, action.SourceSubaccountID, action.DestinationSubaccountID); err != nil {
		return nil, err
	}

	// Enforce that source position has sufficient quantity for transfer
	if err := ensurePositionTransferSourceHasSufficientQty(sourcePosition, action.Quantity); err != nil {
		return nil, err
	}

	destinationPosition = initDestinationPositionForTransfer(destinationPosition, sourcePosition, funding)

	if market.IsPerpetual {
		destinationPosition.ApplyFunding(funding)
		sourcePosition.ApplyFunding(funding)
	}

	// For cross-margin subaccounts, per-position margin is accounting state — pool-level
	// equity covers the position. Skip the per-position check for CM sides; the caller runs
	// ensureCrossMarginPoolHealth which validates solvency at the pool level. Isolated sides
	// must still pass the per-position maintenance margin check.
	if !sourceIsCross {
		if err := ensurePositionAboveMaintenanceMarginRatio(sourcePosition, market, markPrice); err != nil {
			return nil, err
		}
	}
	if !destIsCross {
		if err := ensurePositionAboveMaintenanceMarginRatio(destinationPosition, market, markPrice); err != nil {
			return nil, err
		}
	}

	return destinationPosition, nil
}

func ensurePositionTransferParties(
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	sourceSubaccountID common.Hash,
	destinationSubaccountID common.Hash,
) error {
	sourceAddress := types.SubaccountIDToSdkAddress(sourceSubaccountID)
	destinationAddress := types.SubaccountIDToSdkAddress(destinationSubaccountID)

	contractToUser := contractAddress.Equals(sourceAddress) && origin.Equals(destinationAddress)
	userToContract := origin.Equals(sourceAddress) && contractAddress.Equals(destinationAddress)

	if !contractToUser && !userToContract {
		return errors.Wrapf(
			types.ErrBadSubaccountID,
			"Invalid position transfer parties: source %s and destination %s must be a valid pair of contract address %s and origin address %s",
			sourceAddress.String(),
			destinationAddress.String(),
			contractAddress.String(),
			origin.String(),
		)
	}

	return nil
}

func ensurePositionTransferSourceHasSufficientQty(sourcePosition *v2.Position, quantity math.LegacyDec) error {
	if sourcePosition == nil || sourcePosition.Quantity.LT(quantity) {
		return errors.Wrapf(types.ErrInvalidQuantity, "Source subaccountID position quantity")
	}
	return nil
}

func initDestinationPositionForTransfer(
	destinationPosition *v2.Position,
	sourcePosition *v2.Position,
	funding *v2.PerpetualMarketFunding,
) *v2.Position {
	if destinationPosition != nil {
		return destinationPosition
	}

	var cumulativeFundingEntry math.LegacyDec
	if funding != nil {
		cumulativeFundingEntry = funding.CumulativeFunding
	}
	return v2.NewPosition(sourcePosition.IsLong, cumulativeFundingEntry)
}

func ensurePositionAboveMaintenanceMarginRatio(
	position *v2.Position,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
) error {
	if position.Quantity.IsPositive() {
		positionMarginRatio := position.GetEffectiveMarginRatio(markPrice, math.LegacyZeroDec())
		if positionMarginRatio.LT(market.MaintenanceMarginRatio) {
			return errors.Wrapf(
				types.ErrLowPositionMargin,
				"position margin ratio %s ≥ %s must hold", positionMarginRatio.String(), market.MaintenanceMarginRatio.String(),
			)
		}
	}
	return nil
}

func calcPositionTransferOpenInterestDelta(
	destinationPosition *v2.Position,
	sourcePosition *v2.Position,
	quantity math.LegacyDec,
) math.LegacyDec {
	oiDelta := math.LegacyZeroDec()
	if destinationPosition.Quantity.IsPositive() && (destinationPosition.IsLong != sourcePosition.IsLong) {
		minQuantity := math.LegacyMinDec(quantity, destinationPosition.Quantity)
		oiDeltaFromTrade := minQuantity.Mul(math.LegacyNewDec(2))
		oiDelta = oiDelta.Sub(oiDeltaFromTrade)
	}
	return oiDelta
}

func applyPositionTransferDeltas(
	sourcePosition *v2.Position,
	destinationPosition *v2.Position,
	quantity math.LegacyDec,
	executionPrice math.LegacyDec,
	sourceMarginBefore math.LegacyDec,
) (payout, closeExecutionMargin math.LegacyDec) {
	sourcePosition.ApplyPositionDelta(
		&v2.PositionDelta{
			IsLong:            !sourcePosition.IsLong,
			ExecutionQuantity: quantity,
			ExecutionMargin:   math.LegacyZeroDec(),
			ExecutionPrice:    executionPrice,
		},
		math.LegacyZeroDec(),
	)

	executionMargin := sourceMarginBefore.Sub(sourcePosition.Margin)
	payout, closeExecutionMargin, _, _ = destinationPosition.ApplyPositionDelta(
		&v2.PositionDelta{
			IsLong:            sourcePosition.IsLong,
			ExecutionQuantity: quantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    executionPrice,
		},
		math.LegacyZeroDec(),
	)

	return payout, closeExecutionMargin
}

// Special market balance handling for position transfers:
// - `collateralizationMargin` can be ignored because those funds came from the source position
// - `receiverTradingFee` can be ignored because its paid from user balances
// - `closeExecutionMargin` must be accounted for as those funds came from an existing position and are now leaving the market
func (k WasmKeeper) applyPositionTransferMarketBalanceDelta(
	ctx sdk.Context,
	marketID common.Hash,
	market *v2.DerivativeMarket,
	payout, closeExecutionMargin math.LegacyDec,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "applyPositionTransferMarketBalanceDelta")()

	marketBalanceDelta := payout.Add(closeExecutionMargin).Neg()
	chainFormattedMarketBalanceDelta := market.NotionalToChainFormat(marketBalanceDelta)

	availableMarketFunds := k.derivative.GetAvailableMarketFunds(ctx, marketID)
	isMarketSolvent := v2.IsMarketSolvent(availableMarketFunds, chainFormattedMarketBalanceDelta)
	if !isMarketSolvent {
		return types.ErrInsufficientMarketBalance
	}

	k.derivative.ApplyMarketBalanceDelta(ctx, marketID, chainFormattedMarketBalanceDelta)
	return nil
}

func (k WasmKeeper) applyPositionTransferDeposits(
	ctx sdk.Context,
	action *types.PositionTransfer,
	market *v2.DerivativeMarket,
	payout, closeExecutionMargin, receiverTradingFee math.LegacyDec,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "applyPositionTransferDeposits")()

	chainFormattedDepositDeltaAmount := market.NotionalToChainFormat(payout.Add(closeExecutionMargin).Sub(receiverTradingFee))
	chainFormattedReceiverTradingFee := market.NotionalToChainFormat(receiverTradingFee)

	depositDelta := types.NewUniformDepositDelta(chainFormattedDepositDeltaAmount)
	k.subaccount.UpdateDepositWithDelta(ctx, action.DestinationSubaccountID, market.QuoteDenom, depositDelta)
	k.subaccount.UpdateDepositWithDelta(
		ctx,
		types.AuctionSubaccountID,
		market.QuoteDenom,
		types.NewUniformDepositDelta(chainFormattedReceiverTradingFee),
	)
}

func (k WasmKeeper) initSyntheticTradeState(
	ctx sdk.Context,
	marketIDs []common.Hash,
	markets map[common.Hash]*v2.DerivativeMarketInfo,
	totalMarginAndFees map[string]math.LegacyDec,
	totalFees map[string]math.LegacyDec,
	caps map[common.Hash]*capState,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "initSyntheticTradeState")()

	for _, marketID := range marketIDs {
		if err := k.ensureSyntheticTradeMarketSupported(ctx, marketID); err != nil {
			return err
		}

		m := k.derivative.GetDerivativeMarketInfo(ctx, marketID, true)
		if m.Market == nil || m.MarkPrice.IsNil() {
			return errors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", marketID.Hex())
		}

		markets[marketID] = m
		totalMarginAndFees[m.Market.QuoteDenom] = math.LegacyZeroDec()
		totalFees[m.Market.QuoteDenom] = math.LegacyZeroDec()

		caps[marketID] = &capState{
			openNotionalCap:   m.Market.GetOpenNotionalCap(),
			currOpenNotional:  k.derivative.GetOpenNotionalForMarket(ctx, marketID, m.MarkPrice),
			addedOpenNotional: math.LegacyZeroDec(),
			openInterestDelta: math.LegacyZeroDec(),
			posQty:            make(map[common.Hash]math.LegacyDec),
		}
	}
	return nil
}

type syntheticTradeResult struct {
	marketID           string
	isBuy              bool
	cumulativeFunding  *math.LegacyDec
	tradeLog           *v2.DerivativeTradeLog
	isStrictlyReducing bool
}

func (k WasmKeeper) applySyntheticTrade(
	ctx sdk.Context,
	markets map[common.Hash]*v2.DerivativeMarketInfo,
	caps map[common.Hash]*capState,
	initialPositions v2.ModifiedPositionCache,
	finalPositions v2.ModifiedPositionCache,
	totalMarginAndFees map[string]math.LegacyDec,
	totalFees map[string]math.LegacyDec,
	trade *types.SyntheticTrade,
) (*syntheticTradeResult, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "applySyntheticTrade")()

	m := markets[trade.MarketID]
	market := m.Market
	markPrice := m.MarkPrice

	var fundingInfo *v2.PerpetualMarketFunding
	if market.IsPerpetual {
		fundingInfo = m.Funding
	}

	// Check cross-margin emergency pause before modifying positions.
	if err := k.derivative.RiskEngine().CheckCrossMarginEmergencyPause(ctx, trade.SubaccountID); err != nil {
		return nil, err
	}

	// Initialize position and apply funding
	position := k.GetPosition(ctx, trade.MarketID, trade.SubaccountID)

	// For cross-margin subaccounts, enforce eligibility (enabled denom, market type, active
	// market cap) unless the trade strictly reduces exposure. A trade is strictly reducing
	// only when it closes an existing opposite-side position without exceeding its size.
	// Note: trade.IsReduceOnly() (margin==0) alone is NOT sufficient — a zero-margin trade
	// on a fresh or same-side position would bypass eligibility and pool health checks.
	isOppositeAndWithinPosition := position != nil && !position.Quantity.IsZero() &&
		trade.IsBuy != position.IsLong && trade.Quantity.LTE(position.Quantity)
	isStrictlyReducing := isOppositeAndWithinPosition
	if !isStrictlyReducing {
		if err := k.ensureCrossMarginEligibility(ctx, trade.SubaccountID, market, trade.MarketID); err != nil {
			return nil, err
		}
	}

	position = initSyntheticTradePosition(position, trade.IsBuy, fundingInfo)

	cs := caps[trade.MarketID]
	cs.initSignedQty(trade.SubaccountID, position)

	recordInitialPositionIfNeeded(initialPositions, trade.MarketID, trade.SubaccountID, position)

	orderType := v2.OrderType_SELL
	if trade.IsBuy {
		orderType = v2.OrderType_BUY
	}

	if err := ensureSyntheticTradeMeetsMarketRequirements(trade, market, markPrice, orderType); err != nil {
		return nil, err
	}

	if err := ensureNotionalCapNotBreached(orderType, trade, markPrice, cs); err != nil {
		return nil, err
	}

	tradingFee := trade.Quantity.Mul(markPrice).Mul(market.TakerFeeRate)

	isClosingPosition := trade.IsBuy != position.IsLong && !position.Quantity.IsZero()
	if isClosingPosition {
		closingPrice := trade.Price
		if err := k.ensurePositionAboveBankruptcyForClosing(position, market, closingPrice, tradingFee); err != nil {
			return nil, err
		}
	}

	isInvalidReduceOnly := trade.IsReduceOnly() && (!isClosingPosition || position.Quantity.LT(trade.Quantity))
	if isInvalidReduceOnly {
		return nil, errors.Wrapf(
			types.ErrInsufficientPositionQuantity,
			"invalid reduce-only synthetic trade (position quantity: %s, trade quantity: %s)",
			position.Quantity.String(), trade.Quantity.String(),
		)
	}

	profile, _ := k.derivative.RiskEngine().EffectiveProfile(ctx, trade.SubaccountID)
	isCross := profile != nil && profile.Mode == v2.RiskMode_RISK_MODE_CROSS

	positionDelta := &v2.PositionDelta{
		IsLong:            trade.IsBuy,
		ExecutionQuantity: trade.Quantity,
		ExecutionMargin:   trade.Margin,
		ExecutionPrice:    trade.Price,
	}
	payout, closeExecutionMargin, collateralizationMargin, pnl := position.ApplyPositionDelta(positionDelta, tradingFee)

	// For cross-margin subaccounts, skip the per-position IM check. In CM, position.Margin
	// is accounting state — pool-level equity covers the position. The post-batch
	// ensureCrossMarginPoolHealth validates solvency at the pool level.
	if isCross {
		if position.Quantity.IsNegative() {
			return nil, types.ErrNegativePositionQuantity
		}
	} else {
		if err := ensureSyntheticTradePositionPostDelta(position); err != nil {
			return nil, err
		}
	}

	updateCapsAfterTrade(cs, orderType, trade.Quantity, markPrice, trade.SubaccountID)

	if err := k.ensureAndApplySyntheticTradeMarketBalanceDelta(ctx, trade, market, payout, collateralizationMargin, tradingFee); err != nil {
		return nil, err
	}

	finalPositions.SetPosition(trade.MarketID, trade.SubaccountID, position)
	k.derivative.SavePosition(ctx, trade.MarketID, trade.SubaccountID, position)

	chainFormattedDepositDeltaAmount := market.NotionalToChainFormat(payout.Add(closeExecutionMargin))
	depositDelta := types.NewUniformDepositDelta(chainFormattedDepositDeltaAmount)
	k.subaccount.UpdateDepositWithDelta(ctx, trade.SubaccountID, market.QuoteDenom, depositDelta)

	// defensive programming
	if k.GetDeposit(ctx, trade.SubaccountID, market.QuoteDenom).IsNegative() {
		return nil, errors.Wrapf(
			types.ErrInsufficientDeposit,
			"subaccountID %s has insufficient deposit for market quote denom %s",
			types.SubaccountIDToSdkAddress(trade.SubaccountID).String(),
			market.QuoteDenom,
		)
	}

	chainFormattedFee := market.NotionalToChainFormat(tradingFee)
	totalFees[market.QuoteDenom] = totalFees[market.QuoteDenom].Add(chainFormattedFee)

	totalTransferredFunds := trade.Margin

	// reduce-only trades already pay fees via margin, so we don't double count them here
	if !trade.IsReduceOnly() {
		totalTransferredFunds = totalTransferredFunds.Add(tradingFee)
	}

	chainFormattedMarginAndFee := market.NotionalToChainFormat(totalTransferredFunds)
	totalMarginAndFees[market.QuoteDenom] = totalMarginAndFees[market.QuoteDenom].Add(chainFormattedMarginAndFee)

	tradeLog := &v2.DerivativeTradeLog{
		SubaccountId:        trade.SubaccountID.Bytes(),
		PositionDelta:       positionDelta,
		Payout:              payout,
		Fee:                 tradingFee,
		Pnl:                 pnl,
		OrderHash:           common.Hash{}.Bytes(),
		FeeRecipientAddress: common.Address{}.Bytes(),
	}

	var cumulativeFunding *math.LegacyDec
	if fundingInfo != nil {
		cf := fundingInfo.CumulativeFunding
		cumulativeFunding = &cf
	}

	return &syntheticTradeResult{
		marketID:           market.MarketId,
		isBuy:              trade.IsBuy,
		cumulativeFunding:  cumulativeFunding,
		tradeLog:           tradeLog,
		isStrictlyReducing: isStrictlyReducing,
	}, nil
}

func initSyntheticTradePosition(
	position *v2.Position,
	isBuy bool,
	fundingInfo *v2.PerpetualMarketFunding,
) *v2.Position {
	if position == nil {
		var cumulativeFundingEntry math.LegacyDec
		if fundingInfo != nil {
			cumulativeFundingEntry = fundingInfo.CumulativeFunding
		}
		return v2.NewPosition(isBuy, cumulativeFundingEntry)
	}

	if fundingInfo != nil {
		position.ApplyFunding(fundingInfo)
	}
	return position
}

func recordInitialPositionIfNeeded(
	initialPositions v2.ModifiedPositionCache,
	marketID common.Hash,
	subaccountID common.Hash,
	position *v2.Position,
) {
	if initialPositions.HasPositionBeenModified(marketID, subaccountID) {
		return
	}
	initialPositions.SetPosition(marketID, subaccountID, &v2.Position{
		IsLong:                 position.IsLong,
		Quantity:               position.Quantity,
		EntryPrice:             position.EntryPrice,
		Margin:                 position.Margin,
		CumulativeFundingEntry: position.CumulativeFundingEntry,
	})
}

func ensureNotionalCapNotBreached(
	orderType v2.OrderType,
	trade *types.SyntheticTrade,
	markPrice math.LegacyDec,
	cs *capState,
) error {
	breach, _ := derivative.DoesBreachOpenNotionalCap(
		orderType,
		trade.Quantity,
		markPrice,
		cs.currOpenNotional.Add(cs.addedOpenNotional),
		cs.posQty[trade.SubaccountID],
		cs.openNotionalCap,
	)
	if breach {
		return errors.Wrapf(types.ErrOpenNotionalCapBreached, "market %s: cap breached", trade.MarketID.Hex())
	}
	return nil
}

func ensureSyntheticTradeMeetsMarketRequirements(
	trade *types.SyntheticTrade,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	orderType v2.OrderType,
) error {
	derivativeOrder := &v2.DerivativeOrder{
		OrderInfo: v2.OrderInfo{
			Price:    trade.Price,
			Quantity: trade.Quantity,
		},
		OrderType: orderType,
		Margin:    trade.Margin,
	}

	if err := derivativeOrder.CheckTickSize(market.GetMinPriceTickSize(), market.GetMinQuantityTickSize()); err != nil {
		return err
	}

	if err := derivativeOrder.CheckNotional(market.GetMinNotional()); err != nil {
		return err
	}

	if trade.IsReduceOnly() {
		return nil
	}

	_, err := derivativeOrder.CheckMarginAndGetMarginHold(
		market.GetInitialMarginRatio(),
		markPrice,
		market.GetTakerFeeRate(),
		market.GetMarketType(),
		market.GetOracleScaleFactor(),
	)
	return err
}

func ensureSyntheticTradePositionPostDelta(
	position *v2.Position,
) error {
	if position.Quantity.IsNegative() {
		return types.ErrNegativePositionQuantity
	}
	return nil
}

func updateCapsAfterTrade(
	cs *capState,
	orderType v2.OrderType,
	quantity math.LegacyDec,
	markPrice math.LegacyDec,
	subaccountID common.Hash,
) {
	notionalDelta, qtyDelta, newPositionQuantity := derivative.GetValuesForNotionalCapChecks(
		orderType,
		quantity,
		markPrice,
		cs.posQty[subaccountID],
	)
	cs.openInterestDelta.AddMut(qtyDelta)
	cs.posQty[subaccountID] = newPositionQuantity
	cs.addedOpenNotional.AddMut(notionalDelta)
}

func (k WasmKeeper) ensureAndApplySyntheticTradeMarketBalanceDelta(
	ctx sdk.Context,
	trade *types.SyntheticTrade,
	market *v2.DerivativeMarket,
	payout math.LegacyDec,
	collateralizationMargin math.LegacyDec,
	tradingFee math.LegacyDec,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ensureAndApplySyntheticTradeMarketBalanceDelta")()

	marketBalanceDelta := v2.GetMarketBalanceDelta(payout, collateralizationMargin, tradingFee, trade.Margin.IsZero())
	chainFormattedMarketBalanceDelta := market.NotionalToChainFormat(marketBalanceDelta)
	availableMarketFunds := k.derivative.GetAvailableMarketFunds(ctx, trade.MarketID)

	isMarketSolvent := v2.IsMarketSolvent(availableMarketFunds, chainFormattedMarketBalanceDelta)
	if !isMarketSolvent {
		return types.ErrInsufficientMarketBalance
	}

	k.derivative.ApplyMarketBalanceDelta(ctx, trade.MarketID, chainFormattedMarketBalanceDelta)
	return nil
}

func buildCoinsToTransfer(totalMarginAndFees map[string]math.LegacyDec) sdk.Coins {
	coinsToTransfer := sdk.Coins{}
	for denom, fundsUsed := range totalMarginAndFees {
		fundsUsedCoin := sdk.NewCoin(denom, fundsUsed.Ceil().TruncateInt())
		if !fundsUsedCoin.IsPositive() {
			continue
		}
		coinsToTransfer = coinsToTransfer.Add(fundsUsedCoin)
	}
	return coinsToTransfer
}

func (k WasmKeeper) transferSyntheticTradeFunds(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	coinsToTransfer sdk.Coins,
	totalFees map[string]math.LegacyDec,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "transferSyntheticTradeFunds")()

	if coinsToTransfer.IsZero() {
		return nil
	}

	if err := k.bank.SendCoinsFromAccountToModule(ctx, contractAddress, types.ModuleName, coinsToTransfer); err != nil {
		return errors.Wrap(err, "failed SyntheticTradeAction")
	}

	sortedDenomKeys := GetSortedFeesKeys(totalFees)
	for _, denom := range sortedDenomKeys {
		k.subaccount.UpdateDepositWithDelta(ctx, types.AuctionSubaccountID, denom, types.NewUniformDepositDelta(totalFees[denom]))
	}

	return nil
}

func (WasmKeeper) ensurePositionAboveBankruptcyForClosing(
	position *v2.Position,
	market *v2.DerivativeMarket,
	closingPrice, closingFee math.LegacyDec,
) error {
	if !position.Quantity.IsPositive() {
		return nil
	}

	positionMarginRatio := position.GetEffectiveMarginRatio(closingPrice, closingFee)
	bankruptcyMarginRatio := math.LegacyZeroDec()

	if positionMarginRatio.LT(bankruptcyMarginRatio) {
		return errors.Wrapf(
			types.ErrLowPositionMargin,
			"position margin ratio %s ≥ %s must hold", positionMarginRatio.String(), bankruptcyMarginRatio.String(),
		)
	}

	return nil
}

// ensureCrossMarginEligibility checks whether a market is eligible for cross-margin trading
// for the given subaccount. Returns nil if the subaccount is not in cross-margin mode.
func (k WasmKeeper) ensureCrossMarginEligibility(
	ctx sdk.Context,
	subaccountID common.Hash,
	market *v2.DerivativeMarket,
	marketID common.Hash,
) error {
	profile, _ := k.derivative.RiskEngine().EffectiveProfile(ctx, subaccountID)
	if profile == nil || profile.Mode != v2.RiskMode_RISK_MODE_CROSS {
		return nil
	}

	if err := k.derivative.RiskEngine().CheckCrossMarginMarketEligibility(ctx, market); err != nil {
		return err
	}

	return k.checkCrossMarginActiveMarketCap(ctx, subaccountID, market, marketID)
}

func (k WasmKeeper) checkCrossMarginActiveMarketCap(
	ctx sdk.Context,
	subaccountID common.Hash,
	market *v2.DerivativeMarket,
	marketID common.Hash,
) error {
	maxActive := k.GetParams(ctx).CrossMarginParams.MaxActiveDerivativeMarketsPerPool
	if maxActive == 0 {
		maxActive = 100
	}

	activeMarketIDs := risk.MergeAndSortMarketIDs(
		k.GetActiveDerivativeMarketsBySubaccount(ctx, subaccountID),
		k.GetActiveDerivativeOrderMarketsBySubaccount(ctx, subaccountID),
		k.liveTransientDerivativeOrderMarkets(ctx, subaccountID),
	)
	activeCount := uint32(0)
	hasMarket := false
	for _, id := range activeMarketIDs {
		if id == marketID {
			hasMarket = true
		}
		m := k.GetDerivativeMarketByID(ctx, id)
		if m == nil || m.GetMarketType().IsBinaryOptions() || m.QuoteDenom != market.QuoteDenom {
			continue
		}
		activeCount++
	}
	if !hasMarket && activeCount >= maxActive {
		return errors.Wrapf(types.ErrFeatureDisabled,
			"cross-margin pool %s would exceed max active markets (%d)", market.QuoteDenom, maxActive)
	}
	return nil
}

// liveTransientDerivativeOrderMarkets returns the subset of transient derivative order
// indicator markets that still have at least one live order. Mirrors the core risk engine's
// crossMarginModel.liveTransientDerivativeOrderMarkets to avoid counting stale indicators
// from cancelled/filled same-block orders.
func (k WasmKeeper) liveTransientDerivativeOrderMarkets(ctx sdk.Context, subaccountID common.Hash) []common.Hash {
	candidates := k.GetTransientDerivativeOrderIndicatorMarketsBySubaccount(ctx, subaccountID)
	if len(candidates) == 0 {
		return nil
	}

	live := make([]common.Hash, 0, len(candidates))
	for _, marketID := range candidates {
		if k.hasLiveTransientDerivativeOrder(ctx, marketID, subaccountID) {
			live = append(live, marketID)
		}
	}
	return live
}

func (k WasmKeeper) hasLiveTransientDerivativeOrder(ctx sdk.Context, marketID, subaccountID common.Hash) bool {
	found := false
	for _, isBuy := range []bool{true, false} {
		if found {
			break
		}
		k.IterateTransientDerivativeLimitOrdersBySubaccount(ctx, marketID, isBuy, subaccountID, func(_ *v2.DerivativeLimitOrder) (stop bool) {
			found = true
			return true
		})
	}
	for _, isBuy := range []bool{true, false} {
		if found {
			break
		}
		found = k.HasTransientDerivativeMarketOrderForSubaccount(ctx, marketID, subaccountID, isBuy)
	}
	return found
}

// ensureCrossMarginPoolHealthAfterSyntheticTrades checks pool health for each unique
// (subaccount, quoteDenom) pool affected by the synthetic trade batch that had at least
// one non-reducing trade. Pools with exclusively reducing trades are skipped to allow
// wind-down flows for distressed accounts, matching the normal FBA reduce-only path.
func (k WasmKeeper) ensureCrossMarginPoolHealthAfterSyntheticTrades(
	ctx sdk.Context,
	trades []*types.SyntheticTrade,
	markets map[common.Hash]*v2.DerivativeMarketInfo,
	nonReducingPools map[syntheticPoolKey]struct{},
) error {
	checked := make(map[syntheticPoolKey]struct{})

	for _, trade := range trades {
		m := markets[trade.MarketID]
		key := syntheticPoolKey{subaccountID: trade.SubaccountID, quoteDenom: m.Market.QuoteDenom}

		// Skip pools that only had reducing trades.
		if _, needsCheck := nonReducingPools[key]; !needsCheck {
			continue
		}

		if _, done := checked[key]; done {
			continue
		}
		checked[key] = struct{}{}

		if err := k.ensureCrossMarginPoolHealth(ctx, trade.SubaccountID, m.Market.QuoteDenom, m.Market.QuoteDecimals); err != nil {
			return err
		}
	}

	return nil
}

// ensureCrossMarginPoolHealth verifies that a cross-margin subaccount's pool remains above
// both maintenance margin and order-lock requirement after a state change.
// Returns nil for non-CM subaccounts.
func (k WasmKeeper) ensureCrossMarginPoolHealth(
	ctx sdk.Context,
	subaccountID common.Hash,
	quoteDenom string,
	quoteDecimals uint32,
) error {
	profile, _ := k.derivative.RiskEngine().EffectiveProfile(ctx, subaccountID)
	if profile == nil || profile.Mode != v2.RiskMode_RISK_MODE_CROSS {
		return nil
	}

	snapshot, err := k.derivative.RiskEngine().BuildCrossPoolSnapshot(ctx, subaccountID, quoteDenom, quoteDecimals)
	if err != nil {
		return errors.Wrap(err, "failed to build cross-margin snapshot for pool health check")
	}

	if snapshot.MaintenanceMarginTotal.IsPositive() {
		if snapshot.EquityLiquidation.LT(snapshot.MaintenanceMarginTotal) {
			return errors.Wrapf(
				types.ErrInsufficientMargin,
				"cross-margin pool %s maintenance check failed: equity %s < maintenance %s",
				quoteDenom,
				snapshot.EquityLiquidation.String(),
				snapshot.MaintenanceMarginTotal.String(),
			)
		}
	}

	// Check order-lock admission: changing positions can alter worst-case exposure for
	// resting orders, so equity_admission must still cover the order lock requirement.
	if snapshot.EquityAdmission.LT(snapshot.OrderLockRequirement) {
		return errors.Wrapf(
			types.ErrInsufficientMargin,
			"cross-margin pool %s admission check failed: equity_admission %s < order_lock %s",
			quoteDenom,
			snapshot.EquityAdmission.String(),
			snapshot.OrderLockRequirement.String(),
		)
	}

	return nil
}

func GetSortedFeesKeys(p map[string]math.LegacyDec) []string {
	denoms := make([]string, 0, len(p))
	for k := range p {
		denoms = append(denoms, k)
	}
	sort.SliceStable(denoms, func(i, j int) bool {
		return denoms[i] < denoms[j]
	})
	return denoms
}

func (k WasmKeeper) resolveSyntheticTradeROConflictsForMarket(
	ctx sdk.Context,
	marketID common.Hash,
	initialPositions,
	finalPositions v2.ModifiedPositionCache,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "resolveSyntheticTradeROConflictsForMarket")()

	subaccountIDs := initialPositions.GetSortedSubaccountIDsByMarket(marketID)

	for _, subaccountID := range subaccountIDs {
		initialPosition := initialPositions.GetPosition(marketID, subaccountID)
		finalPosition := finalPositions.GetPosition(marketID, subaccountID)

		hasNoPossibleContentions := initialPosition.IsLong == finalPosition.IsLong && finalPosition.Quantity.GTE(initialPosition.Quantity)
		if hasNoPossibleContentions {
			continue
		}

		metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, !initialPosition.IsLong)
		if initialPosition.IsLong != finalPosition.IsLong || finalPosition.Quantity.IsZero() {
			if err := k.cancelAllReduceOnlyOrders(ctx, marketID, subaccountID, metadata, !initialPosition.IsLong); err != nil {
				return err
			}
			continue
		}

		// partial closing case
		if err := k.checkAndResolveReduceOnlyConflicts(ctx, marketID, subaccountID, finalPosition, !finalPosition.IsLong); err != nil {
			return err
		}
	}

	return nil
}

func (k WasmKeeper) calculateFundsDifference(ctx sdk.Context, sender sdk.AccAddress, fundsBefore sdk.Coins) sdk.Coins {
	defer k.Meter(ctx).FuncTiming(&ctx, "calculateFundsDifference")()

	fundsAfter := sdk.Coins(make([]sdk.Coin, 0, len(fundsBefore)))

	for _, coin := range fundsBefore {
		coinAfter := k.bank.GetBalance(ctx, sender, coin.Denom)
		fundsAfter = fundsAfter.Add(coinAfter)
	}

	fundsDiff, _ := fundsAfter.SafeSub(fundsBefore...)

	return filterNonPositiveCoins(fundsDiff)
}

func filterNonPositiveCoins(coins sdk.Coins) sdk.Coins {
	var filteredCoins sdk.Coins
	for _, coin := range coins {
		if coin.IsPositive() {
			filteredCoins = append(filteredCoins, coin)
		}
	}
	return filteredCoins
}

func (k WasmKeeper) QueryMarketID(ctx sdk.Context, contractAddress string) (common.Hash, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "QueryMarketID")()

	type getMarketIDQuery struct {
	}

	type queryDataStruct struct {
		Data getMarketIDQuery `json:"get_market_id"`
	}

	type baseMsgWrapper struct {
		Base queryDataStruct `json:"base"`
	}

	queryData := baseMsgWrapper{
		queryDataStruct{
			Data: getMarketIDQuery{},
		},
	}

	queryDataBz, err := json.Marshal(queryData)
	if err != nil {
		return common.Hash{}, err
	}

	contractAddressAcc := sdk.MustAccAddressFromBech32(contractAddress)
	bz, err := k.wasmv.QuerySmart(ctx, contractAddressAcc, queryDataBz)
	if err != nil {
		return common.Hash{}, err
	}

	type Data struct {
		MarketId string `json:"market_id"`
	}

	var result Data
	if err := json.Unmarshal(bz, &result); err != nil {
		return common.Hash{}, err
	}

	return common.HexToHash(result.MarketId), nil
}
