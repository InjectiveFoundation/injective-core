package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/common/vouchers"
	erc20types "github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
	"github.com/InjectiveLabs/metrics/v2"
)

type Keeper struct {
	storeKey       storetypes.StoreKey
	objectStoreKey storetypes.StoreKey
	meter          metrics.Meter

	bankKeeper    types.BankKeeper
	tfKeeper      types.TokenFactoryKeeper
	wasmKeeper    types.WasmKeeper
	evmKeeper     types.EvmKeeper
	accountKeeper types.AccountKeeper

	tfModuleAddress string
	moduleAccounts  map[string]bool
	authority       string

	contractPauseListeners       []types.ContractPauseListener
	contractUnpauseListeners     []types.ContractUnpauseListener
	contractBlacklistListeners   []types.ContractBlacklistListener
	contractUnblacklistListeners []types.ContractUnblacklistListener

	vouchersAssistant *vouchers.VouchersAssistant
}

var (
	enforcedContractsKey = []byte("enforcedContracts")
)

// NewKeeper returns a new instance of the x/permissions keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	tfKeeper types.TokenFactoryKeeper,
	wasmKeeper types.WasmKeeper,
	evmKeeper types.EvmKeeper,
	accountKeeper types.AccountKeeper,
	oKey storetypes.StoreKey,
	tfModuleAddress string,
	moduleAccounts map[string]bool,
	authority string,
) *Keeper {
	k := &Keeper{
		storeKey:        storeKey,
		objectStoreKey:  oKey,
		bankKeeper:      bankKeeper,
		tfKeeper:        tfKeeper,
		wasmKeeper:      wasmKeeper,
		evmKeeper:       evmKeeper,
		tfModuleAddress: tfModuleAddress,
		accountKeeper:   accountKeeper,
		moduleAccounts:  moduleAccounts,
		authority:       authority,
	}
	k.vouchersAssistant = vouchers.NewVouchersAssistant(k, bankKeeper)
	return k
}

// GetVouchersStore returns the KV store prefixed for voucher storage (satisfies vouchers.VoucherKeeper).
func (k Keeper) GetVouchersStore(ctx sdk.Context) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), vouchersKey)
}

// ModuleName returns the permissions module name (satisfies vouchers.VoucherKeeper).
func (Keeper) ModuleName() string { return types.ModuleName }

// EmitSetVoucherEvent emits the permissions-module EventSetVoucher (satisfies vouchers.VoucherKeeper).
func (Keeper) EmitSetVoucherEvent(ctx sdk.Context, addr string, voucher sdk.Coin) {
	if err := ctx.EventManager().EmitTypedEvent(&types.EventSetVoucher{
		Addr:    addr,
		Voucher: voucher,
	}); err != nil {
		ctx.Logger().Error("failed to emit EventSetVoucher", "addr", addr, "voucher", voucher, "err", err)
	}
}

// EmitDeleteVoucherEvent emits the permissions-module EventSetVoucher with a zero coin to signal
// deletion (satisfies vouchers.VoucherKeeper). This preserves the existing on-chain event shape.
func (Keeper) EmitDeleteVoucherEvent(ctx sdk.Context, addr, denom string) {
	if err := ctx.EventManager().EmitTypedEvent(&types.EventSetVoucher{
		Addr:    addr,
		Voucher: types.NewEmptyVoucher(denom),
	}); err != nil {
		ctx.Logger().Error("failed to emit EventSetVoucher (delete)", "addr", addr, "denom", denom, "err", err)
	}
}

// Logger returns a logger for the x/permissions module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) Meter(ctx context.Context) metrics.Meter {
	if k.meter == nil {
		k.meter = sdk.UnwrapSDKContext(ctx).Meter().SubMeter(types.ModuleName, metrics.Tag("svc", types.ModuleName))
	}

	return k.meter
}

func (k Keeper) getEnforcedRestrictionsEvmContracts(ctx sdk.Context) []*types.EnforcedContract {
	defer k.Meter(ctx).FuncTiming(&ctx, "getEnforcedRestrictionsEvmContracts")()
	// try to get cached value
	store := ctx.ObjectStore(k.objectStoreKey)
	if val := store.Get(enforcedContractsKey); val != nil {
		return val.([]*types.EnforcedContract) //nolint:revive //ok
	}

	// read from storage, cache and return
	contracts := k.GetParams(ctx).EnforcedRestrictionsEvmContracts
	decodedContracts := make([]*types.EnforcedContract, 0, len(contracts))

	for _, contract := range contracts {
		decodedContract := types.EnforcedContract{
			ContractAddress:    common.HexToAddress(contract.ContractAddress),
			PauseEventId:       crypto.Keccak256Hash([]byte(contract.PauseEventSignature)),
			UnpauseEventId:     crypto.Keccak256Hash([]byte(contract.UnpauseEventSignature)),
			BlacklistEventId:   crypto.Keccak256Hash([]byte(contract.BlacklistEventSignature)),
			UnblacklistEventId: crypto.Keccak256Hash([]byte(contract.UnblacklistEventSignature)),
		}

		decodedContracts = append(decodedContracts, &decodedContract)
	}

	store.Set(enforcedContractsKey, decodedContracts)

	return decodedContracts
}

func (k Keeper) clearCachedEnforcedContracts(ctx sdk.Context) {
	ctx.ObjectStore(k.objectStoreKey).Delete(enforcedContractsKey)
}

func (k Keeper) IsEnforcedRestrictionsDenom(ctx sdk.Context, denom string) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "IsEnforcedRestrictionsDenom")()

	contracts := k.getEnforcedRestrictionsEvmContracts(ctx)
	for i := range contracts {
		if erc20types.DenomPrefix+contracts[i].ContractAddress.Hex() == denom {
			return true
		}
	}
	return false
}

func (k Keeper) PostTxProcessing(ctx sdk.Context, _ *core.Message, receipt *ethtypes.Receipt) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "PostTxProcessing")()

	for _, contract := range k.getEnforcedRestrictionsEvmContracts(ctx) {
		for _, logEntry := range receipt.Logs {
			if err := k.processEnforcedRestrictionsLog(ctx, contract, logEntry); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) processEnforcedRestrictionsLog(ctx sdk.Context, contract *types.EnforcedContract, logEntry *ethtypes.Log) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "processEnforcedRestrictionsLog")()

	if len(logEntry.Topics) == 0 || logEntry.Address.Cmp(contract.ContractAddress) != 0 {
		return nil
	}

	eventID := logEntry.Topics[0]
	contractAddr := contract.ContractAddress.String()

	switch eventID {
	case contract.PauseEventId:
		return k.handlePauseEvent(ctx, contract, contractAddr)
	case contract.UnpauseEventId:
		return k.handleUnpauseEvent(ctx, contract, contractAddr)
	case contract.BlacklistEventId:
		return k.handleBlacklistEvent(ctx, contract, logEntry, contractAddr)
	case contract.UnblacklistEventId:
		return k.handleUnblacklistEvent(ctx, contract, logEntry, contractAddr)
	default:
		return nil
	}
}

func (k Keeper) handlePauseEvent(ctx sdk.Context, contract *types.EnforcedContract, contractAddr string) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handlePauseEvent")()

	k.Logger(ctx).Info("enforced restrictions token pause is detected", "contract_address", contractAddr)

	for _, l := range k.contractPauseListeners {
		if err := l.OnEnforcedRestrictionsEVMContractPause(ctx, contract.ContractAddress); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) handleUnpauseEvent(ctx sdk.Context, contract *types.EnforcedContract, contractAddr string) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleUnpauseEvent")()

	k.Logger(ctx).Info("enforced restrictions token unpause is detected", "contract_address", contractAddr)

	for _, l := range k.contractUnpauseListeners {
		if err := l.OnEnforcedRestrictionsEVMContractUnpause(ctx, contract.ContractAddress); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) handleBlacklistEvent(ctx sdk.Context, contract *types.EnforcedContract, logEntry *ethtypes.Log, contractAddr string) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleBlacklistEvent")()

	account, ok := k.extractAccountFromLog(ctx, logEntry, "blacklist", contractAddr)
	if !ok {
		return nil
	}

	k.Logger(ctx).Info("enforced restrictions token blacklist is detected", "contract_address", contractAddr, "account", account.String())

	for _, l := range k.contractBlacklistListeners {
		if err := l.OnEnforcedRestrictionsEVMContractBlacklist(ctx, contract.ContractAddress, account); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) handleUnblacklistEvent(ctx sdk.Context, contract *types.EnforcedContract, logEntry *ethtypes.Log, contractAddr string) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleUnblacklistEvent")()

	account, ok := k.extractAccountFromLog(ctx, logEntry, "un-blacklist", contractAddr)
	if !ok {
		return nil
	}

	k.Logger(ctx).Info("enforced restrictions token un-blacklist is detected", "contract_address", contractAddr, "account", account.String())

	for _, l := range k.contractUnblacklistListeners {
		if err := l.OnEnforcedRestrictionsEVMContractUnblacklist(ctx, contract.ContractAddress, account); err != nil {
			return err
		}
	}
	return nil
}

// extractAccountFromLog reads the account address from the second topic of the log entry.
// Returns false if the topic is missing.
func (k Keeper) extractAccountFromLog(ctx sdk.Context, logEntry *ethtypes.Log, eventName, contractAddr string) (common.Address, bool) {
	defer k.Meter(ctx).FuncTiming(&ctx, "extractAccountFromLog")()

	if len(logEntry.Topics) < 2 {
		k.Logger(ctx).Warn("enforced restrictions token "+eventName+" is detected but can't derive the account", "contract_address", contractAddr)
		return common.Address{}, false
	}
	return common.BytesToAddress(logEntry.Topics[1].Bytes()), true
}

func (k *Keeper) AddEnforcedRestrictionsEVMContractPauseListener(l types.ContractPauseListener) {
	k.contractPauseListeners = append(k.contractPauseListeners, l)
}
func (k *Keeper) AddEnforcedRestrictionsEVMContractUnpauseListener(l types.ContractUnpauseListener) {
	k.contractUnpauseListeners = append(k.contractUnpauseListeners, l)
}
func (k *Keeper) AddEnforcedRestrictionsEVMContractBlacklistListener(l types.ContractBlacklistListener) {
	k.contractBlacklistListeners = append(k.contractBlacklistListeners, l)
}
func (k *Keeper) AddEnforcedRestrictionsEVMContractUnblacklistListener(l types.ContractUnblacklistListener) {
	k.contractUnblacklistListeners = append(k.contractUnblacklistListeners, l)
}
