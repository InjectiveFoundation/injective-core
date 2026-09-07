package backend

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	cmrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	rpctypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/rpc/types"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// GetTxHashByEthHash returns BFT tx hash by eth tx hash
func (b *Backend) GetTxHashByEthHash(ethHash common.Hash) (common.Hash, error) {
	res, err := b.GetTxByEthHash(ethHash)
	if err != nil {
		return common.Hash{}, err
	}

	block, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(res.Height))
	if err != nil || block == nil {
		return common.Hash{}, fmt.Errorf("block not found, err: %w", err)
	}

	if int(res.TxIndex) >= len(block.Block.Txs) {
		return common.Hash{}, fmt.Errorf("tx index out of bounds")
	}

	bftHash := block.Block.Txs[res.TxIndex].Hash()

	return common.Hash(bftHash), nil
}

// GetTransactionByHash returns the Ethereum format transaction identified by Ethereum transaction hash
func (b *Backend) GetTransactionByHash(txHash common.Hash) (*rpctypes.RPCTransaction, error) {
	res, err := b.GetTxByEthHash(txHash)
	if err != nil {
		return b.getTransactionByHashPending(txHash)
	}

	block, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(res.Height))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, nil
	}

	if int(res.TxIndex) >= len(block.Block.Txs) {
		return nil, errors.New("tx index out of range")
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
	if err != nil {
		return nil, err
	}

	msgs := tx.GetMsgs()
	if int(res.MsgIndex) >= len(msgs) {
		return nil, errors.New("msg index out of range")
	}

	// the `res.MsgIndex` is inferred from tx index, should be within the bound.
	msg, ok := msgs[res.MsgIndex].(*evmtypes.MsgEthereumTx)
	if !ok {
		return nil, errors.New("invalid ethereum tx")
	}

	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		b.logger.Debug("block result not found", "height", block.Block.Height, "error", err.Error())
		return nil, nil
	}

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		ethMsgs := b.EthMsgsFromTendermintBlock(block)
		for i := range ethMsgs {
			if ethMsgs[i].Hash() == txHash {
				res.EthTxIndex = int32(i)
				break
			}
		}
	}
	// if we still unable to find the eth tx index, return error, shouldn't happen.
	if res.EthTxIndex == -1 {
		return nil, errors.New("can't find index of ethereum tx")
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		// handle the error for pruned node.
		b.logger.Error("failed to fetch Base Fee from prunned block. Check node prunning configuration", "height", blockRes.Height, "error", err)
	}

	return rpctypes.NewTransactionFromMsg(
		msg,
		common.BytesToHash(block.BlockID.Hash.Bytes()),
		uint64(res.Height),
		uint64(res.EthTxIndex),
		baseFee,
		b.ChainID().ToInt(),
	)
}

// getTransactionByHashPending find pending tx from mempool
func (b *Backend) getTransactionByHashPending(txHash common.Hash) (*rpctypes.RPCTransaction, error) {
	// try to find tx in mempool
	txs, err := b.PendingTransactions()
	if err != nil {
		b.logger.Debug("tx not found", "hash", txHash, "error", err.Error())
		return nil, nil
	}

	for _, tx := range txs {
		msg, err := evmtypes.UnwrapEthereumMsg(tx, txHash)
		if err != nil {
			// not ethereum tx
			continue
		}

		if msg.Hash() == txHash {
			// use zero block values since it's not included in a block yet
			rpctx, err := rpctypes.NewTransactionFromMsg(
				msg,
				common.Hash{},
				uint64(0),
				uint64(0),
				nil,
				b.ChainID().ToInt(),
			)
			if err != nil {
				return nil, err
			}
			return rpctx, nil
		}
	}

	b.logger.Debug("tx not found", "hash", txHash)
	return nil, nil
}

// GetGasUsed returns gasUsed from transaction
func (b *Backend) GetGasUsed(res *chaintypes.TxResult, gas uint64) uint64 {
	return res.GasUsed
}

// GetTransactionReceipt returns the transaction receipt identified by hash.
func (b *Backend) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	b.logger.Debug("eth_getTransactionReceipt", "hash", hash)

	res, err := b.GetTxByEthHash(hash)
	if err != nil {
		b.logger.Debug("tx not found", "hash", hash, "error", err.Error())
		return nil, nil
	}
	resBlock, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(res.Height))
	if err != nil {
		b.logger.Debug("block not found", "height", res.Height, "error", err.Error())
		return nil, err
	}
	if resBlock == nil || resBlock.Block == nil {
		b.logger.Debug("block not found", "height", res.Height)
		return nil, nil
	}

	if int(res.TxIndex) >= len(resBlock.Block.Txs) {
		b.logger.Warn("txIndex out of range", "txIndex", res.TxIndex, "len", len(resBlock.Block.Txs))
		return nil, nil
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(resBlock.Block.Txs[res.TxIndex])
	if err != nil {
		b.logger.Warn("decoding failed", "error", err.Error())
		return nil, fmt.Errorf("failed to decode tx: %w", err)
	}

	msgs := tx.GetMsgs()
	if int(res.MsgIndex) >= len(msgs) {
		b.logger.Warn("msgIndex out of range", "msgIndex", res.MsgIndex, "len", len(msgs))
		return nil, nil
	}

	ethMsg, ok := msgs[res.MsgIndex].(*evmtypes.MsgEthereumTx)
	if !ok {
		b.logger.Error("invalid ethereum tx type")
		return nil, errors.New("invalid ethereum tx")
	}

	txData := ethMsg.AsTransaction()
	if txData == nil {
		b.logger.Error("failed to unpack tx data")
		return nil, errors.New("failed to unpack tx data")
	}

	blockRes, err := b.TendermintBlockResultByNumber(&res.Height)
	if err != nil {
		b.logger.Warn("failed to retrieve block results", "height", res.Height, "error", err.Error())
		return nil, nil
	}

	cumulativeGasUsed := uint64(0)
	maxTxIndex := int(res.TxIndex)
	if maxTxIndex > len(blockRes.TxResults) {
		maxTxIndex = len(blockRes.TxResults)
	}
	for _, txResult := range blockRes.TxResults[0:maxTxIndex] {
		cumulativeGasUsed += uint64(txResult.GasUsed)
	}
	cumulativeGasUsed += res.CumulativeGasUsed

	var status hexutil.Uint
	if res.Failed {
		status = hexutil.Uint(ethtypes.ReceiptStatusFailed)
	} else {
		status = hexutil.Uint(ethtypes.ReceiptStatusSuccessful)
	}

	from, err := ethMsg.GetSenderLegacy(ethtypes.LatestSignerForChainID(b.ChainID().ToInt()))
	if err != nil {
		return nil, err
	}

	blockHash := common.BytesToHash(resBlock.Block.Header.Hash())

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		ethMsgs := b.EthMsgsFromTendermintBlock(resBlock)
		for i := range ethMsgs {
			if ethMsgs[i].Hash() == hash {
				res.EthTxIndex = int32(i)
				break
			}
		}
	}
	// return error if still unable to find the eth tx index
	if res.EthTxIndex == -1 {
		return nil, errors.New("can't find index of ethereum tx")
	}

	// parse tx logs from events
	var logs []*ethtypes.Log
	if int(res.TxIndex) < len(blockRes.TxResults) {
		logs, err = evmtypes.DecodeMsgLogs(
			blockRes.TxResults[res.TxIndex].Data,
			int(res.MsgIndex),
			uint64(blockRes.Height),
		)
		if err != nil {
			b.logger.Warn("failed to parse logs", "hash", hash, "error", err.Error())
			logs = nil
		}
	}

	// Enrich log fields (blockHash, transactionHash, transactionIndex, etc.)
	for i, log := range logs {
		log.BlockHash = blockHash
		log.TxHash = hash
		log.TxIndex = uint(res.EthTxIndex)
		log.Index = uint(i)
	}

	var baseFee *big.Int
	if txData.Type() == ethtypes.DynamicFeeTxType {
		baseFee, err = b.BaseFee(blockRes)
		if err != nil {
			baseFee = nil
		}
	}

	var contractAddress interface{} = nil
	if txData.To() == nil {
		contractAddress = crypto.CreateAddress(from, txData.Nonce())
	}

	formattedLogs := logs
	if formattedLogs == nil {
		formattedLogs = []*ethtypes.Log{}
	}

	receipt := map[string]interface{}{
		// Consensus fields: These fields are defined by the Yellow Paper
		"status":            status,
		"cumulativeGasUsed": hexutil.Uint64(cumulativeGasUsed),
		"logsBloom":         ethtypes.BytesToBloom(evmtypes.LogsBloom(logs)),
		"logs":              formattedLogs,

		// Implementation fields: These fields are added by geth when processing a transaction.
		"transactionHash": hash,
		"contractAddress": contractAddress,
		"gasUsed":         hexutil.Uint64(b.GetGasUsed(res, txData.Gas())),

		// Inclusion information: These fields provide information about the inclusion
		"blockHash":        blockHash.Hex(),
		"blockNumber":      hexutil.Uint64(res.Height),
		"transactionIndex": hexutil.Uint64(res.EthTxIndex),

		// https://github.com/foundry-rs/foundry/issues/7640
		"effectiveGasPrice": (*hexutil.Big)(effectiveGasPrice(txData, baseFee)),

		// sender and receiver (contract or EOA) addresses
		"from": from,
		"to":   txData.To(),
		"type": hexutil.Uint(txData.Type()),
	}

	return receipt, nil
}

// GetTransactionByBlockHashAndIndex returns the transaction identified by hash and index.
func (b *Backend) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	b.logger.Debug("eth_getTransactionByBlockHashAndIndex", "hash", hash.Hex(), "index", idx)

	sc, ok := b.clientCtx.Client.(cmrpcclient.SignClient)
	if !ok {
		return nil, errors.New("invalid rpc client")
	}

	block, err := sc.BlockByHash(b.ctx, hash.Bytes())
	if err != nil {
		b.logger.Debug("block not found", "hash", hash.Hex(), "error", err.Error())
		return nil, nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "hash", hash.Hex())
		return nil, nil
	}

	return b.GetTransactionByBlockAndIndex(block, idx)
}

// GetTransactionByBlockNumberAndIndex returns the transaction identified by number and index.
func (b *Backend) GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	b.logger.Debug("eth_getTransactionByBlockNumberAndIndex", "number", blockNum, "index", idx)

	block, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		b.logger.Debug("block not found", "height", blockNum.Int64(), "error", err.Error())
		return nil, err
	}

	if block == nil || block.Block == nil {
		b.logger.Debug("block not found", "height", blockNum.Int64())
		return nil, nil
	}

	return b.GetTransactionByBlockAndIndex(block, idx)
}

// GetTxByEthHash uses `/tx_query` to find transaction by ethereum tx hash
func (b *Backend) GetTxByEthHash(hash common.Hash) (*chaintypes.TxResult, error) {
	if b.indexer != nil {
		res, err := b.indexer.GetByTxHash(hash)
		if err == nil && res != nil {
			return res, nil
		}
	}

	// fallback to tendermint tx indexer
	b.logger.Warn("fallback to tendermint tx indexer! failed txns will not be available", "tx", hash.Hex())
	query := fmt.Sprintf("%s.%s='%s'", evmtypes.TypeMsgEthereumTx, evmtypes.AttributeKeyEthereumTxHash, hash.Hex())
	txResult, err := b.queryTendermintTxIndexer(query, func(txs *rpctypes.ParsedTxs) *rpctypes.ParsedTx {
		return txs.GetTxByHash(hash)
	})
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetTxByEthHash %s", hash.Hex())
	}
	return txResult, nil
}

// GetTxByTxIndex uses `/tx_query` to find transaction by tx index of valid ethereum txs
func (b *Backend) GetTxByTxIndex(height int64, index uint) (*chaintypes.TxResult, error) {
	if b.indexer != nil {
		res, err := b.indexer.GetByBlockAndIndex(height, int32(index))
		if err == nil && res != nil {
			return res, nil
		}
	}

	// fallback to tendermint tx indexer
	b.logger.Warn("fallback to tendermint tx indexer! failed txns will not be available", "height", height, "txIndex", index)
	query := fmt.Sprintf("tx.height=%d AND %s.%s=%d",
		height, evmtypes.TypeMsgEthereumTx,
		evmtypes.AttributeKeyTxIndex, index,
	)
	txResult, err := b.queryTendermintTxIndexer(query, func(txs *rpctypes.ParsedTxs) *rpctypes.ParsedTx {
		return txs.GetTxByTxIndex(int(index))
	})
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetTxByTxIndex %d %d", height, index)
	}
	return txResult, nil
}

// queryTendermintTxIndexer query tx in tendermint tx indexer
func (b *Backend) queryTendermintTxIndexer(query string, txGetter func(*rpctypes.ParsedTxs) *rpctypes.ParsedTx) (*chaintypes.TxResult, error) {
	resTxs, err := b.clientCtx.Client.TxSearch(b.ctx, query, false, nil, nil, "")
	if err != nil {
		return nil, err
	}

	if len(resTxs.Txs) == 0 {
		return nil, errors.New("ethereum tx not found")
	}

	txResult := resTxs.Txs[0]

	var tx sdk.Tx
	if rpctypes.TxSuccessOrExceedsBlockGasLimit(&txResult.TxResult) &&
		txResult.TxResult.Code != abci.CodeTypeOK &&
		txResult.TxResult.Codespace != evmtypes.ModuleName {

		// only needed when the tx exceeds block gas limit
		tx, err = b.clientCtx.TxConfig.TxDecoder()(txResult.Tx)
		if err != nil {
			return nil, fmt.Errorf("invalid ethereum tx")
		}
	}

	return rpctypes.ParseTxIndexerResult(txResult, tx, txGetter)
}

// GetTransactionByBlockAndIndex is the common code shared by `GetTransactionByBlockNumberAndIndex` and `GetTransactionByBlockHashAndIndex`.
func (b *Backend) GetTransactionByBlockAndIndex(block *cmrpctypes.ResultBlock, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil, nil
	}

	var msg *evmtypes.MsgEthereumTx
	// find in tx indexer
	res, err := b.GetTxByTxIndex(block.Block.Height, uint(idx))
	if err == nil && res != nil {
		if int(res.TxIndex) < len(block.Block.Txs) {
			tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
			if err == nil {
				msgs := tx.GetMsgs()
				if int(res.MsgIndex) < len(msgs) {
					var ok bool
					msg, ok = msgs[res.MsgIndex].(*evmtypes.MsgEthereumTx)
					if !ok {
						msg = nil
					}
				}
			}
		}
	}

	// Fallback when indexer fail or returned invalid msg
	if msg == nil {
		i := int(idx)
		if i < 0 {
			i = 0
		}
		ethMsgs := b.EthMsgsFromTendermintBlock(block)
		if i >= len(ethMsgs) {
			b.logger.Warn("block txs index out of bound", "index", i)
			return nil, nil
		}

		msg = ethMsgs[i]
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		// handle the error for pruned node.
		b.logger.Error("failed to fetch Base Fee from prunned block. Check node prunning configuration", "height", block.Block.Height, "error", err)
	}

	return rpctypes.NewTransactionFromMsg(
		msg,
		common.BytesToHash(block.Block.Hash()),
		uint64(block.Block.Height),
		uint64(idx),
		baseFee,
		b.ChainID().ToInt(),
	)
}