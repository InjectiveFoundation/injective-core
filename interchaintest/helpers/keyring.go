package helpers

import (
	"crypto/rand"
	"log"

	crypto_cdc "github.com/InjectiveLabs/sdk-go/chain/crypto/codec"
	"github.com/InjectiveLabs/sdk-go/chain/crypto/hd"
	chainsdk "github.com/InjectiveLabs/sdk-go/client/chain"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmcrypto "github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bip39 "github.com/cosmos/go-bip39"
	"github.com/pkg/errors"
)

var InjectiveKeyringOptions = []keyring.Option{
	hd.EthSecp256k1Option(),
}

var cryptoCdc *codec.ProtoCodec

func init() {
	cryptoCdc = InjectiveCryptoCodec()
}

func InjectiveCryptoCodec() *codec.ProtoCodec {
	registry := chainsdk.NewInterfaceRegistry()
	crypto_cdc.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

func NewPrivKeyFromMnemonic(mnemonic string) (ethPrivKey cryptotypes.PrivKey, err error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		err := errors.New("provided memnemonic is not a valid BIP39 mnemonic")
		return nil, err
	}

	pkBytes, err := hd.EthSecp256k1.Derive()(
		mnemonic,
		keyring.DefaultBIP39Passphrase,
		sdk.GetConfig().GetFullBIP44Path(),
	)
	if err != nil {
		err = errors.Wrap(err, "failed to derive ethsecp256k1 private key")
		return nil, err
	}

	ethPrivKey = hd.EthSecp256k1.Generate()(pkBytes)
	return ethPrivKey, nil
}

func NewKeyringFromMnemonic(keyName, mnemonic string) (keyring.Keyring, error) {
	cosmosAccPk, err := NewPrivKeyFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}

	return newKeyringFromPrivKey(cryptoCdc, keyName, cosmosAccPk)
}

func NewMnemonic() string {
	entropy, err := bip39.NewEntropy(256)
	orPanic(err)

	mnemonic, err := bip39.NewMnemonic(entropy)
	orPanic(err)

	return mnemonic
}

// newKeyringFromPrivKey creates a temporary in-mem keyring for a PrivKey.
// Allows to init Context when the key has been provided in plaintext and parsed.
func newKeyringFromPrivKey(cdc codec.Codec, name string, privKey cryptotypes.PrivKey) (keyring.Keyring, error) {
	kb := keyring.NewInMemory(cdc, InjectiveKeyringOptions...)
	tmpPhrase := randPhrase(64)
	armored := cosmcrypto.EncryptArmorPrivKey(privKey, tmpPhrase, privKey.Type())
	err := kb.ImportPrivKey(name, armored, tmpPhrase)
	if err != nil {
		err = errors.Wrap(err, "failed to import privkey")
		return nil, err
	}

	return kb, nil
}

func randPhrase(size int) string {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	orPanic(err)

	return string(buf)
}

func orPanic(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
