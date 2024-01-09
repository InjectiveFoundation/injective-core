package ante

import (
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/xlab/suplog"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ethsecp256k1 "github.com/ethereum/go-ethereum/crypto/secp256k1"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/ante/typeddata"

	secp256k1 "github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// Verify all signatures for a tx and return an error if any are invalid. Note,
// the Eip712SigVerificationDecorator decorator will not get executed on ReCheck.
//
// CONTRACT: Pubkeys are set in context for all signers before this decorator runs
// CONTRACT: Tx must implement SigVerifiableTx interface
type Eip712SigVerificationDecorator struct {
	ak              AccountKeeper
	signModeHandler authsigning.SignModeHandler
}

func NewEip712SigVerificationDecorator(ak AccountKeeper, signModeHandler authsigning.SignModeHandler) Eip712SigVerificationDecorator {
	return Eip712SigVerificationDecorator{
		ak:              ak,
		signModeHandler: signModeHandler,
	}
}

func (svd Eip712SigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// no need to verify signatures on recheck tx
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	// stdSigs contains the sequence number, account number, and signatures.
	// When simulating, this would just be a 0-length slice.
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, err
	}

	signerAddrs := sigTx.GetSigners()

	// check that signer length and signature length are the same
	if len(sigs) != len(signerAddrs) {
		return ctx, errors.Wrapf(sdkerrors.ErrUnauthorized, "invalid number of signer;  expected: %d, got %d", len(signerAddrs), len(sigs))
	}

	for i, sig := range sigs {
		acc, err := authante.GetSignerAcc(ctx, svd.ak, signerAddrs[i])
		if err != nil {
			return ctx, err
		}

		// retrieve pubkey
		pubKey := acc.GetPubKey()
		if !simulate && pubKey == nil {
			return ctx, errors.Wrap(sdkerrors.ErrInvalidPubKey, "pubkey on account is not set")
		}

		// Check account sequence number.
		if sig.Sequence != acc.GetSequence() {
			return ctx, errors.Wrapf(
				sdkerrors.ErrWrongSequence,
				"account sequence mismatch, expected %d, got %d", acc.GetSequence(), sig.Sequence,
			)
		}

		// retrieve signer data
		genesis := ctx.BlockHeight() == 0
		chainID := ctx.ChainID()
		var accNum uint64
		if !genesis {
			accNum = acc.GetAccountNumber()
		}
		signerData := authsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNum,
			Sequence:      acc.GetSequence(),
		}

		if !simulate {
			typedData, err := GenerateTypedDataAndVerifySignatureEIP712(pubKey, signerData, sig.Data, tx.(authsigning.Tx))
			if err != nil {
				log.WithError(err).Errorln("Eip712SigVerificationDecorator failed to verify signature")
				errMsg := fmt.Sprintf("signature verification failed: %s; please verify account number (%d) and chain-id (%s)", err.Error(), accNum, chainID)
				if typedData != nil {
					bz, _ := json.Marshal(typedData)
					errMsg = fmt.Sprintf("%s, eip712: %s", errMsg, string(bz))
				}

				return ctx, errors.Wrap(sdkerrors.ErrUnauthorized, errMsg)
			}
		}
	}

	return next(ctx, tx, simulate)
}

var (
	chainTypesCodec codec.ProtoCodecMarshaler
	GlobalCdc       = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

func init() {
	registry := codectypes.NewInterfaceRegistry()
	chaintypes.RegisterInterfaces(registry)
	chainTypesCodec = codec.NewProtoCodec(registry)
}

// VerifySignature verifies a transaction signature contained in SignatureData abstracting over different signing modes
// and single vs multi-signatures.
func GenerateTypedDataAndVerifySignatureEIP712(
	pubKey cryptotypes.PubKey,
	signerData authsigning.SignerData,
	sigData signing.SignatureData,
	tx authsigning.Tx,
) (*typeddata.TypedData, error) {
	switch data := sigData.(type) {
	case *signing.SingleSignatureData:
		var eip712Wrapper EIP712Wrapper
		switch data.SignMode {
		case signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON:
			eip712Wrapper = WrapTxToEIP712
		case signing.SignMode_SIGN_MODE_EIP712_V2:
			eip712Wrapper = WrapTxToEIP712V2
		default:
			return nil, fmt.Errorf("unexpected SignatureData %T: wrong SignMode: %v", sigData, data.SignMode)
		}

		// @contract: this code is reached only when Msg has Web3Tx extension (so this custom Ante handler flow),
		// and the signature is SIGN_MODE_LEGACY_AMINO_JSON which is supported for EIP712 for now
		msgs := tx.GetMsgs()
		var (
			chainID      uint64
			err          error
			feePayer     sdk.AccAddress
			feePayerSig  []byte
			feeDelegated bool
		)

		if txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx); ok {
			if opts := txWithExtensions.GetExtensionOptions(); len(opts) > 0 {
				var optIface txtypes.TxExtensionOptionI
				if err := chainTypesCodec.UnpackAny(opts[0], &optIface); err != nil {
					err = errors.Wrap(err, "failed to proto-unpack ExtensionOptionsWeb3Tx")
					return nil, err
				} else if extOpt, ok := optIface.(*chaintypes.ExtensionOptionsWeb3Tx); ok {
					// chainID in EIP712 typed data is allowed to not match signerData.ChainID,
					// but limited to certain options: 1 (mainnet), 5 (Goerli), thus Metamask will
					// be able to submit signatures without switching networks.

					if extOpt.TypedDataChainID == 1 || extOpt.TypedDataChainID == 5 {
						chainID = extOpt.TypedDataChainID
					}

					if len(extOpt.FeePayer) > 0 {
						feePayer, err = sdk.AccAddressFromBech32(extOpt.FeePayer)
						if err != nil {
							err = errors.Wrap(err, "failed to parse feePayer from ExtensionOptionsWeb3Tx")
							return nil, err
						}

						feePayerSig = extOpt.FeePayerSig
						if len(feePayerSig) == 0 {
							return nil, fmt.Errorf("no feePayerSig provided in ExtensionOptionsWeb3Tx")
						}

						feeDelegated = true
					}
				}
			}
		}

		if chainID == 0 {
			chainID, err = strconv.ParseUint(signerData.ChainID, 10, 64)
			if err != nil {
				err = errors.Wrapf(err, "failed to parse chainID: %s", signerData.ChainID)
				return nil, err
			}
		}

		var typedData typeddata.TypedData
		var sigHash []byte

		feeInfo := legacytx.StdFee{
			Amount: tx.GetFee(),
			Gas:    tx.GetGas(),
		}
		if feeDelegated {
			feeDelegation := &FeeDelegationOptions{
				FeePayer: feePayer,
			}

			typedData, err = eip712Wrapper(GlobalCdc, chainID, &signerData, tx.GetTimeoutHeight(), tx.GetMemo(), feeInfo, msgs, feeDelegation)
			if err != nil {
				return nil, errors.Wrap(err, "failed to pack tx data in EIP712 object")
			}

			sigHash, err = typeddata.ComputeTypedDataHash(typedData)
			if err != nil {
				return &typedData, err
			}

			feePayerPubkey, err := ethsecp256k1.RecoverPubkey(sigHash, feePayerSig)
			if err != nil {
				err = errors.Wrap(err, "failed to recover delegated fee payer from sig")
				return &typedData, err
			}

			ecPubKey, err := ethcrypto.UnmarshalPubkey(feePayerPubkey)
			if err != nil {
				return &typedData, errors.Wrap(err, "failed to unmarshal recovered fee payer pubkey")
			}

			recoveredFeePayerAcc := sdk.AccAddress((&secp256k1.PubKey{
				Key: ethcrypto.CompressPubkey(ecPubKey),
			}).Address().Bytes())

			if !recoveredFeePayerAcc.Equals(feePayer) {
				return &typedData, fmt.Errorf("failed to verify delegated fee payer sig")
			}
		} else {
			typedData, err = eip712Wrapper(GlobalCdc, chainID, &signerData, tx.GetTimeoutHeight(), tx.GetMemo(), feeInfo, msgs, nil)
			if err != nil {
				return &typedData, errors.Wrap(err, "failed to pack tx data in EIP712 object")
			}

			sigHash, err = typeddata.ComputeTypedDataHash(typedData)
			if err != nil {
				return &typedData, err
			}
		}

		if len(data.Signature) != 65 {
			return &typedData, fmt.Errorf("signature length doesnt match typical [R||S||V] signature 65 bytes")
		}

		// VerifySignature of secp256k1 accepts 64 byte signature [R||S]
		// WARNING! Under NO CIRCUMSTANCES try to use pubKey.VerifySignature there
		if !ethsecp256k1.VerifySignature(pubKey.Bytes(), sigHash, data.Signature[:len(data.Signature)-1]) {
			return &typedData, fmt.Errorf("unable to verify signer signature of EIP712 typed data")
		}

		return &typedData, nil
	default:
		return nil, fmt.Errorf("unexpected SignatureData %T", sigData)
	}
}
