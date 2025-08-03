package keeper

import (
	"math/big"

	cosmostracing "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/tracing"

	errorsmod "cosmossdk.io/errors"
	rpctypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/rpc/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/statedb"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// EVMBlockConfig encapsulates the common parameters needed to execute an EVM message,
// it's cached in object store during the block execution.
type EVMBlockConfig struct {
	Params      types.Params
	ChainConfig *params.ChainConfig
	CoinBase    common.Address

	// not supported, always zero
	Random *common.Hash
	// unused, always zero
	Difficulty *big.Int
	// cache the big.Int version of block number, avoid repeated allocation
	BlockNumber *big.Int
	BlockTime   uint64
	Rules       params.Rules
}

// EVMConfig encapsulates common parameters needed to create an EVM to execute a message
// It's mainly to reduce the number of method parameters
type EVMConfig struct {
	*EVMBlockConfig
	TxConfig       statedb.TxConfig
	Tracer         *cosmostracing.Hooks
	DebugTrace     bool
	Overrides      *rpctypes.StateOverride
	BlockOverrides *rpctypes.BlockOverrides
}

// EVMBlockConfig creates the EVMBlockConfig based on current state
func (k *Keeper) EVMBlockConfig(ctx sdk.Context) (*EVMBlockConfig, error) {
	if k.blockParamsCache != nil {
		return k.blockParamsCache, nil
	}

	evmParams := k.GetParams(ctx)
	ethCfg := evmParams.ChainConfig.EthereumConfig()

	// get the coinbase address from the block proposer
	coinbase, err := k.GetCoinbaseAddress(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to obtain coinbase address")
	}

	blockTime := uint64(ctx.BlockHeader().Time.Unix())
	blockNumber := big.NewInt(ctx.BlockHeight())
	rules := ethCfg.Rules(blockNumber, ethCfg.MergeNetsplitBlock != nil, blockTime)

	var zero common.Hash
	cfg := &EVMBlockConfig{
		Params:      evmParams,
		ChainConfig: ethCfg,
		CoinBase:    coinbase,
		Difficulty:  big.NewInt(0),
		Random:      &zero,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		Rules:       rules,
	}
	k.blockParamsCache = cfg
	return cfg, nil
}

func (k *Keeper) RemoveParamsCache(ctx sdk.Context) {
	k.blockParamsCache = nil
}

// EVMConfig creates the EVMConfig based on current state
func (k *Keeper) EVMConfig(ctx sdk.Context, txHash common.Hash) (*EVMConfig, error) {
	blockCfg, err := k.EVMBlockConfig(ctx)
	if err != nil {
		return nil, err
	}

	var txConfig statedb.TxConfig
	if txHash == (common.Hash{}) {
		txConfig = statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))
	} else {
		txConfig = k.TxConfig(ctx, txHash)
	}

	cfg := &EVMConfig{
		EVMBlockConfig: blockCfg,
		TxConfig:       txConfig,
		Tracer:         k.evmTracer,
	}

	return cfg, nil
}

// TxConfig loads `TxConfig` from current transient storage
func (k *Keeper) TxConfig(ctx sdk.Context, txHash common.Hash) statedb.TxConfig {
	return statedb.NewTxConfig(
		common.BytesToHash(ctx.HeaderHash()), // BlockHash
		txHash,                               // TxHash
		0, 0,
	)
}

// VMConfig creates an EVM configuration from the debug setting and the extra EIPs enabled on the
// module parameters. The config generated uses the default JumpTable from the EVM.
func (k Keeper) VMConfig(ctx sdk.Context, cfg *EVMConfig) vm.Config {
	vmCfg := vm.Config{
		NoBaseFee: true,
		ExtraEips: cfg.Params.EIPs(),
	}

	if vmCfg.Tracer == nil && cfg.Tracer != nil && cfg.Tracer.Hooks != nil {
		vmCfg.Tracer = cfg.Tracer.Hooks
	}

	return vmCfg
}
