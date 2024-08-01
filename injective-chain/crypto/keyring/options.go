package keyring

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmoshd "github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/hd"
)

// AppName defines the Ledger app used for signing
const AppName = "Ethereum"

var (
	// SupportedAlgorithms defines the list of signing algorithms used on Injective:
	//  - eth_secp256k1 (Ethereum)
	//  - secp256k1 (Tendermint)
	SupportedAlgorithms = keyring.SigningAlgoList{hd.EthSecp256k1, cosmoshd.Secp256k1}

	// SupportedAlgorithmsLedger defines the list of signing algorithms used on Evmos for the Ledger device:
	//  - secp256k1 (in order to comply with Cosmos SDK)
	// The Ledger derivation function is responsible for all signing and address generation
	SupportedAlgorithmsLedger = keyring.SigningAlgoList{hd.EthSecp256k1}

	// LedgerDerivation defines the Ledger Go derivation (Ethereum app with EIP-712 signing)
	LedgerDerivation = DiscoverLedgerFn

	// CreatePubkey uses the ethsecp256k1 pubkey with Ethereum address generation and keccak hashing
	CreatePubkey = func(key []byte) types.PubKey { return &ethsecp256k1.PubKey{Key: key} }

	// SkipDERConversion represents whether the signed Ledger output should skip conversion from DER to BER.
	// This is set to true for signing performed by the Ledger Ethereum app
	SkipDERConversion = true
)

// EthSecp256k1Option defines a function keys options for the ethereum Secp256k1 curve.
// It supports eth_secp256k1 keys for accounts
func EthSecp256k1Option() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
		options.LedgerDerivation = LedgerDerivation
		options.LedgerCreateKey = CreatePubkey
		options.LedgerAppName = AppName
		options.LedgerSigSkipDERConv = SkipDERConversion
	}
}

func LedgerPreprocessTxHook(
	chainID string,
	key keyring.KeyType,
	tx client.TxBuilder,
) error {
	if key != keyring.TypeLedger {
		return nil
	}

	extTxBuilder, ok := tx.(client.ExtendedTxBuilder)
	if !ok {
		return nil
	}

	var typedDataChainID uint64
	if isMainnet := chainID == "injective-1"; isMainnet {
		typedDataChainID = 1
	}

	if isNonMainnet := chainID == "injective-777" || chainID == "injective-888"; isNonMainnet {
		typedDataChainID = 11155111
	}

	if typedDataChainID == 0 {
		return errors.New("unknown chain id")
	}

	extOptions := &chaintypes.ExtensionOptionsWeb3Tx{
		TypedDataChainID: typedDataChainID,
	}

	option, err := codectypes.NewAnyWithValue(extOptions)
	if err != nil {
		return err
	}

	extTxBuilder.SetExtensionOptions(option)

	sigs, err := tx.GetTx().GetSignaturesV2()
	if err != nil {
		return err
	}

	singleSig, isSingle := sigs[0].Data.(*signing.SingleSignatureData)
	if isSingle {
		singleSig.SignMode = signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON
		if err := tx.SetSignatures(sigs[0]); err != nil {
			return err
		}
	}

	return nil
}
