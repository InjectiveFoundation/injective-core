package keeper

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetEthIntrinsicGas returns the intrinsic gas cost for the transaction
func (k *Keeper) GetEthIntrinsicGas(msg *core.Message, rules params.Rules, isContractCreation bool) (uint64, error) {
	return core.IntrinsicGas(
		msg.Data,
		msg.AccessList,
		msg.SetCodeAuthorizations,
		isContractCreation,
		rules.IsHomestead,
		rules.IsIstanbul,
		rules.IsShanghai,
	)
}

// RefundGas transfers the leftover gas to the sender of the message
//
//nolint:all
func (k *Keeper) RefundGas(ctx sdk.Context, msg *core.Message, leftoverGas uint64, denom string) error {
	// DISABLED: due to DoS attack possibility by filling up whole block gas space almost for free due to refunds
	return nil
}

// ResetGasMeterAndConsumeGas reset first the gas meter consumed value to zero and set it back to the new value
// 'gasUsed'
func (k *Keeper) ResetGasMeterAndConsumeGas(ctx sdk.Context, gasUsed uint64) {
	// reset the gas count
	ctx.GasMeter().RefundGas(ctx.GasMeter().GasConsumed(), "reset the gas count")
	ctx.GasMeter().ConsumeGas(gasUsed, "apply evm transaction")
}

// GasToRefund calculates the amount of gas the state machine should refund to the sender. It is
// capped by the refund quotient value.
// Note: do not pass 0 to refundQuotient
func GasToRefund(availableRefund, gasConsumed, refundQuotient uint64) uint64 {
	// Apply refund counter
	refund := gasConsumed / refundQuotient
	if refund > availableRefund {
		return availableRefund
	}
	return refund
}
