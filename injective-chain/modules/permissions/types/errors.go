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
	ErrWasmHookError            = errors.Register(ModuleName, 12, "wasm hook query error")
	ErrVoucherNotFound          = errors.Register(ModuleName, 13, "voucher was not found")
	ErrInvalidContractHook      = errors.Register(ModuleName, 14, "invalid contract hook")
	ErrUnknownPolicy            = errors.Register(ModuleName, 15, "unknown policy")
	ErrUnauthorizedPolicyChange = errors.Register(ModuleName, 16, "unauthorized policy change")
)
