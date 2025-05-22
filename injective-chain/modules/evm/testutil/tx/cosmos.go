package tx

import (
	"math"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	"github.com/InjectiveLabs/injective-core/injective-chain/app"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
)

var (
	feeAmt     = math.Pow10(16)
	DefaultFee = sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewIntFromUint64(uint64(feeAmt)))
)

// CosmosTxArgs contains the params to create a cosmos tx
type CosmosTxArgs struct {
	// TxCfg is the client transaction config
	TxCfg client.TxConfig
	// Priv is the private key that will be used to sign the tx
	Priv cryptotypes.PrivKey
	// ChainID is the chain's id on cosmos format, e.g. 'injective-999'
	ChainID string
	// Gas to be used on the tx
	Gas uint64
	// GasPrice to use on tx
	GasPrice *sdkmath.Int
	// Fees is the fee to be used on the tx (amount and denom)
	Fees sdk.Coins
	// FeeGranter is the account address of the fee granter
	FeeGranter sdk.AccAddress
	// Msgs slice of messages to include on the tx
	Msgs []sdk.Msg
}

// PrepareCosmosTx creates a cosmos tx and signs it with the provided messages and private key.
// It returns the signed transaction and an error
func PrepareCosmosTx(
	ctx sdk.Context,
	injApp *app.InjectiveApp,
	args CosmosTxArgs,
) (authsigning.Tx, error) {
	txBuilder := args.TxCfg.NewTxBuilder()

	txBuilder.SetGasLimit(args.Gas)

	var fees sdk.Coins
	if args.GasPrice != nil {
		fees = sdk.Coins{{Denom: evmtypes.DefaultEVMDenom, Amount: args.GasPrice.MulRaw(int64(args.Gas))}}
	} else {
		fees = sdk.Coins{DefaultFee}
	}

	txBuilder.SetFeeAmount(fees)
	if err := txBuilder.SetMsgs(args.Msgs...); err != nil {
		return nil, err
	}

	txBuilder.SetFeeGranter(args.FeeGranter)

	return signCosmosTx(
		ctx,
		injApp,
		args,
		txBuilder,
	)
}

// signCosmosTx signs the cosmos transaction on the txBuilder provided using
// the provided private key
func signCosmosTx(
	ctx sdk.Context,
	injApp *app.InjectiveApp,
	args CosmosTxArgs,
	txBuilder client.TxBuilder,
) (authsigning.Tx, error) {
	addr := sdk.AccAddress(args.Priv.PubKey().Address().Bytes())
	seq, err := injApp.AccountKeeper.GetSequence(ctx, addr)
	if err != nil {
		return nil, err
	}

	signMode, err := authsigning.APISignModeToInternal(args.TxCfg.SignModeHandler().DefaultMode())
	if err != nil {
		return nil, err
	}

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	sigV2 := signing.SignatureV2{
		PubKey: args.Priv.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signMode,
			Signature: nil,
		},
		Sequence: seq,
	}

	sigsV2 := []signing.SignatureV2{sigV2}

	if err := txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	accNumber := injApp.AccountKeeper.GetAccount(ctx, addr).GetAccountNumber()
	signerData := authsigning.SignerData{
		ChainID:       args.ChainID,
		AccountNumber: accNumber,
		Sequence:      seq,
	}
	sigV2, err = tx.SignWithPrivKey(
		ctx,
		signMode,
		signerData,
		txBuilder, args.Priv, args.TxCfg,
		seq,
	)
	if err != nil {
		return nil, err
	}

	sigsV2 = []signing.SignatureV2{sigV2}
	if err := txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}
	return txBuilder.GetTx(), nil
}
