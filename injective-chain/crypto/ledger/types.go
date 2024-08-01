package ledger

import (
	"crypto/ecdsa"

	gethaccounts "github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
)

type Account struct {
	Address common.Address
	PubKey  *ecdsa.PublicKey
}

// Wallet represents a software or hardware wallet that might contain one or more
// accounts (derived from the same seed).
type Wallet interface {
	// Open initializes access to a wallet instance. It is not meant to unlock or
	// decrypt account keys, rather simply to establish a connection to hardware
	// wallets and/or to access derivation seeds.
	//
	// The passphrase parameter may or may not be used by the implementation of a
	// particular wallet instance. The reason there is no password-less open method
	// is to strive towards a uniform wallet handling, oblivious to the different
	// backend providers.
	//
	// Please note, if you open a wallet, you must close it to release any allocated
	// resources (especially important when working with hardware wallets).
	Open(passphrase string) error

	// Close releases any resources held by an open wallet instance.
	Close() error

	// Derive attempts to explicitly derive a hierarchical deterministic account at
	// the specified derivation path. If requested, the derived account will be added
	// to the wallet's tracked account list.
	Derive(path gethaccounts.DerivationPath, pin bool) (Account, error)

	// SignTypedData signs a typed data payload using EIP-712 encoding
	SignTypedData(account Account, typedData []byte) ([]byte, error)
}
