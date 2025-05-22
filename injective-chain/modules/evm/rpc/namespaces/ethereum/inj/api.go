package inj

import (
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/log"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/rpc/backend"
)

type InjectiveAPI interface {
	GetTxHashByEthHash(common.Hash) (common.Hash, error)
}

var _ InjectiveAPI = (*API)(nil)

// API holds the Injective custom API endpoints
type API struct {
	ctx     *server.Context
	logger  log.Logger
	backend backend.EVMBackend
}

// NewPrivateAPI creates an instance of the Miner API.
func NewInjectiveAPI(
	ctx *server.Context,
	evmBackend backend.EVMBackend,
) *API {
	return &API{
		ctx:     ctx,
		logger:  ctx.Logger.With("api", "inj"),
		backend: evmBackend,
	}
}

// SetEtherbase sets the etherbase of the miner
func (api *API) GetTxHashByEthHash(ethHash common.Hash) (common.Hash, error) {
	api.logger.Debug("inj_GetTxHashByEthHash")
	return api.backend.GetTxHashByEthHash(ethHash)
}
