package keeper

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

type OcrHooks interface {
	SetHooks(
		h types.OcrHooks,
	) Keeper
}

// Set the hooks
func (k *keeper) SetHooks(h types.OcrHooks) Keeper {
	if k.hooks != nil {
		panic("cannot set hooks twice")
	}

	k.hooks = h

	return k
}
