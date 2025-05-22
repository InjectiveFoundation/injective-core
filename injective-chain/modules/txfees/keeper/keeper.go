package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"

	mempool1559 "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper/mempool-1559"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	consensusKeeper  types.ConsensusKeeper
	dataDir          string
	cachedConsParams cmtproto.ConsensusParams
	CurFeeState      *mempool1559.FeeState

	svcTags   metrics.Tags
	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	consensusKeeper types.ConsensusKeeper,
	dataDir string,
	authority string,
) Keeper {
	return Keeper{
		storeKey:        storeKey,
		cdc:             cdc,
		consensusKeeper: consensusKeeper,
		dataDir:         dataDir,
		// Initialize the EIP state with the default values. They will be updated in the BeginBlocker.
		CurFeeState: mempool1559.DefaultFeeState(),
		authority:   authority,
		svcTags: metrics.Tags{
			"svc": "txfees_k",
		},
	}
}

func (*Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// GetConsParams returns the current consensus parameters from the consensus params store.
func (k *Keeper) GetConsParams(ctx sdk.Context) (*consensustypes.QueryParamsResponse, error) {
	return k.consensusKeeper.Params(ctx, &consensustypes.QueryParamsRequest{})
}
