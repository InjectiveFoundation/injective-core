package tracing

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BlockchainTracerKeyType string

const BlockchainTracerKey = BlockchainTracerKeyType("evm_and_state_logger")

func SetTracingHooks(ctx sdk.Context, hooks *Hooks) sdk.Context {
	return ctx.WithContext(context.WithValue(ctx.Context(), BlockchainTracerKey, hooks))
}

func GetTracingHooks(ctx sdk.Context) *Hooks {
	rawVal := ctx.Context().Value(BlockchainTracerKey)
	if rawVal == nil {
		return nil
	}
	logger, ok := rawVal.(*Hooks)
	if !ok {
		return nil
	}
	return logger
}
