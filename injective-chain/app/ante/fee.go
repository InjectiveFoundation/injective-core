package ante

import (
	"fmt"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// DeductFeeDecorator deducts fees from the first signer of the tx
// If the first signer does not have the funds to pay for the fees, return with InsufficientFunds error
// Call next AnteHandler if fees successfully deducted
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type DeductFeeDecorator struct {
	ak           authante.AccountKeeper
	bankKeeper   types.BankKeeper
	txFeeChecker authante.TxFeeChecker
}

func NewDeductFeeDecorator(ak authante.AccountKeeper, bk types.BankKeeper) DeductFeeDecorator {
	return DeductFeeDecorator{
		ak:           ak,
		bankKeeper:   bk,
		txFeeChecker: authante.CheckTxFeeWithValidatorMinGasPrices,
	}
}

func (dfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errors.Wrapf(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	if addr := dfd.ak.GetModuleAddress(types.FeeCollectorName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.FeeCollectorName))
	}

	var feeDelegated bool
	var feePayer sdk.AccAddress

	if txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx); ok {
		if opts := txWithExtensions.GetExtensionOptions(); len(opts) > 0 {
			var optIface txtypes.TxExtensionOptionI
			if err := chainTypesCodec.UnpackAny(opts[0], &optIface); err != nil {
				return ctx, errors.Wrap(sdkerrors.ErrUnpackAny, "failed to proto-unpack ExtensionOptionsWeb3Tx")
			} else if extOpt, ok := optIface.(*chaintypes.ExtensionOptionsWeb3Tx); ok {
				if extOpt.FeePayer != "" {
					feePayer, err = sdk.AccAddressFromBech32(extOpt.FeePayer)
					if err != nil {
						err = errors.Wrapf(sdkerrors.ErrInvalidAddress,
							"failed to parse feePayer %s from ExtensionOptionsWeb3Tx", extOpt.FeePayer)
						return ctx, err
					}

					feeDelegated = true
				}
			}
		}
	}

	if !feeDelegated {
		feePayer = feeTx.FeePayer()
	}

	feePayerAcc := dfd.ak.GetAccount(ctx, feePayer)
	if feePayerAcc == nil {
		return ctx, errors.Wrapf(sdkerrors.ErrUnknownAddress, "fee payer address: %s does not exist", feePayer)
	}

	var priority int64

	fee := feeTx.GetFee()
	if !simulate {
		fee, priority, err = dfd.txFeeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}

	// deduct the fees
	if !fee.IsZero() {
		err = DeductFees(dfd.bankKeeper, ctx, feePayerAcc, fee)
		if err != nil {
			return ctx, err
		}
	}

	newCtx = ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

// DeductFees deducts fees from the given account.
func DeductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc sdk.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return errors.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, acc.GetAddress().String()),
		),
	}
	ctx.EventManager().EmitEvents(events)
	return nil
}
