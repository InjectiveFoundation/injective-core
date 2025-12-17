package ante

import (
	"bytes"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// DeductFeeDecorator deducts fees from the first signer of the tx
// If the first signer does not have the funds to pay for the fees, return with InsufficientFunds error
// Call next AnteHandler if fees successfully deducted
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type DeductFeeDecorator struct {
	ak           authante.AccountKeeper
	bankKeeper   authtypes.BankKeeper
	txFeeChecker authante.TxFeeChecker
}

func NewDeductFeeDecorator(ak authante.AccountKeeper, bk authtypes.BankKeeper) DeductFeeDecorator {
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

	if addr := dfd.ak.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", authtypes.FeeCollectorName))
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
func DeductFees(bankKeeper authtypes.BankKeeper, ctx sdk.Context, acc sdk.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), auctiontypes.ModuleName, fees)
	if err != nil {
		return errors.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	// Alternative way (from EVM support): doesn't work for now in wasmx tests.
	// err = bankKeeper.SendCoinsFromAccountToModuleVirtual(ctx, acc.GetAddress(), authtypes.FeeCollectorName, fees)

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

// AuctionFeeDecorator replaces the original cosmos DeductFeeDecorator so fees are sent to the auction module
type AuctionFeeDecorator struct {
	accountKeeper  authante.AccountKeeper
	bankKeeper     authtypes.BankKeeper
	feegrantKeeper authante.FeegrantKeeper
	txFeeChecker   authante.TxFeeChecker
}

func NewAuctionFeeDecorator(
	ak authante.AccountKeeper,
	bk authtypes.BankKeeper,
	fk authante.FeegrantKeeper,
	tfc authante.TxFeeChecker,
) AuctionFeeDecorator {
	if tfc == nil {
		tfc = authante.CheckTxFeeWithValidatorMinGasPrices
	}

	return AuctionFeeDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		txFeeChecker:   tfc,
	}
}

// nolint:revive // ok
func (afd AuctionFeeDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errors.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	var (
		priority int64
		err      error
	)

	fee := feeTx.GetFee()
	if !simulate {
		fee, priority, err = afd.txFeeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}

	if err := afd.checkDeductFee(ctx, feeTx, fee); err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

// nolint:revive // ok
func (afd AuctionFeeDecorator) checkDeductFee(ctx sdk.Context, tx sdk.FeeTx, fee sdk.Coins) error {
	feePayer := tx.FeePayer()
	feeGranter := tx.FeeGranter()
	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		feeGranterAddr := sdk.AccAddress(feeGranter)

		if afd.feegrantKeeper == nil {
			return sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !bytes.Equal(feeGranterAddr, feePayer) {
			err := afd.feegrantKeeper.UseGrantedFees(ctx, feeGranterAddr, feePayer, fee, tx.GetMsgs())
			if err != nil {
				return errors.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranterAddr
	}

	deductFeesFromAcc := afd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	// deduct the fees
	if !fee.IsZero() {
		if !fee.IsValid() {
			return errors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fee)
		}

		if err := afd.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			deductFeesFromAcc.GetAddress(),
			auctiontypes.ModuleName,
			fee,
		); err != nil {
			return errors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, sdk.AccAddress(deductFeesFrom).String()),
		),
	}

	ctx.EventManager().EmitEvents(events)

	return nil
}
