package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// ParsedTx is the tx infos parsed from events (success) or log field (failure).
type ParsedTx struct {
	MsgIndex int

	Hash       common.Hash
	EthTxIndex int32
	GasUsed    uint64
	Failed     bool
}

const EthTxIndexUnitialized int32 = -1

// NewParsedTx initialize a ParsedTx
func NewParsedTx(msgIndex int) ParsedTx {
	return ParsedTx{
		MsgIndex:   msgIndex,
		EthTxIndex: EthTxIndexUnitialized,
	}
}

// ParsedTxs is the tx infos parsed from eth tx events.
type ParsedTxs struct {
	// one item per message
	Txs []ParsedTx
	// map tx hash to msg index
	TxHashes map[common.Hash]int
}

// ParseTxResult parse eth tx infos from ABCI TxResult.
// Uses info from events (success) or log field (failure).
func ParseTxResult(result *abci.ExecTxResult, tx sdk.Tx) (*ParsedTxs, error) {
	p := &ParsedTxs{
		TxHashes: make(map[common.Hash]int),
	}

	for _, event := range result.Events {
		if event.Type != evmtypes.EventTypeEthereumTx {
			continue
		}

		if err := p.parseTxFromEvent(event.Attributes); err != nil {
			return nil, err
		}
	}

	if result.Code != abci.CodeTypeOK && result.Codespace == evmtypes.ModuleName {
		for i := 0; i < len(p.Txs); i++ {
			// fail all evm txns in the tx result
			p.Txs[i].Failed = true
		}

		if err := p.parseFromLog(result.Log); err != nil {
			return nil, err
		}

		return p, nil
	}

	// if namespace is not evm, assume block gas limit is reached
	//
	// TODO: proper code matching
	if result.Code != abci.CodeTypeOK && result.Codespace != evmtypes.ModuleName && tx != nil {
		for i := 0; i < len(p.Txs); i++ {
			p.Txs[i].Failed = true
			// replace gasUsed with gasLimit because that's what's actually deducted.
			//
			// TODO: check  if this is still correct
			gasLimit := tx.GetMsgs()[i].(*evmtypes.MsgEthereumTx).GetGas()
			p.Txs[i].GasUsed = gasLimit
		}
	}

	return p, nil
}

// ParseTxIndexerResult parse tm tx result to a format compatible with the custom tx indexer.
func ParseTxIndexerResult(txResult *cmrpctypes.ResultTx, tx sdk.Tx, getter func(*ParsedTxs) *ParsedTx) (*chaintypes.TxResult, error) {
	txs, err := ParseTxResult(&txResult.TxResult, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tx events: block %d, index %d, %w", txResult.Height, txResult.Index, err)
	}

	parsedTx := getter(txs)
	if parsedTx == nil {
		return nil, fmt.Errorf("ethereum tx not found in msgs: block %d, index %d", txResult.Height, txResult.Index)
	}

	return &chaintypes.TxResult{
		Height:            txResult.Height,
		TxIndex:           txResult.Index,
		MsgIndex:          uint32(parsedTx.MsgIndex),
		EthTxIndex:        parsedTx.EthTxIndex,
		Failed:            parsedTx.Failed,
		GasUsed:           parsedTx.GasUsed,
		CumulativeGasUsed: txs.AccumulativeGasUsed(parsedTx.MsgIndex),
	}, nil
}

// parseTxFromEvent parses a new tx from event attrs.
func (p *ParsedTxs) parseTxFromEvent(attrs []abci.EventAttribute) error {
	msgIndex := len(p.Txs)
	tx := NewParsedTx(msgIndex)

	if err := fillTxAttributes(&tx, attrs); err != nil {
		return err
	}

	p.Txs = append(p.Txs, tx)
	p.TxHashes[tx.Hash] = msgIndex
	return nil
}

type abciLogVMError struct {
	Hash    string `json:"tx_hash"`
	GasUsed uint64 `json:"gas_used"`
	Reason  string `json:"reason,omitempty"`
	VMError string `json:"vm_error"`
	Ret     []byte `json:"ret,omitempty"`
}

// ugly hacks ahead... thanks to errors.Wrapf in BaseApp
var msgErrJSONRx = regexp.MustCompile(`failed to execute message; message index: (\d+): (.*)`)

// newTx parse a new tx from events, called during parsing.
func (p *ParsedTxs) parseFromLog(logText string) error {
	var vmErr abciLogVMError

	parts := msgErrJSONRx.FindStringSubmatch(logText)
	if len(parts) != 3 {
		return errors.New("failed to locate message error in abci log")
	}

	// parts[0] is the whole match
	msgIndexStr := parts[1]
	msgIndex, err := strconv.Atoi(msgIndexStr)
	if err != nil {
		return errorsmod.Wrap(err, "failed to parse message index as int")
	}

	logJSON := parts[2]
	if err := json.Unmarshal([]byte(logJSON), &vmErr); err != nil {
		err = errorsmod.Wrap(err, "failed to parse abci log as JSON")
		return err
	}

	txHash := common.HexToHash(vmErr.Hash)
	parsedTx := ParsedTx{
		MsgIndex:   msgIndex,
		Hash:       txHash,
		EthTxIndex: EthTxIndexUnitialized,
		GasUsed:    vmErr.GasUsed,
		Failed:     true,
	}

	p.Txs = append(p.Txs, parsedTx)
	p.TxHashes[txHash] = msgIndex
	return nil
}

// GetTxByHash find ParsedTx by tx hash, returns nil if not exists.
func (p *ParsedTxs) GetTxByHash(hash common.Hash) *ParsedTx {
	if idx, ok := p.TxHashes[hash]; ok {
		return &p.Txs[idx]
	}
	return nil
}

// GetTxByMsgIndex returns ParsedTx by msg index
func (p *ParsedTxs) GetTxByMsgIndex(i int) *ParsedTx {
	if i < 0 || i >= len(p.Txs) {
		return nil
	}
	return &p.Txs[i]
}

// GetTxByTxIndex returns ParsedTx by tx index
func (p *ParsedTxs) GetTxByTxIndex(txIndex int) *ParsedTx {
	if len(p.Txs) == 0 {
		return nil
	}
	// assuming the `EthTxIndex` increase continuously,
	// convert TxIndex to MsgIndex by subtract the begin TxIndex.
	msgIndex := txIndex - int(p.Txs[0].EthTxIndex)
	// GetTxByMsgIndex will check the bound
	return p.GetTxByMsgIndex(msgIndex)
}

// AccumulativeGasUsed calculates the accumulated gas used within the batch of txs
func (p *ParsedTxs) AccumulativeGasUsed(msgIndex int) (result uint64) {
	for i := 0; i <= msgIndex; i++ {
		result += p.Txs[i].GasUsed
	}
	return result
}

// fillTxAttribute parse attributes by name, less efficient than hardcode the index, but more stable against event
// format changes.
func fillTxAttribute(tx *ParsedTx, key, value []byte) error {
	switch string(key) {
	case evmtypes.AttributeKeyEthereumTxHash:
		tx.Hash = common.HexToHash(string(value))
	case evmtypes.AttributeKeyTxIndex:
		txIndex, err := strconv.ParseUint(string(value), 10, 31)
		if err != nil {
			return err
		}
		tx.EthTxIndex = int32(txIndex)
	case evmtypes.AttributeKeyTxGasUsed:
		gasUsed, err := strconv.ParseUint(string(value), 10, 64)
		if err != nil {
			return err
		}
		tx.GasUsed = gasUsed
	case evmtypes.AttributeKeyEthereumTxFailed:
		tx.Failed = len(value) > 0
	}
	return nil
}

func fillTxAttributes(tx *ParsedTx, attrs []abci.EventAttribute) error {
	for _, attr := range attrs {
		if err := fillTxAttribute(tx, []byte(attr.Key), []byte(attr.Value)); err != nil {
			return err
		}
	}
	return nil
}
