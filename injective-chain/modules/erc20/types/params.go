package types

import (
	"cosmossdk.io/math"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// ParamTable
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(wasmHookQueryMaxGas uint64) Params {
	return Params{}
}

// default erc20 module parameters.
func DefaultParams() Params {
	return Params{
		DenomCreationFee: sdk.NewCoin(chaintypes.InjectiveCoin, math.NewIntWithDecimal(1, 18)), // 1 INJ
	}
}

// validate params.
func (p Params) Validate() error {
	return nil
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}
