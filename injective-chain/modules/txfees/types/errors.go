package types

import (
	errorsmod "cosmossdk.io/errors"
)

// DONTCOVER

// x/txfees module errors.
var (
	ErrInvalidFeeToken = errorsmod.Register(ModuleName, 1, "invalid fee token")
	ErrTooManyFeeCoins = errorsmod.Register(ModuleName, 2, "more than one coin in fee")
)
