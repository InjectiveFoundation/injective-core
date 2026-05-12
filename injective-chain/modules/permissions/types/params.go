package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// ParamTable
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(contractHookQueryMaxGas uint64) Params {
	return Params{
		ContractHookMaxGas: contractHookQueryMaxGas,
	}
}

// default module parameters.
func DefaultParams() Params {
	return Params{
		ContractHookMaxGas: 15_000_000,
	}
}

// validate params.
func (p Params) Validate() error {
	for _, contract := range p.GetEnforcedRestrictionsEvmContracts() {
		if !ethcommon.IsHexAddress(contract.ContractAddress) {
			return fmt.Errorf("is not valid EVM address: %s", contract.ContractAddress)
		}
	}
	return nil
}

// Implements params.ParamSet.
func (*Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}
