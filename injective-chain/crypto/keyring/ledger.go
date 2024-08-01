package keyring

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/InjectiveLabs/injective-core/injective-chain/app/ante/typeddata"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ledger"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ledger/hub"
	sdkledger "github.com/cosmos/cosmos-sdk/crypto/ledger"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func DiscoverLedgerFn() (sdkledger.SECP256K1, error) {
	l := &ledgerSECP256K1{}
	return connect(l)
}

var _ sdkledger.SECP256K1 = &ledgerSECP256K1{}

type ledgerSECP256K1 struct {
	hub           *hub.Hub
	primaryWallet ledger.Wallet
}

func connect(l *ledgerSECP256K1) (sdkledger.SECP256K1, error) {
	h, err := hub.NewLedgerHub()
	if err != nil {
		return nil, err
	}

	if h == nil {
		return nil, errors.New("no hardware wallets detected")
	}

	wallets := h.Wallets()
	if len(wallets) == 0 {
		return nil, errors.New("no Ledger devices detected")
	}

	if len(wallets) > 1 {
		return nil, errors.New("multiple Ledger devices detected: connect a single device and retry")
	}

	// Default to use first wallet found
	primaryWallet := wallets[0]

	// Open wallet for the first time. Unlike with other cases, we want to handle the error here.
	if err := primaryWallet.Open(""); err != nil {
		return nil, err
	}

	l.hub = h
	l.primaryWallet = primaryWallet

	return l, nil

}

func (l ledgerSECP256K1) openWallet() error {
	if l.primaryWallet == nil {
		return errors.New("no wallet found")

	}

	// Ignore error returned here, since we open the wallet during init
	_ = l.primaryWallet.Open("")

	return nil
}

// Close closes the associated primary wallet. Any requests on
// the object after a successful Close() should not work
func (l ledgerSECP256K1) Close() error {
	if l.primaryWallet == nil {
		return errors.New("could not close Ledger: no wallet found")
	}

	return l.primaryWallet.Close()
}

// GetPublicKeySECP256K1 returns the public key associated with the address derived from
// the provided hdPath using the primary wallet
func (l ledgerSECP256K1) GetPublicKeySECP256K1(hdPath []uint32) ([]byte, error) {
	if err := l.openWallet(); err != nil {
		return nil, fmt.Errorf("could not get Ledger public key: %w", err)
	}

	account, err := l.primaryWallet.Derive(hdPath, true)
	if err != nil {
		return nil, errors.New("unable to derive public key, please retry")
	}

	return crypto.FromECDSAPub(account.PubKey), nil
}

// GetAddressPubKeySECP256K1 takes in the HD path as well as a "Human Readable Prefix" (HRP, e.g. "evmos")
// to return the public key bytes in secp256k1 format as well as the account address
func (l ledgerSECP256K1) GetAddressPubKeySECP256K1(hdPath []uint32, hrp string) ([]byte, string, error) {
	if err := l.openWallet(); err != nil {
		return nil, "", fmt.Errorf("could not get Ledger address: %w", err)
	}

	account, err := l.primaryWallet.Derive(hdPath, true)
	if err != nil {
		return nil, "", errors.New("unable to derive Ledger address, please open the Ethereum app and retry")
	}

	address, err := sdk.Bech32ifyAddressBytes(hrp, account.Address.Bytes())
	if err != nil {
		return nil, "", err
	}

	return crypto.FromECDSAPub(account.PubKey), address, nil
}

// SignSECP256K1 returns the signature bytes generated from signing a transaction using the EIP712 signature
func (l ledgerSECP256K1) SignSECP256K1(hdPath []uint32, signDocBytes []byte, signMode byte) ([]byte, error) {
	if signMode != 0 {
		return nil, errors.New("signMode must be 0 (LEGACY_AMINO_JSON)")
	}

	fmt.Println("Generating payload, please check your Ledger...")

	if err := l.openWallet(); err != nil {
		return nil, fmt.Errorf("unable to sign with Ledger: %w", err)
	}

	// Derive requested account
	account, err := l.primaryWallet.Derive(hdPath, true)
	if err != nil {
		return nil, errors.New("unable to derive Ledger address, please open the Ethereum app and retry")
	}

	typedData, err := typeddata.GetEIP712TypedDataForMsg(signDocBytes)
	if err != nil {
		return nil, err
	}

	_, data, err := typeddata.ComputeTypedDataAndHash(typedData)
	if err != nil {
		return nil, err
	}

	// Sign with EIP712 signature
	signature, err := l.primaryWallet.SignTypedData(account, data)
	if err != nil {
		return nil, fmt.Errorf("error generating signature, please retry: %w", err)
	}

	if err := verifySignature(account, data, signature); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return signature, nil
}

func verifySignature(account ledger.Account, data, sig []byte) error {
	if len(sig) != crypto.SignatureLength {
		return fmt.Errorf("invalid signature length: %d", len(sig))
	}

	// Copy signature as it would otherwise be modified
	sigCopy := append(make([]byte, 0, len(sig)), sig...)

	// Subtract 27 to match ECDSA standard
	sigCopy[crypto.RecoveryIDOffset] -= 27

	derivedPubkey, err := crypto.Ecrecover(crypto.Keccak256(data), sigCopy)
	if err != nil {
		return err
	}

	if !bytes.Equal(derivedPubkey, crypto.FromECDSAPub(account.PubKey)) {
		return errors.New("unauthorized: invalid signature verification")
	}

	return nil
}
