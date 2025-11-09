package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func (k *Keeper) createBinaryOptionsMarketOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
) (orderHash common.Hash, err error) {
	orderHash, _, err = k.createBinaryOptionsMarketOrderWithResultsForAtomicExecution(ctx, sender, derivativeOrder, market, markPrice)
	return orderHash, err
}

func (k *Keeper) createBinaryOptionsMarketOrderWithResultsForAtomicExecution(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market DerivativeMarketInterface,
	_ math.LegacyDec,
) (orderHash common.Hash, results *v2.DerivativeMarketOrderResults, err error) {
	requiredMargin := derivativeOrder.GetRequiredBinaryOptionsMargin(market.GetOracleScaleFactor())
	if derivativeOrder.Margin.GT(requiredMargin) {
		// decrease order margin to the required amount if greater, since there's no need to overpay
		derivativeOrder.Margin = requiredMargin
	}

	return k.createDerivativeMarketOrder(ctx, sender, derivativeOrder, market, math.LegacyDec{})
}
