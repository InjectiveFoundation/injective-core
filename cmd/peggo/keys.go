package main

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/common"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/ethereum/keystore"
)

var emptyEthAddress = ethcmn.Address{}

func initEthereumAccountsManager(
	ethChainID uint64,
	ethKeystoreDir *string,
	ethKeyFrom *string,
	ethPassphrase *string,
	ethPrivKey *string,
	ethUseLedger *bool,
) (
	ethKeyFromAddress ethcmn.Address,
	signerFn bind.SignerFn,
	personalSignFn keystore.PersonalSignFn,
	err error,
) {
	switch {
	case *ethUseLedger:
		if ethKeyFrom == nil {
			err := errors.New("cannot use Ledger without from address specified")
			return emptyEthAddress, nil, nil, err
		}

		ethKeyFromAddress = ethcmn.HexToAddress(*ethKeyFrom)
		if ethKeyFromAddress == (ethcmn.Address{}) {
			err = errors.Wrap(err, "failed to parse Ethereum from address")
			return emptyEthAddress, nil, nil, err
		}

		ledgerBackend, err := usbwallet.NewLedgerHub()
		if err != nil {
			err = errors.Wrap(err, "failed to connect with Ethereum app on Ledger device")
			return emptyEthAddress, nil, nil, err
		}

		signerFn = func(from common.Address, tx *ethtypes.Transaction) (*ethtypes.Transaction, error) {
			acc := accounts.Account{
				Address: from,
			}

			wallets := ledgerBackend.Wallets()
			for _, w := range wallets {
				if err := w.Open(""); err != nil {
					err = errors.Wrap(err, "failed to connect to wallet on Ledger device")
					return nil, err
				}

				if !w.Contains(acc) {
					if err := w.Close(); err != nil {
						err = errors.Wrap(err, "failed to disconnect the wallet on Ledger device")
						return nil, err
					}

					continue
				}

				tx, err = w.SignTx(acc, tx, new(big.Int).SetUint64(ethChainID))
				_ = w.Close()
				return tx, err
			}

			return nil, errors.Errorf("account %s not found on Ledger", from.String())
		}

		personalSignFn = func(from common.Address, data []byte) (sig []byte, err error) {
			acc := accounts.Account{
				Address: from,
			}

			wallets := ledgerBackend.Wallets()
			for _, w := range wallets {
				if err := w.Open(""); err != nil {
					err = errors.Wrap(err, "failed to connect to wallet on Ledger device")
					return nil, err
				}

				if !w.Contains(acc) {
					if err := w.Close(); err != nil {
						err = errors.Wrap(err, "failed to disconnect the wallet on Ledger device")
						return nil, err
					}

					continue
				}

				sig, err = w.SignText(acc, data)
				_ = w.Close()
				return sig, err
			}

			return nil, errors.Errorf("account %s not found on Ledger", from.String())
		}

		return ethKeyFromAddress, signerFn, personalSignFn, nil

	case len(*ethPrivKey) > 0:
		ethPk, err := crypto.HexToECDSA(*ethPrivKey)
		if err != nil {
			err = errors.Wrap(err, "failed to hex-decode Ethereum ECDSA Private Key")
			return emptyEthAddress, nil, nil, err
		}

		ethAddressFromPk := ethcrypto.PubkeyToAddress(ethPk.PublicKey)

		if len(*ethKeyFrom) > 0 {
			addr := ethcmn.HexToAddress(*ethKeyFrom)
			if addr == (ethcmn.Address{}) {
				err = errors.Wrap(err, "failed to parse Ethereum from address")
				return emptyEthAddress, nil, nil, err
			} else if addr != ethAddressFromPk {
				err = errors.Wrap(err, "Ethereum from address does not match address from ECDSA Private Key")
				return emptyEthAddress, nil, nil, err
			}
		}

		txOpts, err := bind.NewKeyedTransactorWithChainID(ethPk, new(big.Int).SetUint64(ethChainID))
		if err != nil {
			err = errors.New("failed to init NewKeyedTransactorWithChainID")
			return emptyEthAddress, nil, nil, err
		}

		personalSignFn, err := keystore.PrivateKeyPersonalSignFn(ethPk)
		if err != nil {
			err = errors.New("failed to init PrivateKeyPersonalSignFn")
			return emptyEthAddress, nil, nil, err
		}

		return txOpts.From, txOpts.Signer, personalSignFn, nil

	case len(*ethKeystoreDir) > 0:
		if ethKeyFrom == nil {
			err := errors.New("cannot use Ethereum keystore without from address specified")
			return emptyEthAddress, nil, nil, err
		}

		ethKeyFromAddress = ethcmn.HexToAddress(*ethKeyFrom)
		if ethKeyFromAddress == (ethcmn.Address{}) {
			err = errors.Wrap(err, "failed to parse Ethereum from address")
			return emptyEthAddress, nil, nil, err
		}

		if info, err := os.Stat(*ethKeystoreDir); err != nil || !info.IsDir() {
			err = errors.New("failed to locate keystore dir")
			return emptyEthAddress, nil, nil, err
		}

		ks, err := keystore.New(*ethKeystoreDir)
		if err != nil {
			err = errors.Wrap(err, "failed to load keystore")
			return emptyEthAddress, nil, nil, err
		}

		var pass string
		if len(*ethPassphrase) > 0 {
			pass = *ethPassphrase
		} else {
			pass, err = ethPassFromStdin()
			if err != nil {
				return emptyEthAddress, nil, nil, err
			}
		}

		signerFn, err := ks.SignerFn(ethChainID, ethKeyFromAddress, pass)
		if err != nil {
			err = errors.Wrapf(err, "failed to load key for %s", ethKeyFromAddress)
			return emptyEthAddress, nil, nil, err
		}

		personalSignFn, err := ks.PersonalSignFn(ethKeyFromAddress, pass)
		if err != nil {
			err = errors.Wrapf(err, "failed to load key for %s", ethKeyFromAddress)
			return emptyEthAddress, nil, nil, err
		}

		return ethKeyFromAddress, signerFn, personalSignFn, nil

	default:
		err := errors.New("insufficient ethereum key details provided")
		return emptyEthAddress, nil, nil, err
	}
}

func ethPassFromStdin() (string, error) {
	fmt.Print("Passphrase for Ethereum account: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		err := errors.Wrap(err, "failed to read password from stdin")
		return "", err
	}

	password := string(bytePassword)
	return strings.TrimSpace(password), nil
}

func newPassReader(pass string) io.Reader {
	return &passReader{
		pass: pass,
		buf:  new(bytes.Buffer),
	}
}

type passReader struct {
	pass string
	buf  *bytes.Buffer
}

var _ io.Reader = &passReader{}

func (r *passReader) Read(p []byte) (n int, err error) {
	n, err = r.buf.Read(p)
	if err == io.EOF || n == 0 {
		r.buf.WriteString(r.pass + "\n")

		n, err = r.buf.Read(p)
	}

	return
}
