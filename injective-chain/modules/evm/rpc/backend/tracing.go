package backend

import (
	"encoding/json"
	"fmt"

	rpctypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/rpc/types"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	cmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (b *Backend) TraceTransaction(hash common.Hash, config *rpctypes.TraceConfig) (interface{}, error) {
	// Get transaction by hash
	transaction, err := b.GetTxByEthHash(hash)
	if err != nil {
		b.logger.Debug("tx not found", "hash", hash)
		return nil, err
	}

	// check if block number is 0
	if transaction.Height == 0 {
		return nil, errors.New("genesis is not traceable")
	}

	block, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(transaction.Height))
	if err != nil || block == nil {
		b.logger.Debug("block not found", "height", transaction.Height)
		return nil, fmt.Errorf("block not found, err: %w", err)
	}

	// check tx index is not out of bound
	if uint32(len(block.Block.Txs)) < transaction.TxIndex {
		b.logger.Debug("tx index out of bounds", "index", transaction.TxIndex, "hash", hash.String(), "height", block.Block.Height)
		return nil, fmt.Errorf("transaction not included in block %v", block.Block.Height)
	}

	var predecessors []*evmtypes.MsgEthereumTx
	for _, txBz := range block.Block.Txs[:transaction.TxIndex] {
		tx, err := b.clientCtx.TxConfig.TxDecoder()(txBz)
		if err != nil {
			b.logger.Debug("failed to decode transaction in block", "height", block.Block.Height, "error", err.Error())
			continue
		}
		for _, msg := range tx.GetMsgs() {
			ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
			if !ok {
				continue
			}

			predecessors = append(predecessors, ethMsg)
		}
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[transaction.TxIndex])
	if err != nil {
		b.logger.Debug("tx not found", "hash", hash)
		return nil, err
	}

	// add predecessor messages in current cosmos tx
	for i := 0; i < int(transaction.MsgIndex); i++ {
		ethMsg, ok := tx.GetMsgs()[i].(*evmtypes.MsgEthereumTx)
		if !ok {
			continue
		}
		predecessors = append(predecessors, ethMsg)
	}

	ethMessage, ok := tx.GetMsgs()[transaction.MsgIndex].(*evmtypes.MsgEthereumTx)
	if !ok {
		b.logger.Debug("invalid transaction type", "type", fmt.Sprintf("%T", tx))
		return nil, fmt.Errorf("invalid transaction type %T", tx)
	}

	traceTxRequest := evmtypes.QueryTraceTxRequest{
		Msg:             ethMessage,
		Predecessors:    predecessors,
		BlockNumber:     block.Block.Height,
		BlockTime:       block.Block.Time,
		BlockHash:       common.Bytes2Hex(block.BlockID.Hash),
		ProposerAddress: sdk.ConsAddress(block.Block.ProposerAddress),
		ChainId:         b.ChainID().ToInt().Int64(),
	}

	if config != nil {
		traceTxRequest.TraceConfig = b.convertConfig(config)
	}

	// minus one to get the context of block beginning
	contextHeight := transaction.Height - 1
	if contextHeight < 1 {
		// 0 is a special value in `ContextWithHeight`
		contextHeight = 1
	}
	traceResult, err := b.queryClient.TraceTx(rpctypes.ContextWithHeight(contextHeight), &traceTxRequest)
	if err != nil {
		return nil, err
	}

	// Response format is unknown due to custom tracer config param
	// More information can be found here https://geth.ethereum.org/docs/dapp/tracing-filtered
	var decodedResult interface{}
	err = json.Unmarshal(traceResult.Data, &decodedResult)
	if err != nil {
		return nil, err
	}

	return decodedResult, nil
}

func (b *Backend) convertConfig(config *rpctypes.TraceConfig) *evmtypes.TraceConfig {
	if config == nil {
		return &evmtypes.TraceConfig{}
	}
	cfg := config.TraceConfig
	cfg.TracerJsonConfig = string(config.TracerConfig)
	cfg.StateOverrides = []byte(config.StateOverrides)
	cfg.BlockOverrides = []byte(config.BlockOverrides)
	return &cfg
}

// TraceBlock configures a new tracer according to the provided configuration, and
// executes all the transactions contained within. The return value will be one item
// per transaction, dependent on the requested tracer.
func (b *Backend) TraceBlock(height rpctypes.BlockNumber,
	config *rpctypes.TraceConfig,
	block *cmrpctypes.ResultBlock,
) ([]*evmtypes.TxTraceResult, error) {
	txs := block.Block.Txs
	txsLength := len(txs)

	if txsLength == 0 {
		// If there are no transactions return empty array
		return []*evmtypes.TxTraceResult{}, nil
	}

	txDecoder := b.clientCtx.TxConfig.TxDecoder()

	var txsMessages []*evmtypes.MsgEthereumTx
	for i, tx := range txs {
		decodedTx, err := txDecoder(tx)
		if err != nil {
			b.logger.Warn("failed to decode transaction", "hash", txs[i].Hash(), "error", err.Error())
			continue
		}

		for _, msg := range decodedTx.GetMsgs() {
			ethMessage, ok := msg.(*evmtypes.MsgEthereumTx)
			if !ok {
				// Just considers Ethereum transactions
				continue
			}
			txsMessages = append(txsMessages, ethMessage)
		}
	}

	// minus one to get the context at the beginning of the block
	contextHeight := height - 1
	if contextHeight < 1 {
		// 0 is a special value for `ContextWithHeight`.
		contextHeight = 1
	}
	ctxWithHeight := rpctypes.ContextWithHeight(int64(contextHeight))

	traceBlockRequest := &evmtypes.QueryTraceBlockRequest{
		Txs:             txsMessages,
		TraceConfig:     b.convertConfig(config),
		BlockNumber:     block.Block.Height,
		BlockTime:       block.Block.Time,
		BlockHash:       common.Bytes2Hex(block.BlockID.Hash),
		ProposerAddress: sdk.ConsAddress(block.Block.ProposerAddress),
		ChainId:         b.ChainID().ToInt().Int64(),
	}

	res, err := b.queryClient.TraceBlock(ctxWithHeight, traceBlockRequest)
	if err != nil {
		return nil, err
	}

	decodedResults := make([]*evmtypes.TxTraceResult, txsLength)
	if err := json.Unmarshal(res.Data, &decodedResults); err != nil {
		return nil, err
	}

	return decodedResults, nil
}

// TraceCall returns the structured logs created during the execution of EVM call
// and returns them as a JSON object.
func (b *Backend) TraceCall(
	args evmtypes.TransactionArgs, blockNrOrHash rpctypes.BlockNumberOrHash, config *rpctypes.TraceConfig,
) (interface{}, error) {
	bz, err := json.Marshal(&args)
	if err != nil {
		return nil, err
	}
	blockNr, err := b.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}
	block, err := b.TendermintBlockByNumber(blockNr)
	if err != nil || block == nil {
		// the error message imitates geth behavior
		return nil, errors.New("header not found")
	}

	traceCallRequest := evmtypes.QueryTraceCallRequest{
		Args:            bz,
		GasCap:          b.RPCGasCap(),
		ProposerAddress: sdk.ConsAddress(block.Block.ProposerAddress),
		BlockNumber:     block.Block.Height,
		BlockHash:       common.Bytes2Hex(block.BlockID.Hash),
		BlockTime:       block.Block.Time,
		ChainId:         b.ChainID().ToInt().Int64(),
	}

	if config != nil {
		traceCallRequest.TraceConfig = b.convertConfig(config)
	}

	// get the context of provided block
	contextHeight := block.Block.Height
	if contextHeight < 1 {
		// 0 is a special value in `ContextWithHeight`
		contextHeight = 1
	}
	traceResult, err := b.queryClient.TraceCall(rpctypes.ContextWithHeight(contextHeight), &traceCallRequest)
	if err != nil {
		return nil, err
	}

	// Response format is unknown due to custom tracer config param
	// More information can be found here https://geth.ethereum.org/docs/dapp/tracing-filtered
	var decodedResult interface{}
	err = json.Unmarshal(traceResult.Data, &decodedResult)
	if err != nil {
		return nil, err
	}

	return decodedResult, nil
}
