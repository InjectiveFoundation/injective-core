package cosmos

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/hd"
	injcodec "github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos/codec"
)

const (
	DefaultKeyName = "validator"
)

type KeyringConfig struct {
	KeyringDir,
	KeyringAppName,
	KeyringBackend,
	KeyFrom,
	KeyPassphrase,
	PrivateKey string
	UseLedger bool
}

type Keyring struct {
	keyring.Keyring

	Addr cosmostypes.AccAddress
}

func NewKeyring(cfg KeyringConfig) (Keyring, error) {
	if withPK := cfg.PrivateKey != ""; withPK {
		return newInMemoryKeyring(cfg)
	}

	return newKeyringFromDir(cfg)
}

func newInMemoryKeyring(cfg KeyringConfig) (Keyring, error) {
	if cfg.UseLedger {
		return Keyring{}, errors.New("cannot use both private key and Ledger")
	}

	pk := cfg.PrivateKey
	if strings.HasPrefix(pk, "0x") {
		pk = pk[2:]
	}

	pkRaw, err := hex.DecodeString(pk)
	if err != nil {
		return Keyring{}, errors.Wrap(err, "invalid private key")
	}

	var (
		cosmosPK   = &ethsecp256k1.PrivKey{Key: pkRaw}
		cosmosAddr = cosmostypes.AccAddress(cosmosPK.PubKey().Address())
		keyName    = DefaultKeyName
	)

	if len(cfg.KeyFrom) > 0 {
		from, err := cosmostypes.AccAddressFromBech32(cfg.KeyFrom)
		if err != nil {
			keyName = cfg.KeyFrom // use it as key name
		} else if !bytes.Equal(from.Bytes(), cosmosAddr.Bytes()) {
			return Keyring{}, errors.Errorf("expected account address %s but got %s from the private key", from.String(), cosmosAddr.String())
		}
	}

	// Create a temporary in-mem keyring for cosmosPK.
	// Allows to init Context when the key has been provided in plaintext and parsed.
	tmpPhrase := randPhrase(64)
	armored := cosmoscrypto.EncryptArmorPrivKey(cosmosPK, tmpPhrase, cosmosPK.Type())

	cdc := injcodec.Codec()
	kr := keyring.NewInMemory(cdc, hd.EthSecp256k1Option())
	if err := kr.ImportPrivKey(keyName, armored, tmpPhrase); err != nil {
		return Keyring{}, errors.Wrap(err, "failed to import private key")
	}

	k := Keyring{
		Keyring: kr,
		Addr:    cosmosAddr,
	}

	return k, nil
}

func newKeyringFromDir(cfg KeyringConfig) (Keyring, error) {
	if len(cfg.KeyFrom) == 0 {
		return Keyring{}, errors.New("insufficient cosmos details provided")
	}

	keyringDir := cfg.KeyringDir
	if !filepath.IsAbs(keyringDir) {
		dir, err := filepath.Abs(keyringDir)
		if err != nil {
			return Keyring{}, errors.Wrap(err, "failed to get absolute path of keyring dir")
		}

		keyringDir = dir
	}

	var reader io.Reader = os.Stdin
	if len(cfg.KeyPassphrase) > 0 {
		reader = newPassReader(cfg.KeyPassphrase)
	}

	cdc := injcodec.Codec()
	kr, err := keyring.New(
		cfg.KeyringAppName,
		cfg.KeyringBackend,
		keyringDir,
		reader,
		cdc,
		hd.EthSecp256k1Option(),
	)

	if err != nil {
		return Keyring{}, errors.Wrap(err, "failed to initialize cosmos keyring")
	}

	var keyRecord *keyring.Record
	if cosmosAddr, err := cosmostypes.AccAddressFromBech32(cfg.KeyFrom); err != nil {
		r, err := kr.Key(cfg.KeyFrom)
		if err != nil {
			return Keyring{}, err
		}

		keyRecord = r
	} else {
		r, err := kr.KeyByAddress(cosmosAddr)
		if err != nil {
			return Keyring{}, err
		}

		keyRecord = r
	}

	switch keyRecord.GetType() {
	case keyring.TypeLocal:
		// kb has a key and it's totally usable
		addr, err := keyRecord.GetAddress()
		if err != nil {
			return Keyring{}, errors.Wrap(err, "failed to get address from key record")
		}

		k := Keyring{
			Keyring: kr,
			Addr:    addr,
		}

		return k, nil
	case keyring.TypeLedger:
		// the kb stores references to ledger keys, so we must explicitly
		// check that. kb doesn't know how to scan HD keys - they must be added manually before
		if !cfg.UseLedger {
			return Keyring{}, errors.Errorf("key %s is a Ledger reference, enable Ledger option", keyRecord.Name)
		}

		addr, err := keyRecord.GetAddress()
		if err != nil {
			return Keyring{}, errors.Wrap(err, "failed to get address from key record")

		}

		k := Keyring{
			Keyring: kr,
			Addr:    addr,
		}

		return k, nil
	default:
		return Keyring{}, errors.Errorf("unsupported key type: %s", keyRecord.GetType())
	}
}

func randPhrase(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic("rand failed")
	}

	return string(buf)
}

var _ io.Reader = &passReader{}

type passReader struct {
	pass string
	buf  *bytes.Buffer
}

func newPassReader(pass string) io.Reader {
	return &passReader{
		pass: pass,
		buf:  new(bytes.Buffer),
	}
}

func (r *passReader) Read(p []byte) (n int, err error) {
	n, err = r.buf.Read(p)
	if err == io.EOF || n == 0 {
		r.buf.WriteString(r.pass + "\n")

		n, err = r.buf.Read(p)
	}

	return
}
