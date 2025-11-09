package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"cosmossdk.io/math"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
)

// EVMParams defines the EVM module parameters
type EVMParams struct {
	EvmDenom            string      `json:"evm_denom,omitempty"`
	EnableCreate        bool        `json:"enable_create,omitempty"`
	EnableCall          bool        `json:"enable_call,omitempty"`
	ExtraEIPs           []int64     `json:"extra_eips,omitempty"`
	ChainConfig         ChainConfig `json:"chain_config"`
	AllowUnprotectedTxs bool        `json:"allow_unprotected_txs,omitempty"`
	AuthorizedDeployers []string    `json:"authorized_deployers,omitempty"`
	Permissioned        bool        `json:"permissioned,omitempty"`
}

type ChainConfig struct {
	HomesteadBlock      *math.Int           `json:"homestead_block,omitempty"`
	DAOForkBlock        *math.Int           `json:"dao_fork_block,omitempty"`
	DAOForkSupport      bool                `json:"dao_fork_support,omitempty"`
	EIP150Block         *math.Int           `json:"eip150_block,omitempty"`
	EIP150Hash          string              `json:"eip150_hash,omitempty"`
	EIP155Block         *math.Int           `json:"eip155_block,omitempty"`
	EIP158Block         *math.Int           `json:"eip158_block,omitempty"`
	ByzantiumBlock      *math.Int           `json:"byzantium_block,omitempty"`
	ConstantinopleBlock *math.Int           `json:"constantinople_block,omitempty"`
	PetersburgBlock     *math.Int           `json:"petersburg_block,omitempty"`
	IstanbulBlock       *math.Int           `json:"istanbul_block,omitempty"`
	MuirGlacierBlock    *math.Int           `json:"muir_glacier_block,omitempty"`
	BerlinBlock         *math.Int           `json:"berlin_block,omitempty"`
	LondonBlock         *math.Int           `json:"london_block,omitempty"`
	ArrowGlacierBlock   *math.Int           `json:"arrow_glacier_block,omitempty"`
	GrayGlacierBlock    *math.Int           `json:"gray_glacier_block,omitempty"`
	MergeNetsplitBlock  *math.Int           `json:"merge_netsplit_block,omitempty"`
	ShanghaiTime        *math.Int           `json:"shanghai_time,omitempty"`
	CancunTime          *math.Int           `json:"cancun_time,omitempty"`
	PragueTime          *math.Int           `json:"prague_time,omitempty"`
	EIP155ChainID       *math.Int           `json:"eip155_chain_id,omitempty"`
	BlobScheduleConfig  *BlobScheduleConfig `json:"blob_schedule_config,omitempty"`
}

type BlobScheduleConfig struct {
	Cancun *BlobConfig `json:"cancun,omitempty"`
	Prague *BlobConfig `json:"prague,omitempty"`
	Osaka  *BlobConfig `json:"osaka,omitempty"`
	Verkle *BlobConfig `json:"verkle,omitempty"`
}

type BlobConfig struct {
	Target                uint64 `json:"target,string"`
	Max                   uint64 `json:"max,string"`
	BaseFeeUpdateFraction uint64 `json:"base_fee_update_fraction,string"`
}

// GetEVMParams returns the parameters of the txfees module.
func GetEVMParams(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
) EVMParams {
	type result struct {
		Params EVMParams `json:"params"`
	}

	var paramsResult result
	resp, _, err := node.ExecQuery(ctx, "evm", "params", "--chain-id", node.Chain.Config().ChainID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, json.Unmarshal(resp, &paramsResult))

	return paramsResult.Params
}
