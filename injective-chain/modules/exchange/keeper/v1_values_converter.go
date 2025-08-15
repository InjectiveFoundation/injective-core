package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type ChainValuesConverter interface {
	PriceFromChainFormat(price math.LegacyDec) math.LegacyDec
	QuantityFromChainFormat(quantity math.LegacyDec) math.LegacyDec
	NotionalFromChainFormat(notional math.LegacyDec) math.LegacyDec
	PriceToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec
	QuantityToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec
	NotionalToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec
}

var (
	_ ChainValuesConverter = PreHumanReadableUpgradeValuesConverter{}
	_ ChainValuesConverter = PostHumanReadableUpgradeValuesConverter{}
)

func NewChainValuesConverter(ctx sdk.Context, k *Keeper, market MarketInterface) ChainValuesConverter {
	var converter ChainValuesConverter
	if ctx.BlockHeight() < k.GetParams(ctx).HumanReadableUpgradeBlockHeight {
		converter = PreHumanReadableUpgradeValuesConverter{market}
	} else {
		converter = PostHumanReadableUpgradeValuesConverter{market}
	}
	return converter
}

func ConditionalNotionalToChainFormat(ctx sdk.Context, k *Keeper, value math.LegacyDec, decimals uint32) math.LegacyDec {
	var notional math.LegacyDec
	if isPreHumanReadableUpgradeBlock(ctx, k) {
		notional = value
	} else {
		notional = types.NotionalToChainFormat(value, decimals)
	}
	return notional
}

func isPreHumanReadableUpgradeBlock(ctx sdk.Context, k *Keeper) bool {
	return ctx.BlockHeight() < k.GetParams(ctx).HumanReadableUpgradeBlockHeight
}

type PreHumanReadableUpgradeValuesConverter struct {
	market MarketInterface
}

func (PreHumanReadableUpgradeValuesConverter) PriceFromChainFormat(price math.LegacyDec) math.LegacyDec {
	// In blocks before the v1.16.0 upgrade, prices were stored in chain format already by the keeper
	return price
}

func (PreHumanReadableUpgradeValuesConverter) QuantityFromChainFormat(quantity math.LegacyDec) math.LegacyDec {
	// In blocks before the v1.16.0 upgrade, quantities were stored in chain format already by the keeper
	return quantity
}

func (PreHumanReadableUpgradeValuesConverter) NotionalFromChainFormat(notional math.LegacyDec) math.LegacyDec {
	// In blocks before the v1.16.0 upgrade, notional values were stored in chain format already by the keeper
	return notional
}

func (PreHumanReadableUpgradeValuesConverter) PriceToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec {
	// In blocks before the v1.16.0 upgrade, prices were stored in chain format already by the keeper
	return humanReadableValue
}

func (PreHumanReadableUpgradeValuesConverter) QuantityToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec {
	// In blocks before the v1.16.0 upgrade, quantities were stored in chain format already by the keeper
	return humanReadableValue
}

func (PreHumanReadableUpgradeValuesConverter) NotionalToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec {
	// In blocks before the v1.16.0 upgrade, notional values were stored in chain format already by the keeper
	return humanReadableValue
}

type PostHumanReadableUpgradeValuesConverter struct {
	market MarketInterface
}

func (c PostHumanReadableUpgradeValuesConverter) PriceFromChainFormat(price math.LegacyDec) math.LegacyDec {
	return c.market.PriceFromChainFormat(price)
}

func (c PostHumanReadableUpgradeValuesConverter) QuantityFromChainFormat(quantity math.LegacyDec) math.LegacyDec {
	return c.market.QuantityFromChainFormat(quantity)
}

func (c PostHumanReadableUpgradeValuesConverter) NotionalFromChainFormat(notional math.LegacyDec) math.LegacyDec {
	return c.market.NotionalFromChainFormat(notional)
}

func (c PostHumanReadableUpgradeValuesConverter) PriceToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec {
	return c.market.PriceToChainFormat(humanReadableValue)
}

func (c PostHumanReadableUpgradeValuesConverter) QuantityToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec {
	return c.market.QuantityToChainFormat(humanReadableValue)
}

func (c PostHumanReadableUpgradeValuesConverter) NotionalToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec {
	return c.market.NotionalToChainFormat(humanReadableValue)
}
