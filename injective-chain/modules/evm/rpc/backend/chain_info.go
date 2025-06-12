package backend

import (
	"fmt"
	gomath "math"
	"math/big"
	"sync"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	rpctypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/rpc/types"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
)

// ChainID is the EIP-155 replay-protection chain id for the current ethereum chain config.
func (b *Backend) ChainID() *hexutil.Big {
	config := b.ChainConfig()
	return (*hexutil.Big)(config.ChainID)
}

// ChainConfig returns the latest ethereum chain configuration
func (b *Backend) ChainConfig() *params.ChainConfig {
	queryParams, err := b.queryClient.Params(b.ctx, &evmtypes.QueryParamsRequest{})
	if err != nil {
		return nil
	}
	return queryParams.Params.ChainConfig.EthereumConfig()
}

// CurrentHeader returns the latest block header
func (b *Backend) CurrentHeader() (*ethtypes.Header, error) {
	return b.HeaderByNumber(rpctypes.EthLatestBlockNumber)
}

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (b *Backend) PendingTransactions() ([]sdk.Tx, error) {
	mc, ok := b.clientCtx.Client.(cmtrpcclient.MempoolClient)
	if !ok {
		return nil, errors.New("invalid rpc client")
	}
	res, err := mc.UnconfirmedTxs(b.ctx, nil)
	if err != nil {
		return nil, err
	}

	result := make([]sdk.Tx, 0, len(res.Txs))
	for _, txBz := range res.Txs {
		tx, err := b.clientCtx.TxConfig.TxDecoder()(txBz)
		if err != nil {
			return nil, err
		}
		result = append(result, tx)
	}

	return result, nil
}

// GetCoinbase is the address that staking rewards will be send to (alias for Etherbase).
func (b *Backend) GetCoinbase() (sdk.AccAddress, error) {
	node, err := b.clientCtx.GetNode()
	if err != nil {
		return nil, err
	}

	status, err := node.Status(b.ctx)
	if err != nil {
		return nil, err
	}

	req := &evmtypes.QueryValidatorAccountRequest{
		ConsAddress: sdk.ConsAddress(status.ValidatorInfo.Address).String(),
	}

	res, err := b.queryClient.ValidatorAccount(b.ctx, req)
	if err != nil {
		return nil, err
	}

	address, _ := sdk.AccAddressFromBech32(res.AccountAddress)
	return address, nil
}

var (
	errInvalidPercentile = fmt.Errorf("invalid reward percentile")
	errRequestBeyondHead = fmt.Errorf("request beyond head block")
)

// FeeHistory returns data relevant for fee estimation based on the specified range of blocks.
func (b *Backend) FeeHistory(
	userBlockCount math.HexOrDecimal64, // number blocks to fetch, maximum is 100
	lastBlock rpc.BlockNumber, // the block to start search , to oldest
	rewardPercentiles []float64, // percentiles to fetch reward
) (*rpctypes.FeeHistoryResult, error) {
	for i, p := range rewardPercentiles {
		if p < 0 || p > 100 {
			return nil, fmt.Errorf("%w: %f", errInvalidPercentile, p)
		}
		if i > 0 && p < rewardPercentiles[i-1] {
			return nil, fmt.Errorf("%w: #%d:%f > #%d:%f", errInvalidPercentile, i-1, rewardPercentiles[i-1], i, p)
		}
	}
	blockNumber, err := b.BlockNumber()
	if err != nil {
		return nil, err
	}
	blockEnd := int64(lastBlock)
	if blockEnd < 0 {
		blockEnd = int64(blockNumber)
	} else if int64(blockNumber) < blockEnd {
		return nil, fmt.Errorf("%w: requested %d, head %d", errRequestBeyondHead, blockEnd, int64(blockNumber))
	}

	blocks := int64(userBlockCount)
	maxBlockCount := int64(b.cfg.JSONRPC.FeeHistoryCap)
	if blocks > maxBlockCount {
		return nil, fmt.Errorf("FeeHistory user block count %d higher than %d", blocks, maxBlockCount)
	}
	if blockEnd < gomath.MaxInt64 && blockEnd+1 < blocks {
		blocks = blockEnd + 1
	}
	// Ensure not trying to retrieve before genesis.
	blockStart := blockEnd + 1 - blocks
	oldestBlock := (*hexutil.Big)(big.NewInt(blockStart))

	// prepare space
	reward := make([][]*hexutil.Big, blocks)
	rewardCount := len(rewardPercentiles)
	for i := 0; i < int(blocks); i++ {
		reward[i] = make([]*hexutil.Big, rewardCount)
	}

	thisBaseFee := make([]*hexutil.Big, blocks+1)
	thisGasUsedRatio := make([]float64, blocks)

	// rewards should only be calculated if reward percentiles were included
	calculateRewards := rewardCount != 0
	const maxBlockFetchers = 4
	for blockID := blockStart; blockID <= blockEnd; blockID += maxBlockFetchers {
		wg := sync.WaitGroup{}
		wgDone := make(chan bool)
		chanErr := make(chan error)
		for i := 0; i < maxBlockFetchers; i++ {
			if blockID+int64(i) >= blockEnd+1 {
				break
			}
			wg.Add(1)
			go func(index int32) {
				defer func() {
					if r := recover(); r != nil {
						err = errorsmod.Wrapf(errortypes.ErrPanic, "%v", r)
						b.logger.Error("FeeHistory panicked", "error", err)
						chanErr <- err
					}
					wg.Done()
				}()
				// fetch block
				// tendermint block
				blockNum := rpctypes.BlockNumber(blockStart + int64(index))
				tendermintblock, err := b.TendermintBlockByNumber(blockNum)
				if tendermintblock == nil {
					chanErr <- err
					return
				}

				// eth block
				ethBlock, err := b.GetBlockByNumber(blockNum, true)
				if ethBlock == nil {
					chanErr <- err
					return
				}

				// tendermint block result
				tendermintBlockResult, err := b.TendermintBlockResultByNumber(&tendermintblock.Block.Height)
				if tendermintBlockResult == nil {
					b.logger.Debug("block result not found", "height", tendermintblock.Block.Height, "error", err.Error())
					chanErr <- err
					return
				}

				oneFeeHistory := rpctypes.OneFeeHistory{}
				err = b.processBlocker(tendermintblock, ethBlock, rewardPercentiles, tendermintBlockResult, &oneFeeHistory)
				if err != nil {
					chanErr <- err
					return
				}

				// copy
				thisBaseFee[index] = (*hexutil.Big)(oneFeeHistory.BaseFee)
				// only use NextBaseFee as last item to avoid concurrent write
				if int(index) == len(thisBaseFee)-2 {
					thisBaseFee[index+1] = (*hexutil.Big)(oneFeeHistory.NextBaseFee)
				}
				thisGasUsedRatio[index] = oneFeeHistory.GasUsedRatio
				if calculateRewards {
					for j := 0; j < rewardCount; j++ {
						reward[index][j] = (*hexutil.Big)(oneFeeHistory.Reward[j])
						if reward[index][j] == nil {
							reward[index][j] = (*hexutil.Big)(big.NewInt(0))
						}
					}
				}
			}(int32(blockID - blockStart + int64(i)))
		}
		go func() {
			wg.Wait()
			close(wgDone)
		}()
		select {
		case <-wgDone:
		case err := <-chanErr:
			return nil, err
		}
	}

	feeHistory := rpctypes.FeeHistoryResult{
		OldestBlock:  oldestBlock,
		BaseFee:      thisBaseFee,
		GasUsedRatio: thisGasUsedRatio,
	}

	if calculateRewards {
		feeHistory.Reward = reward
	}

	return &feeHistory, nil
}

/*
SuggestedGasTipCap, GlobalMinGasPrice, BaseFee are stubbed out for now, because
we don't use the FeeMarket module.
*/

// SuggestGasTipCap returns the suggested tip cap
// Although we don't support tx prioritization yet, but we return a positive value to help client to
// mitigate the base fee changes.
func (b *Backend) SuggestGasTipCap(baseFee *big.Int) (*big.Int, error) {
	return big.NewInt(0), nil
}

// GlobalMinGasPrice returns MinGasPrice param from FeeMarket
func (b *Backend) GlobalMinGasPrice() (sdkmath.LegacyDec, error) {
	return sdkmath.LegacyZeroDec(), nil
}

// BaseFee returns the base fee tracked by the Fee Market module.
// If the base fee is not enabled globally, the query returns nil.
// If the London hard fork is not activated at the current height, the query will
// return nil.
func (b *Backend) BaseFee(blockRes *cmtrpctypes.ResultBlockResults) (*big.Int, error) {
	return b.RPCMinGasPrice(), nil
}
