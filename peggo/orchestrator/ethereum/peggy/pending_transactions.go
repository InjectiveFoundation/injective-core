package peggy

import (
	"bytes"
	"context"
	"time"

	log "github.com/xlab/suplog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type PendingTxInput struct {
	InputData    hexutil.Bytes
	ReceivedTime time.Time
}

type PendingTxInputList []PendingTxInput

func (p *PendingTxInputList) AddPendingTxInput(pendingTx *RPCTransaction) {

	if !IsBatchOrValsetUpdateTx(pendingTx.Input) {
		return
	}

	pendingTxInput := PendingTxInput{
		InputData:    pendingTx.Input,
		ReceivedTime: time.Now(),
	}

	// Enqueue pending tx input
	*p = append(*p, pendingTxInput)
	// Persisting top 100 pending txs of peggy contract only.
	if len(*p) > 100 {
		(*p)[0] = PendingTxInput{} // to avoid memory leak
		// Dequeue pending tx input
		*p = (*p)[1:]
	}
}

func IsBatchOrValsetUpdateTx(inputData hexutil.Bytes) bool {

	submitBatchMethod := peggyABI.Methods["submitBatch"]
	valsetUpdateMethod := peggyABI.Methods["updateValset"]

	if bytes.Equal(submitBatchMethod.ID, inputData[:4]) || bytes.Equal(valsetUpdateMethod.ID, inputData[:4]) {
		return true
	} else {
		return false
	}
}

func (p PendingTxInputList) IsPendingTxInput(txInput []byte, pendingTxWaitDuration time.Duration) bool {
	for _, pendingTxInput := range p {
		if bytes.Equal(pendingTxInput.InputData, txInput) {

			if time.Now().Before(pendingTxInput.ReceivedTime.Add(pendingTxWaitDuration)) {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func (s *peggyContract) SubscribeToPendingTxs(alchemyWebsocketURL string) {
	args := map[string]interface{}{
		"toAddress": s.peggyAddress.Hex(),
	}

	wsClient, err := rpc.Dial(alchemyWebsocketURL)
	if err != nil {
		log.WithField("ws", alchemyWebsocketURL).WithError(err).Warningln("failed to connect to Ethereum Alchemy websocket")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to Transactions
	ch := make(chan *RPCTransaction)
	if _, err = wsClient.EthSubscribe(ctx, ch, "alchemy_pendingTransactions", args); err != nil {
		log.WithField("ws", alchemyWebsocketURL).WithError(err).Warningln("failed to subscribe to pending transactions")
		return
	}

	for {
		// Check that the transaction was send over the channel
		pendingTransaction := <-ch
		s.pendingTxInputList.AddPendingTxInput(pendingTransaction)
	}
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	Type             hexutil.Uint64  `json:"type"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}
