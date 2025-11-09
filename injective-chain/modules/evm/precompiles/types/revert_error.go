package types

import (
	"github.com/ethereum/go-ethereum/core/vm"

	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
)

// RevertReasonAndError returns a ExecutionReverted error with revert reason
// that should align with the behavior of go-ethereum implementation.
//
// In the EVM interpreter, an opCall error is reported as ExecutionReverted,
// and its revert reason is stored in EVM memory and then returned by opRevert.
// Since precompiles are also invoked via opCall, they should be handled the same way.
// Therefore, the returned error must be ABI-encoded and returned,
// and the error type changed to ErrExecutionReverted.
//
// related issue: https://github.com/cosmos/evm/issues/223
func RevertReasonAndError(err error) ([]byte, error) {
	revertReasonBz, encErr := evmtypes.RevertReasonBytes(err.Error())
	if encErr != nil {
		return nil, vm.ErrExecutionReverted
	}
	return revertReasonBz, vm.ErrExecutionReverted
}
