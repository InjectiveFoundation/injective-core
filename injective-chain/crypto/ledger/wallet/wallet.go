package wallet

import (
	"crypto/ecdsa"
	"io"
	"sync"
	"time"

	gethaccounts "github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zondax/hid"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ledger"
)

const heartbeatCycle = time.Second * 1

type Hub interface {
	AddPendingConfirmation()
	RemovePendingConfirmation()
}

// Driver defines the vendor specific functionality hardware wallets instances
// must implement to allow using them with the wallet lifecycle management
type Driver interface {
	// Status returns a textual status to aid the user in the current state of the
	// wallet. It also returns an error indicating any failure the wallet might have
	// encountered
	Status() (string, error)

	// Open initializes access to a wallet instance. The passphrase parameter may
	// or may not be used by the implementation of a particular wallet instance
	Open(device io.ReadWriter, passphrase string) error

	// Close releases any resources held by an open wallet instance
	Close() error

	// Heartbeat performs a sanity check against the hardware wallet to see if it
	// is still online and healthy
	Heartbeat() error

	// Derive sends a derivation request to the USB device and returns the Ethereum
	// address located on that path
	Derive(path gethaccounts.DerivationPath) (common.Address, *ecdsa.PublicKey, error)

	// SignTypedMessage sends the message to the Ledger device and waits for the user to sign
	// or deny the transaction
	SignTypedMessage(path gethaccounts.DerivationPath, messageHash []byte, domainHash []byte) ([]byte, error)
}

type Wallet struct {
	hub    Hub
	driver Driver           // Hardware implementation of the low level device operations
	info   hid.DeviceInfo   // Known USB device infos about the wallet
	url    gethaccounts.URL // Textual URL uniquely identifying this wallet

	paths             map[common.Address]gethaccounts.DerivationPath // Known derivation paths for signing operations
	device            *hid.Device                                    // USB device advertising itself as a hardware wallet
	heartbeatLoopQuit chan chan struct{}

	// Locking a hardware wallet is a bit special. Since hardware devices are lower
	// performing, any communication with them might take a non-negligible amount of
	// time. Worse still, waiting for user confirmation can take arbitrarily long,
	// but exclusive communication must be upheld during. Locking the entire wallet
	// in the meantime however would stall any parts of the system that don't want
	// to communicate, just read some state (e.g. list the accounts).
	//
	// As such, a hardware wallet needs two locks to function correctly. A wallet
	// lock can be used to protect the wallet's software-side internal state, which
	// must not be held exclusively during hardware communication. A driver lock
	// can be used to achieve exclusive access to the device itself, this one
	// however should allow "skipping" waiting for operations that might want to
	// use the device, but can live without too (e.g. account self-derivation).
	//
	// Since we have two locks, it's important to know how to properly use them:
	//   - Communication requires the `device` to not change, so obtaining the
	//     driver should be done after having the wallet lock.
	//   - Communication must not disable read access to the wallet state, so it
	//     must only ever hold a *read* lock of the wallet.
	walletMux sync.RWMutex
	driverMux sync.Mutex
}

func NewLedgerWallet(
	hub Hub,
	driver Driver,
	url gethaccounts.URL,
	info hid.DeviceInfo,
) *Wallet {
	return &Wallet{
		hub:    hub,
		driver: driver,
		info:   info,
		url:    url,
	}
}

func (w *Wallet) URL() gethaccounts.URL {
	return w.url
}

func (w *Wallet) Status() (string, error) {
	w.walletMux.RLock() // No device communication, state lock is enough
	defer w.walletMux.RUnlock()

	status, failure := w.driver.Status()
	if w.device == nil {
		return "Closed", failure
	}

	return status, failure
}

func (w *Wallet) Open(pwd string) error {
	w.walletMux.Lock()
	defer w.walletMux.Unlock()

	if w.device != nil {
		return gethaccounts.ErrWalletAlreadyOpen
	}

	device, err := w.info.Open()
	if err != nil {
		return err
	}

	w.device = device
	if err := w.driver.Open(w.device, pwd); err != nil {
		return err
	}

	w.paths = map[common.Address]gethaccounts.DerivationPath{}
	w.heartbeatLoopQuit = make(chan chan struct{})

	go w.heartbeat()

	return nil
}

func (w *Wallet) heartbeat() {
	for {
		select {
		case stopCh := <-w.heartbeatLoopQuit:
			close(stopCh)
			return
		case <-time.After(heartbeatCycle):
			if err := w.checkPulse(); err != nil {
				_ = w.Close()
				return
			}
		}
	}
}

func (w *Wallet) checkPulse() error {
	w.walletMux.RLock()
	defer w.walletMux.RUnlock()

	if w.device == nil {
		return gethaccounts.ErrWalletClosed
	}

	w.driverMux.Lock()
	defer w.driverMux.Unlock()

	return w.driver.Heartbeat()
}

func (w *Wallet) Close() error {
	w.walletMux.Lock()
	defer w.walletMux.Unlock()

	if w.device == nil {
		return gethaccounts.ErrWalletClosed
	}

	// stop heartbeat loop
	stopCh := make(chan struct{})
	w.heartbeatLoopQuit <- stopCh
	<-stopCh

	_ = w.device.Close()
	close(w.heartbeatLoopQuit)

	w.device, w.paths, w.heartbeatLoopQuit = nil, nil, nil

	return w.driver.Close()
}

func (w *Wallet) Derive(path gethaccounts.DerivationPath, pin bool) (ledger.Account, error) {
	// path format
	for i := 0; i < 3; i++ {
		if path[i] < 0x80000000 {
			path[i] += 0x80000000
		}
	}

	addr, pubKey, err := w.deriveAddressAndPubKeyFromPath(path)
	if err != nil {
		return ledger.Account{}, err
	}

	acc := ledger.Account{
		Address: addr,
		PubKey:  pubKey,
	}

	if !pin {
		return acc, nil
	}

	// Pinning needs to modify the state
	w.walletMux.Lock()
	defer w.walletMux.Unlock()

	if _, ok := w.paths[addr]; !ok {
		w.paths[addr] = append(make(gethaccounts.DerivationPath, 0, len(path)), path...)
	}

	return acc, nil
}

func (w *Wallet) deriveAddressAndPubKeyFromPath(path gethaccounts.DerivationPath) (common.Address, *ecdsa.PublicKey, error) {
	w.walletMux.RLock()
	defer w.walletMux.RUnlock()

	if w.device == nil {
		return common.Address{}, nil, gethaccounts.ErrWalletClosed
	}

	w.driverMux.Lock()
	defer w.driverMux.Unlock()

	return w.driver.Derive(path)
}

func (w *Wallet) SignTypedData(account ledger.Account, typedData []byte) ([]byte, error) {
	// only eip712 is supported
	if len(typedData) != 66 || typedData[0] != 0x19 || typedData[1] != 0x01 {
		return nil, gethaccounts.ErrNotSupported
	}

	w.walletMux.RLock()
	defer w.walletMux.RUnlock()

	// If the wallet is closed, abort
	if w.device == nil {
		return nil, gethaccounts.ErrWalletClosed
	}

	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, gethaccounts.ErrUnknownAccount
	}

	w.driverMux.Lock()
	defer w.driverMux.Unlock()

	w.hub.AddPendingConfirmation()
	defer w.hub.RemovePendingConfirmation()

	sig, err := w.driver.SignTypedMessage(path, typedData[2:34], typedData[34:66])
	if err != nil {
		return nil, err
	}

	return sig, nil
}
