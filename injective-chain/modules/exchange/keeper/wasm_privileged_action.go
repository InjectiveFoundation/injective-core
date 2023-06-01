package keeper

import (
	"sort"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) handlePrivilegedAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action types.InjectiveAction,
) error {
	switch t := action.(type) {
	case *types.SyntheticTradeAction:
		return k.handleSyntheticTradeAction(ctx, contractAddress, origin, t)
	case *types.PositionTransfer:
		return k.handlePositionTransferAction(ctx, contractAddress, origin, t)
	default:
		return types.ErrUnsupportedAction
	}
}

func GetSortedFeesKeys(p map[string]sdk.Dec) []string {
	denoms := make([]string, 0)
	for k := range p {
		denoms = append(denoms, k)
	}
	sort.SliceStable(denoms, func(i, j int) bool {
		return denoms[i] < denoms[j]
	})
	return denoms
}

func (k *Keeper) ensurePositionAboveBankruptcyForClosing(
	position *types.Position,
	market *types.DerivativeMarket,
	closingPrice, closingFee sdk.Dec,
) error {
	if !position.Quantity.IsPositive() {
		return nil
	}

	positionMarginRatio := position.GetEffectiveMarginRatio(closingPrice, closingFee)
	bankruptcyMarginRatio := sdk.ZeroDec()

	if positionMarginRatio.LT(bankruptcyMarginRatio) {
		return errors.Wrapf(types.ErrLowPositionMargin, "position margin ratio %s ≥ %s must hold", positionMarginRatio.String(), market.InitialMarginRatio.String())
	}

	return nil
}

func (k *Keeper) ensurePositionAboveInitialMarginRatio(
	position *types.Position,
	market *types.DerivativeMarket,
	markPrice sdk.Dec,
) error {
	if !position.Quantity.IsPositive() {
		return nil
	}

	positionMarginRatio := position.GetEffectiveMarginRatio(markPrice, sdk.ZeroDec())

	if positionMarginRatio.LT(market.InitialMarginRatio) {
		return errors.Wrapf(types.ErrLowPositionMargin, "position margin ratio %s ≥ %s must hold", positionMarginRatio.String(), market.InitialMarginRatio.String())
	}

	return nil
}

func (k *Keeper) handleSyntheticTradeAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action *types.SyntheticTradeAction,
) error {
	summary, err := action.Summarize()
	if err != nil {
		return err
	}

	// Enforce that subaccountIDs provided match either the contract address or the origin address
	if !contractAddress.Equals(summary.ContractAddress) || !origin.Equals(summary.UserAddress) {
		return errors.Wrapf(types.ErrBadSubaccountID, "subaccountID address %s does not match either contract address %s or origin address %s", summary.UserAddress.String(), contractAddress.String(), origin.String())
	}

	marketIDs := summary.GetMarketIDs()
	totalMarginAndFees := make(map[string]sdk.Dec)
	totalFees := make(map[string]sdk.Dec)
	markets := make(map[common.Hash]*types.DerivativeMarketInfo)

	for _, marketID := range marketIDs {
		m := k.GetDerivativeMarketInfo(ctx, marketID, true)
		if m.Market == nil || m.MarkPrice.IsNil() {
			return errors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", marketID.Hex())
		}

		markets[marketID] = m
		totalMarginAndFees[m.Market.QuoteDenom] = sdk.ZeroDec()
		totalFees[m.Market.QuoteDenom] = sdk.ZeroDec()
	}

	initialPositions := NewModifiedPositionCache()
	finalPositions := NewModifiedPositionCache()

	trades := action.UserTrades
	trades = append(trades, action.ContractTrades...)

	for _, trade := range trades {
		m := markets[trade.MarketID]
		market := m.Market
		markPrice := m.MarkPrice

		var fundingInfo *types.PerpetualMarketFunding
		if market.IsPerpetual {
			fundingInfo = m.Funding
		}

		// Initialize position and apply funding
		position := k.GetPosition(ctx, trade.MarketID, trade.SubaccountID)
		if position == nil {
			var cumulativeFundingEntry sdk.Dec
			if fundingInfo != nil {
				cumulativeFundingEntry = fundingInfo.CumulativeFunding
			}
			position = types.NewPosition(trade.IsBuy, cumulativeFundingEntry)
		} else if market.IsPerpetual {
			position.ApplyFunding(fundingInfo)
		}

		// only store the initial position state
		if !initialPositions.HasPositionBeenModified(trade.MarketID, trade.SubaccountID) {
			initialPositions.SetPosition(trade.MarketID, trade.SubaccountID, &types.Position{
				IsLong:                 position.IsLong,
				Quantity:               position.Quantity,
				EntryPrice:             position.EntryPrice,
				Margin:                 position.Margin,
				CumulativeFundingEntry: position.CumulativeFundingEntry,
			})
		}

		tradingFee := trade.Quantity.Mul(markPrice).Mul(market.TakerFeeRate)

		isClosingPosition := trade.IsBuy != position.IsLong && !position.Quantity.IsZero()
		if isClosingPosition {
			closingPrice := trade.Price
			if err := k.ensurePositionAboveBankruptcyForClosing(position, market, closingPrice, tradingFee); err != nil {
				return err
			}
		}

		payout, closeExecutionMargin, _ := position.ApplyPositionDelta(trade.ToPositionDelta(), sdk.ZeroDec())

		// Enforce that a position cannot have a negative quantity
		if position.Quantity.IsNegative() {
			return types.ErrNegativePositionQuantity
		}

		if err := k.ensurePositionAboveInitialMarginRatio(position, market, markPrice); err != nil {
			return err
		}

		finalPositions.SetPosition(trade.MarketID, trade.SubaccountID, position)
		k.SetPosition(ctx, trade.MarketID, trade.SubaccountID, position)

		depositDelta := types.NewUniformDepositDelta(payout.Add(closeExecutionMargin))
		k.UpdateDepositWithDelta(ctx, trade.SubaccountID, market.QuoteDenom, depositDelta)

		totalMarginAndFees[market.QuoteDenom] = totalMarginAndFees[market.QuoteDenom].Add(tradingFee).Add(trade.Margin)
		totalFees[market.QuoteDenom] = totalFees[market.QuoteDenom].Add(tradingFee)
	}

	// Transfer funds from the contract to exchange module to pay for the synthetic trades
	coinsToTransfer := sdk.Coins{}
	for denom, fundsUsed := range totalMarginAndFees {
		fundsUsedCoin := sdk.NewCoin(denom, fundsUsed.Ceil().TruncateInt())
		if !fundsUsedCoin.IsPositive() {
			continue
		}
		coinsToTransfer = coinsToTransfer.Add(fundsUsedCoin)
	}

	if !coinsToTransfer.IsZero() {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, contractAddress, types.ModuleName, coinsToTransfer); err != nil {
			return errors.Wrap(err, "failed SyntheticTradeAction")
		}

		sortedDenomKeys := GetSortedFeesKeys(totalFees)
		for _, denom := range sortedDenomKeys {
			k.UpdateDepositWithDelta(ctx, types.AuctionSubaccountID, denom, types.NewUniformDepositDelta(totalFees[denom]))
		}
	}

	for _, marketID := range marketIDs {
		k.resolveSyntheticTradeROConflictsForMarket(ctx, marketID, initialPositions, finalPositions)
	}

	return nil
}

func (k *Keeper) resolveSyntheticTradeROConflictsForMarket(
	ctx sdk.Context,
	marketID common.Hash,
	initialPositions, finalPositions ModifiedPositionCache,
) {
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
			k.cancelAllReduceOnlyOrders(ctx, marketID, subaccountID, metadata, !initialPosition.IsLong)
			continue
		}

		// partial closing case
		k.checkAndResolveReduceOnlyConflicts(ctx, marketID, subaccountID, finalPosition, !finalPosition.IsLong)
	}
}

func (k *Keeper) handlePositionTransferAction(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	origin sdk.AccAddress,
	action *types.PositionTransfer,
) error {
	m := k.GetDerivativeMarketInfo(ctx, action.MarketID, true)

	var (
		market    = m.Market
		markPrice = m.MarkPrice
		funding   = m.Funding
	)

	if market == nil || markPrice.IsNil() {
		return errors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", action.MarketID.Hex())
	}

	sourceAddress := types.SubaccountIDToSdkAddress(action.SourceSubaccountID)
	destinationAddress := types.SubaccountIDToSdkAddress(action.DestinationSubaccountID)

	// TODO consider also allowing position transfer from user to contract address
	if !contractAddress.Equals(sourceAddress) {
		return errors.Wrapf(types.ErrBadSubaccountID, "Source subaccountID address %s does not match origin address %s", sourceAddress.String(), origin.String())
	}
	if !origin.Equals(destinationAddress) {
		return errors.Wrapf(types.ErrBadSubaccountID, "Destination subaccountID address %s does not match contract address %s", destinationAddress.String(), contractAddress.String())
	}

	sourcePosition := k.GetPosition(ctx, action.MarketID, action.SourceSubaccountID)
	destinationPosition := k.GetPosition(ctx, action.MarketID, action.DestinationSubaccountID)

	// Enforce that source position has sufficient quantity for transfer
	if sourcePosition == nil || sourcePosition.Quantity.LT(action.Quantity) {
		return errors.Wrapf(types.ErrInvalidQuantity, "Source subaccountID position quantity")
	}

	if destinationPosition == nil {
		var cumulativeFundingEntry sdk.Dec
		if funding != nil {
			cumulativeFundingEntry = funding.CumulativeFunding
		}
		destinationPosition = types.NewPosition(sourcePosition.IsLong, cumulativeFundingEntry)
	}

	if market.IsPerpetual {
		destinationPosition.ApplyFunding(funding)
		sourcePosition.ApplyFunding(funding)
	}

	// Enforce each position's effectiveMargin / (markPrice * quantity) ≥ maintenanceMarginRatio
	if sourcePosition.Quantity.IsPositive() {
		positionMarginRatio := sourcePosition.GetEffectiveMarginRatio(markPrice, sdk.ZeroDec())
		if positionMarginRatio.LT(market.MaintenanceMarginRatio) {
			return errors.Wrapf(types.ErrLowPositionMargin, "position margin ratio %s ≥ %s must hold", positionMarginRatio.String(), market.MaintenanceMarginRatio.String())
		}
	}
	if destinationPosition.Quantity.IsPositive() {
		positionMarginRatio := destinationPosition.GetEffectiveMarginRatio(markPrice, sdk.ZeroDec())
		if positionMarginRatio.LT(market.MaintenanceMarginRatio) {
			return errors.Wrapf(types.ErrLowPositionMargin, "position margin ratio %s ≥ %s must hold", positionMarginRatio.String(), market.MaintenanceMarginRatio.String())
		}
	}

	executionPrice := sourcePosition.EntryPrice
	sourceMarginBefore := sourcePosition.Margin
	isSourceLongBefore, isDestinationLongBefore := sourcePosition.IsLong, destinationPosition.IsLong

	sourcePosition.ApplyPositionDelta(
		&types.PositionDelta{
			IsLong:            !sourcePosition.IsLong,
			ExecutionQuantity: action.Quantity,
			ExecutionMargin:   sdk.ZeroDec(),
			ExecutionPrice:    executionPrice,
		},
		sdk.ZeroDec(),
	)

	executionMargin := sourceMarginBefore.Sub(sourcePosition.Margin)
	payout, closeExecutionMargin, _ := destinationPosition.ApplyPositionDelta(
		&types.PositionDelta{
			IsLong:            sourcePosition.IsLong,
			ExecutionQuantity: action.Quantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    executionPrice,
		},
		sdk.ZeroDec(),
	)
	receiverTradingFee := markPrice.Mul(action.Quantity).Mul(market.TakerFeeRate)

	k.SetPosition(ctx, action.MarketID, action.SourceSubaccountID, sourcePosition)
	k.SetPosition(ctx, action.MarketID, action.DestinationSubaccountID, destinationPosition)

	depositDelta := types.NewUniformDepositDelta(payout.Add(closeExecutionMargin).Sub(receiverTradingFee))
	k.UpdateDepositWithDelta(ctx, action.DestinationSubaccountID, market.QuoteDenom, depositDelta)
	k.UpdateDepositWithDelta(ctx, types.AuctionSubaccountID, market.QuoteDenom, types.NewUniformDepositDelta(receiverTradingFee))

	k.checkAndResolveReduceOnlyConflicts(ctx, action.MarketID, action.SourceSubaccountID, sourcePosition, !sourcePosition.IsLong)

	isDestinationPositionNettingInSameDirection := isSourceLongBefore == isDestinationLongBefore
	if isDestinationPositionNettingInSameDirection {
		return nil
	}

	// if destination position flipped or is closed, cancel all RO orders
	if isDestinationLongBefore != destinationPosition.IsLong || destinationPosition.Quantity.IsZero() {
		metadata := k.GetSubaccountOrderbookMetadata(ctx, action.MarketID, action.DestinationSubaccountID, !isDestinationLongBefore)
		k.cancelAllReduceOnlyOrders(ctx, action.MarketID, action.DestinationSubaccountID, metadata, !isDestinationLongBefore)
		return nil
	}

	// partial closing case
	k.checkAndResolveReduceOnlyConflicts(ctx, action.MarketID, action.DestinationSubaccountID, destinationPosition, !destinationPosition.IsLong)
	return nil
}

func (k *Keeper) checkAndResolveReduceOnlyConflicts(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	position *types.Position,
	isReduceOnlyDirectionBuy bool,
) {
	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isReduceOnlyDirectionBuy)

	if metadata.ReduceOnlyLimitOrderCount == 0 {
		return
	}

	if position.Quantity.IsZero() {
		k.cancelAllReduceOnlyOrders(ctx, marketID, subaccountID, metadata, isReduceOnlyDirectionBuy)
		return
	}

	cumulativeOrderSideQuantity := metadata.AggregateReduceOnlyQuantity.Add(metadata.AggregateVanillaQuantity)

	maxRoQuantityToCancel := cumulativeOrderSideQuantity.Sub(position.Quantity)
	if maxRoQuantityToCancel.IsNegative() || maxRoQuantityToCancel.IsZero() {
		return
	}

	subaccountEOBResults := NewSubaccountOrderResults()
	k.cancelMinimumReduceOnlyOrders(ctx, marketID, subaccountID, metadata, isReduceOnlyDirectionBuy, position.Quantity, subaccountEOBResults, nil)
}
