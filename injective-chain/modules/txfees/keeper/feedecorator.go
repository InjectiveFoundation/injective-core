package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types" // as chaintypes
)

// MempoolFeeDecorator will check if the transaction's fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeDecorator.
type MempoolFeeDecorator struct {
	TxFeesKeeper *Keeper
}

func NewMempoolFeeDecorator(txFeesKeeper *Keeper) MempoolFeeDecorator {
	return MempoolFeeDecorator{
		TxFeesKeeper: txFeesKeeper,
	}
}

// The complexity is acceptable and we want to keep the function similar to the Osmosis version
//
//nolint:revive // the simulate parameters is a flag parameter, but it is required by the sdk
func (mfd MempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	txfeesParams := mfd.TxFeesKeeper.GetParams(ctx)

	if simulate || ctx.BlockHeight() == 0 {
		// If this is genesis height, don't check the fee.
		// This is needed so that gentx's can be created without having to pay a fee (chicken/egg problem).

		return next(ctx, tx, simulate)
	}

	feeTx, err := mfd.getValidatedFeeTx(ctx, tx, txfeesParams)
	if err != nil {
		return ctx, err
	} else if feeTx == nil {
		// no fee provided
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "no fee provided in tx")
	}

	// The rest of the logic is only needed for the 1559 mempool
	if !txfeesParams.Mempool1559Enabled {
		return next(ctx, tx, simulate)
	}

	// TODO: Is there a better way to do this?
	// I want ctx.IsDeliverTx() but that doesn't exist.
	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() {
		mfd.TxFeesKeeper.CurFeeState.DeliverTxCode(ctx, feeTx)
	}

	minBaseGasPrice := mfd.GetMinBaseGasPriceForTx(ctx, feeTx)
	if minBaseGasPrice.IsZero() {
		// If minBaseGasPrice is zero, then we don't need to check the fee. Continue
		return next(ctx, tx, simulate)
	}

	feeCoins := feeTx.GetFee()
	if err := mfd.isSufficientFee(minBaseGasPrice, feeTx.GetGas(), feeCoins[0]); err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}

func (mfd MempoolFeeDecorator) GetMinBaseGasPriceForTx(ctx sdk.Context, feeTx sdk.FeeTx) math.LegacyDec {
	txfeesParams := mfd.TxFeesKeeper.GetParams(ctx)
	minBaseGasPrice := mfd.TxFeesKeeper.CurFeeState.MinBaseFee

	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() {
		return minBaseGasPrice
	}

	// Handle high gas transactions
	if feeTx.GetGas() >= txfeesParams.HighGasTxThreshold {
		minBaseGasPrice = math.LegacyMaxDec(minBaseGasPrice, txfeesParams.MinGasPriceForHighGasTx)
	}

	return mfd.getMempool1559GasPrice(ctx, minBaseGasPrice)
}

// getValidatedFeeTx returns a FeeTx if the tx is a FeeTx, otherwise it returns an error
// if the tx is a FeeTx, it also checks that the fee is valid (only INJ)
// if there is no fee, it returns nil
func (MempoolFeeDecorator) getValidatedFeeTx(ctx sdk.Context, tx sdk.Tx, txfeesParams types.Params) (sdk.FeeTx, error) {
	// The SDK currently requires all txs to be FeeTx's in CheckTx, within its mempool fee decorator.
	// See: https://github.com/cosmos/cosmos-sdk/blob/f726a2398a26bdaf71d78dbf56a82621e84fd098/x/auth/middleware/fee.go#L34-L37
	// So this is not a real restriction at the moment.
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	// Ensure that the provided gas is less than the maximum gas per tx,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	if ctx.IsCheckTx() {
		if feeTx.GetGas() > txfeesParams.MaxGasWantedPerTx {
			msg := "Too much gas wanted: %d, maximum is %d"
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidGasLimit, msg, feeTx.GetGas(), txfeesParams.MaxGasWantedPerTx)
		}
	}

	feeCoins := feeTx.GetFee()
	if len(feeCoins) > 1 {
		return nil, types.ErrTooManyFeeCoins
	} else if len(feeCoins) == 0 {
		// no fee provided, return nil
		return nil, nil
	}

	// If there is a fee attached to the tx, make sure the fee denom is a denom accepted by the chain
	if feeDenom := feeCoins.GetDenomByIndex(0); feeDenom != chaintypes.InjectiveCoin && feeDenom != "stake" {
		return nil, errorsmod.Wrapf(types.ErrInvalidFeeToken, "fee denom is not a valid denom (%s or stake): %s",
			chaintypes.InjectiveCoin,
			feeDenom,
		)
	}

	return feeTx, nil
}

func (mfd MempoolFeeDecorator) getMempool1559GasPrice(ctx sdk.Context, minBaseGasPrice math.LegacyDec) math.LegacyDec {
	if ctx.IsCheckTx() && !ctx.IsReCheckTx() {
		return math.LegacyMaxDec(minBaseGasPrice, mfd.TxFeesKeeper.CurFeeState.GetCurBaseFee())
	}

	if ctx.IsReCheckTx() {
		return math.LegacyMaxDec(minBaseGasPrice, mfd.TxFeesKeeper.CurFeeState.GetCurRecheckBaseFee())
	}

	return minBaseGasPrice
}

func (MempoolFeeDecorator) isSufficientFee(minBaseGasPrice math.LegacyDec, gasRequested uint64, feeCoin sdk.Coin) error {
	// Determine the required fees by multiplying the required minimum gas
	// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
	// note we mutate this one line below, to avoid extra heap allocations.
	glDec := math.LegacyNewDec(int64(gasRequested))
	baseFeeAmt := glDec.MulMut(minBaseGasPrice).Ceil().RoundInt()
	requiredBaseFee := chaintypes.NewInjectiveCoin(baseFeeAmt)

	// check to ensure that the convertedFee should always be greater than or equal to the requireBaseFee
	if !(feeCoin.IsGTE(requiredBaseFee)) {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "got: %s required: %s", feeCoin, requiredBaseFee)
	}

	return nil
}
