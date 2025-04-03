package helpers

import (
	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

func ConvertMsgsIntoTx(
	txConfig client.TxConfig,
	feePayer sdk.AccAddress,
	privateKeys []cryptotypes.PrivKey,
	msg []sdk.Msg,
	opts ...func(txBuilder client.TxBuilder),
) (sdk.Tx, error) {
	txBuilder := txConfig.NewTxBuilder()

	err := txBuilder.SetMsgs(msg...)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(txBuilder)
	}

	sigV2s := make([]signing.SignatureV2, len(privateKeys))

	for i, privateKey := range privateKeys {
		sigV2 := signing.SignatureV2{
			PubKey: privateKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  signing.SignMode_SIGN_MODE_EIP712_V2,
				Signature: nil,
			},
			Sequence: 0,
		}

		sigV2s[i] = sigV2
	}

	err = txBuilder.SetSignatures(sigV2s...)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFeePayer(feePayer)
	tx := txBuilder.GetTx()

	return tx, nil
}
