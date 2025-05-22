package keeper

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
)

// GetCoinbaseAddress returns the block proposer's validator operator address.
func (k Keeper) GetCoinbaseAddress(ctx sdk.Context) (common.Address, error) {
	proposerAddress := sdk.ConsAddress(ctx.BlockHeader().ProposerAddress)
	if len(proposerAddress) == 0 {
		// it's ok that proposer address don't exsits in some contexts like CheckTx.
		return common.Address{}, nil
	}
	validator, err := k.stakingKeeper.GetValidatorByConsAddr(ctx, proposerAddress)
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(
			stakingtypes.ErrNoValidatorFound,
			"failed to retrieve validator from block proposer address %s, %s",
			proposerAddress.String(), err.Error(),
		)
	}

	bz, err := sdk.ValAddressFromBech32(validator.GetOperator())
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(
			err,
			"failed to convert validator operator address %s to bytes",
			validator.GetOperator(),
		)
	}

	return common.BytesToAddress(bz), nil
}

// GetProposerAddress returns current block proposer's address when provided proposer address is empty.
func GetProposerAddress(ctx sdk.Context, proposerAddress sdk.ConsAddress) sdk.ConsAddress {
	if len(proposerAddress) == 0 {
		proposerAddress = ctx.BlockHeader().ProposerAddress
	}
	return proposerAddress
}

// DeductTxCostsFromUserBalance deducts the fees from the user balance. Returns an
// error if the specified sender address does not exist or the account balance is not sufficient.
func (k *Keeper) DeductTxCostsFromUserBalance(
	ctx sdk.Context,
	fees sdk.Coins,
	from common.Address,
) error {
	// fetch sender account
	signerAcc, err := authante.GetSignerAcc(ctx, k.accountKeeper, from.Bytes())
	if err != nil {
		return errorsmod.Wrapf(err, "account not found for sender %s", from)
	}

	// deduct the full gas cost from the user balance
	if err := DeductFees(k.bankKeeper, ctx, signerAcc, fees); err != nil {
		return errorsmod.Wrapf(err, "failed to deduct full gas cost %s from the user %s balance", fees, from)
	}

	return nil
}

// VerifyFee is used to return the fee for the given transaction data in sdk.Coins.
// It checks that the gas limit is not reached, and that the gas limit is higher
// than the intrinsic gas.
func VerifyFee(
	msg *types.MsgEthereumTx,
	denom string,
	homestead, istanbul, shanghai, isCheckTx bool,
) (sdk.Coins, error) {
	tx := msg.AsTransaction()
	isContractCreation := tx.To() == nil

	gasLimit := tx.Gas()

	accessList := tx.AccessList()
	intrinsicGas, err := core.IntrinsicGas(tx.Data(), accessList, isContractCreation, homestead, istanbul, shanghai)
	if err != nil {
		return nil, errorsmod.Wrapf(
			err,
			"failed to retrieve intrinsic gas, contract creation = %t; homestead = %t, istanbul = %t, shanghai = %t",
			isContractCreation, homestead, istanbul, shanghai,
		)
	}

	// intrinsic gas verification during CheckTx
	if isCheckTx && gasLimit < intrinsicGas {
		return nil, errorsmod.Wrapf(
			errortypes.ErrOutOfGas,
			"gas limit too low: %d (gas limit) < %d (intrinsic gas)", gasLimit, intrinsicGas,
		)
	}

	feeAmt := msg.GetFee()
	if feeAmt.Sign() == 0 {
		// zero fee, no need to deduct
		return sdk.Coins{}, nil
	}

	return sdk.Coins{{Denom: denom, Amount: sdkmath.NewIntFromBigInt(feeAmt)}}, nil
}

// CheckSenderBalance validates that the tx cost value is positive and that the
// sender has enough funds to pay for the fees and value of the transaction.
func CheckSenderBalance(
	balance sdkmath.Int,
	tx *ethtypes.Transaction,
) error {
	cost := tx.Cost()

	if cost.Sign() < 0 {
		return errorsmod.Wrapf(
			errortypes.ErrInvalidCoins,
			"tx cost (%s) is negative and invalid", cost,
		)
	}

	if balance.IsNegative() || balance.BigInt().Cmp(cost) < 0 {
		return errorsmod.Wrapf(
			errortypes.ErrInsufficientFunds,
			"sender balance < tx cost (%s < %s)", balance, tx.Cost(),
		)
	}
	return nil
}

// DeductFees deducts fees from the given account.
func DeductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc sdk.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errorsmod.Wrapf(errortypes.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}
	if ctx.BlockHeight() > 0 {
		if err := bankKeeper.SendCoinsFromAccountToModuleVirtual(ctx, acc.GetAddress(), authtypes.FeeCollectorName, fees); err != nil {
			return errorsmod.Wrapf(errortypes.ErrInsufficientFunds, "failed to deduct fees from account %s: %s", acc.GetAddress(), err.Error())
		}
	}
	return nil
}
