package hub

import (
	"errors"
	"runtime"
	"sync"
	"time"

	gethaccounts "github.com/ethereum/go-ethereum/accounts"
	"github.com/zondax/hid"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ledger"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ledger/driver"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ledger/wallet"
)

const (
	LedgerScheme = "ledger"

	VendorID   = 0x2c97
	UsageID    = 0xffa0
	EndpointID = 0

	refreshWalletCooldown = time.Millisecond * 500

	IsLinuxOS = runtime.GOOS == "linux"
)

var (
	LedgerProductIDs = []uint16{
		// Device definitions taken from
		// https://github.com/LedgerHQ/ledger-live/blob/38012bc8899e0f07149ea9cfe7e64b2c146bc92b/libs/ledgerjs/packages/devices/src/index.ts

		// Original product IDs
		0x0000, /* Ledger Blue */
		0x0001, /* Ledger Nano S */
		0x0004, /* Ledger Nano X */
		0x0005, /* Ledger Nano S Plus */
		0x0006, /* Ledger Nano FTS */

		0x0015, /* HID + U2F + WebUSB Ledger Blue */
		0x1015, /* HID + U2F + WebUSB Ledger Nano S */
		0x4015, /* HID + U2F + WebUSB Ledger Nano X */
		0x5015, /* HID + U2F + WebUSB Ledger Nano S Plus */
		0x6015, /* HID + U2F + WebUSB Ledger Nano FTS */

		0x0011, /* HID + WebUSB Ledger Blue */
		0x1011, /* HID + WebUSB Ledger Nano S */
		0x4011, /* HID + WebUSB Ledger Nano X */
		0x5011, /* HID + WebUSB Ledger Nano S Plus */
		0x6011, /* HID + WebUSB Ledger Nano FTS */
	}
)

type Hub struct {
	scheme       string                      // Protocol scheme prefixing account and wallet URLs.
	vendorID     uint16                      // USB vendor identifier used for device discovery
	productIDs   []uint16                    // USB product identifiers used for device discovery
	usageID      uint16                      // USB usage page identifier used for macOS device discovery
	endpointID   int                         // USB endpoint identifier used for non-macOS device discovery
	makeDriverFn func() *driver.LedgerDriver // Factory method to construct a vendor specific driver

	refreshed time.Time        // Time instance when the list of wallets was last refreshed
	wallets   []*wallet.Wallet // List of USB wallet devices currently tracking

	hubMux sync.RWMutex // Protects the internals of the hub from race access

	// TODO(karalabe): remove if hotplug lands on Windows
	commsPend int        // Number of operations blocking enumeration
	commsLock sync.Mutex // Lock protecting the pending counter and enumeration
}

func NewLedgerHub() (*Hub, error) {
	if !hid.Supported() {
		return nil, errors.New("unsupported platform")
	}

	hub := &Hub{
		scheme:       LedgerScheme,
		vendorID:     VendorID,
		productIDs:   LedgerProductIDs,
		usageID:      UsageID,
		endpointID:   EndpointID,
		makeDriverFn: driver.NewLedgerDriver,
	}

	hub.refreshWallets()

	return hub, nil
}

func (h *Hub) refreshWallets() {
	if !h.refreshCooldownPassed() {
		return
	}

	// Retrieve the current list of USB wallet devices
	devices := h.loadDevices()
	if len(devices) == 0 {
		return
	}

	h.updateWallets(devices)
}

func (h *Hub) refreshCooldownPassed() bool {
	// Don't scan the USB like crazy it the user fetches wallets in a loop
	h.hubMux.RLock()
	defer h.hubMux.RUnlock()

	return time.Since(h.refreshed) > refreshWalletCooldown
}

func (h *Hub) loadDevices() []hid.DeviceInfo {
	if IsLinuxOS {
		// hidapi on Linux opens the device during enumeration to retrieve some infos,
		// breaking the Ledger protocol if that is waiting for user confirmation. This
		// is a bug acknowledged at Ledger, but it won't be fixed on old devices, so we
		// need to prevent concurrent comms ourselves. The more elegant solution would
		// be to ditch enumeration in favor of hotplug events, but that don't work yet
		// on Windows so if we need to hack it anyway, this is more elegant for now.
		return h.loadVendorDevicesOnLinux()
	}

	return h.loadVendorDevices()
}

func (h *Hub) loadVendorDevicesOnLinux() []hid.DeviceInfo {
	h.commsLock.Lock()
	defer h.commsLock.Unlock()

	if h.commsPend > 0 {
		return nil
	}

	return h.loadVendorDevices()
}

func (h *Hub) loadVendorDevices() []hid.DeviceInfo {
	deviceInfos := hid.Enumerate(h.vendorID, 0)
	if len(deviceInfos) == 0 {
		return nil
	}

	var devices []hid.DeviceInfo
	for _, info := range deviceInfos {
		for _, id := range h.productIDs {
			// Windows and macOS use UsageID matching, Linux uses Interface matching
			if info.ProductID == id && (info.UsagePage == h.usageID || info.Interface == h.endpointID) {
				devices = append(devices, info)
				break
			}
		}
	}

	return devices
}

func (h *Hub) updateWallets(deviceInfos []hid.DeviceInfo) {
	// Transform the current list of wallets into the new one
	h.hubMux.Lock()
	defer h.hubMux.Unlock()

	wallets := make([]*wallet.Wallet, 0, len(deviceInfos))

	for _, info := range deviceInfos {
		url := gethaccounts.URL{
			Scheme: h.scheme,
			Path:   info.Path,
		}

		// Drop wallets in front of the next device or those that failed for some reason
		for len(h.wallets) > 0 {
			// Abort if we're past the current device and found an operational one
			_, err := h.wallets[0].Status()
			if h.wallets[0].URL().Cmp(url) >= 0 || err == nil {
				break
			}

			// Drop the stale and failed devices
			h.wallets = h.wallets[1:]
		}

		// If there are no more wallets or the device is before the next, wrap new wallet
		if len(h.wallets) == 0 || h.wallets[0].URL().Cmp(url) > 0 {
			wallets = append(wallets, wallet.NewLedgerWallet(h, h.makeDriverFn(), url, info))
			continue
		}

		// If the device is the same as the first wallet, keep it
		if h.wallets[0].URL().Cmp(url) == 0 {
			wallets = append(wallets, h.wallets[0])
			h.wallets = h.wallets[1:]
			continue
		}
	}

	h.refreshed = time.Now().UTC()
	h.wallets = wallets
}

func (h *Hub) AddPendingConfirmation() {
	h.hubMux.Lock()
	defer h.hubMux.Unlock()

	h.commsPend++
}

func (h *Hub) RemovePendingConfirmation() {
	h.hubMux.Lock()
	defer h.hubMux.Unlock()

	h.commsPend--
}

func (h *Hub) Wallets() []ledger.Wallet {
	h.refreshWallets()

	h.hubMux.RLock()
	defer h.hubMux.RUnlock()

	wallets := make([]ledger.Wallet, 0, len(h.wallets))
	for _, w := range h.wallets {
		wallets = append(wallets, w)
	}

	return wallets
}
