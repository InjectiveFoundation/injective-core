package helpers

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewAccAddress(addr []byte) sdk.AccAddress {
	return addr
}
