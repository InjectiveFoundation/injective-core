package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/tokenfactory module sentinel errors
var (
	ErrDenomNamespaceExists     = errors.Register(ModuleName, 2, "attempting to create a namespace for denom that already exists")
	ErrUnauthorized             = errors.Register(ModuleName, 3, "unauthorized account")
	ErrInvalidGenesis           = errors.Register(ModuleName, 4, "invalid genesis")
	ErrInvalidNamespace         = errors.Register(ModuleName, 5, "invalid namespace")
	ErrInvalidPermission        = errors.Register(ModuleName, 6, "invalid permissions")
	ErrUnknownRole              = errors.Register(ModuleName, 7, "unknown role")
	ErrUnknownWasmHook          = errors.Register(ModuleName, 8, "unknown contract address")
	ErrRestrictedAction         = errors.Register(ModuleName, 9, "restricted action")
	ErrInvalidRole              = errors.Register(ModuleName, 10, "invalid role")
	ErrUnknownDenom             = errors.Register(ModuleName, 11, "namespace for denom does not exist")
	ErrContractHookError        = errors.Register(ModuleName, 12, "contract hook query error")
	ErrVoucherNotFound          = errors.Register(ModuleName, 13, "voucher was not found")
	ErrInvalidWasmHook          = errors.Register(ModuleName, 14, "invalid wasm hook")
	ErrUnknownPolicy            = errors.Register(ModuleName, 15, "unknown policy")
	ErrUnauthorizedPolicyChange = errors.Register(ModuleName, 16, "unauthorized policy change")
	ErrInvalidEVMHook           = errors.Register(ModuleName, 17, "invalid evm hook")
	ErrInvalidERC20Denom        = errors.Register(ModuleName, 18, "invalid erc20 denom")
	ErrContractPostHookError    = errors.Register(ModuleName, 19, "contract post hook execution error")
	ErrInvalidEVMPostHook       = errors.Register(ModuleName, 20, "invalid evm post send hook")
)
