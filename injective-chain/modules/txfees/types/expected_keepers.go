package types

import (
	"context"

	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
)

type ConsensusKeeper interface {
	Params(ctx context.Context, _ *consensustypes.QueryParamsRequest) (*consensustypes.QueryParamsResponse, error)
}
