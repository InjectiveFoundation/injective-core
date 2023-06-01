package exported

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params/types"
)

type (
	ParamSet = types.ParamSet

	Subspace interface {
		GetParamSet(ctx sdk.Context, set ParamSet)
	}
)
