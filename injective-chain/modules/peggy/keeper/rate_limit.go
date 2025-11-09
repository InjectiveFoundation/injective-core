package keeper

import (
	"errors"
	"math/big"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/metrics"
)

var (
	ErrRateLimitOverflow         = errors.New("rate limit overflow")
	ErrAbsoluteMintLimitOverflow = errors.New("absolute mint limit overflow")
)

func (k *Keeper) CheckRateLimit(
	ctx sdk.Context,
	tokenAddress gethcommon.Address,
	newTxs []*types.OutgoingTransferTx,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimit := k.GetRateLimit(ctx, tokenAddress)
	if rateLimit == nil {
		return nil // no-op
	}

	var (
		totalInBatches  = sdkmath.ZeroInt()
		totalInNewTxs   = sdkmath.ZeroInt()
		totalInflow     = rateLimit.TotalInflow()
		totalOutflow    = rateLimit.TotalOutflow()
		surplus         = totalOutflow.Sub(totalInflow)
		existingBatches = make([]*types.OutgoingTxBatch, 0)
	)

	// 1. Determine if more tokens are being withdrawn than deposited
	if thereIsActuallyMoreInThanOut := !surplus.IsPositive(); thereIsActuallyMoreInThanOut {
		surplus = sdkmath.ZeroInt()
	}

	// 2. Sum existing batches
	k.IterateOutgoingTXBatches(ctx, func(_ []byte, batch *types.OutgoingTxBatch) bool {
		if batch.TokenContract == tokenAddress.String() {
			existingBatches = append(existingBatches, batch)
		}
		return false
	})

	for _, batch := range existingBatches {
		for _, tx := range batch.Transactions {
			totalInBatches = totalInBatches.Add(tx.Erc20Fee.Amount)
			totalInBatches = totalInBatches.Add(tx.Erc20Token.Amount)
		}
	}

	// 3. Sum new txs
	for _, tx := range newTxs {
		totalInNewTxs = totalInNewTxs.Add(tx.Erc20Fee.Amount)
		totalInNewTxs = totalInNewTxs.Add(tx.Erc20Token.Amount)
	}

	entireWithdrawAmountSoFar := surplus.Add(totalInBatches).Add(totalInNewTxs)
	quantity := entireWithdrawAmountSoFar.ToLegacyDec()
	quantity = quantity.Quo(sdkmath.LegacyNewDec(10).Power(uint64(rateLimit.TokenDecimals))) // human-readable

	valueInUSD := k.OracleKeeper.GetPythPrice(ctx, rateLimit.TokenPriceId, "USD")
	if valueInUSD == nil {
		// todo(dusan): perform check during MsgServer CreateRateLimit?
		return errors.New("nil Pyth price")
	}

	notional := quantity.Mul(*valueInUSD)
	if notional.GTE(rateLimit.RateLimitUsd) {
		return sdkerrors.Wrapf(ErrRateLimitOverflow, "configured limit: %sUSD", rateLimit.RateLimitUsd.String())
	}

	// todo?(dusan): peggo sidecar should be smarter when creating batches (not to waste its funds)

	return nil
}

func (k *Keeper) TrackTokenInflow(ctx sdk.Context, tokenAddress gethcommon.Address, in sdkmath.Int) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimit := k.GetRateLimit(ctx, tokenAddress)
	if rateLimit == nil {
		return // no-op
	}

	rateLimit.Transfers = append(rateLimit.Transfers, &types.BridgeTransfer{
		BlockNumber: uint64(ctx.BlockHeight()),
		Amount:      in,
		IsDeposit:   true,
	})

	k.SetRateLimit(ctx, rateLimit)
}

func (k *Keeper) TrackTokenOutflow(ctx sdk.Context, tokenAddress gethcommon.Address, out sdkmath.Int) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimit := k.GetRateLimit(ctx, tokenAddress)
	if rateLimit == nil {
		return // no-op
	}

	rateLimit.Transfers = append(rateLimit.Transfers, &types.BridgeTransfer{
		BlockNumber: uint64(ctx.BlockHeight()),
		Amount:      out,
		IsDeposit:   false,
	})

	k.SetRateLimit(ctx, rateLimit)

	currentAmount := k.GetMintAmountERC20(ctx, tokenAddress)
	k.SetMintAmountERC20(ctx, tokenAddress, currentAmount.Sub(currentAmount, out.BigInt()))
}

func (k *Keeper) SetRateLimit(ctx sdk.Context, rateLimit *types.RateLimit) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimitStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.RateLimitsKey)
	rateLimitStore.Set(gethcommon.HexToAddress(rateLimit.TokenAddress).Bytes(), k.cdc.MustMarshal(rateLimit))
}

func (k *Keeper) GetRateLimit(ctx sdk.Context, tokenAddress gethcommon.Address) *types.RateLimit {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimitStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.RateLimitsKey)

	bz := rateLimitStore.Get(tokenAddress.Bytes())
	if len(bz) == 0 {
		return nil
	}

	var rateLimit types.RateLimit
	k.cdc.MustUnmarshal(bz, &rateLimit)

	return &rateLimit
}

func (k *Keeper) DeleteRateLimit(ctx sdk.Context, tokenAddress gethcommon.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimitStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.RateLimitsKey)
	rateLimitStore.Delete(tokenAddress.Bytes())
}

func (k *Keeper) GetRateLimits(ctx sdk.Context) []*types.RateLimit {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimitsStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.RateLimitsKey)
	iter := rateLimitsStore.Iterator(nil, nil)
	defer iter.Close()

	var rateLimits []*types.RateLimit
	for ; iter.Valid(); iter.Next() {
		var rateLimit types.RateLimit
		k.cdc.MustUnmarshal(iter.Value(), &rateLimit)
		rateLimits = append(rateLimits, &rateLimit)
	}

	return rateLimits
}

func (k *Keeper) GetMintAmountERC20(ctx sdk.Context, tokenAddress gethcommon.Address) *big.Int {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetMintAmountERC20Key(tokenAddress.Bytes()))
	if len(bz) == 0 {
		return big.NewInt(0)
	}

	return big.NewInt(0).SetBytes(bz)
}

func (k *Keeper) SetMintAmountERC20(ctx sdk.Context, tokenAddress gethcommon.Address, amount *big.Int) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetMintAmountERC20Key(tokenAddress.Bytes()), amount.Bytes())
}

func (k *Keeper) CheckAbsoluteLimit(ctx sdk.Context, tokenAddress gethcommon.Address, amount *big.Int) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rateLimit := k.GetRateLimit(ctx, tokenAddress)
	if rateLimit == nil {
		return nil // no-op
	}

	absoluteLimit := rateLimit.AbsoluteMintLimit.BigInt()
	currentAmount := k.GetMintAmountERC20(ctx, tokenAddress)
	if remaining := absoluteLimit.Sub(absoluteLimit, currentAmount); remaining.Cmp(amount) < 0 {
		return ErrAbsoluteMintLimitOverflow
	}

	return nil
}
