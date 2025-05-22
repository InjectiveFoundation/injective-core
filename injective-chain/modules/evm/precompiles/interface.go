package precompiles

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/statedb"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// ExtStateDB defines extra methods of statedb to support stateful precompiled contracts
type ExtStateDB interface {
	vm.StateDB
	ExecuteNativeAction(contract common.Address, converter statedb.EventConverter, action func(ctx sdk.Context) error) error
	Context() sdk.Context
	CacheContext() sdk.Context
}
