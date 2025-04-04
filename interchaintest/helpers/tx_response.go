package helpers

import (
	"context"
	"time"

	retry "github.com/avast/retry-go/v4"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/cosmos/cosmos-sdk/client/tx"
	clientctx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/pkg/errors"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

// Tx contains some of Cosmos transaction details.
type Tx struct {
	Height uint64
	TxHash string

	GasWanted uint64
	GasUsed   uint64

	ErrorCode uint32
}

// BroadcastTxAsync broadcasts a transaction and returns immediately, without
// waiting for the transaction to be committed to state.
func BroadcastTxAsync(
	broadcaster *cosmos.Broadcaster,
	txFactory tx.Factory,
	user cosmos.User,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	clientCtx, err := broadcaster.GetClientContext(context.TODO(), user)
	if err != nil {
		return nil, err
	}

	err = clientctx.BroadcastTx(clientCtx, txFactory, msgs...)
	if err != nil {
		return nil, err
	}

	txBytes, err := broadcaster.GetTxResponseBytes(context.TODO(), user)
	if err != nil {
		return nil, err
	}

	txResponse, err := broadcaster.UnmarshalTxResponseBytes(context.TODO(), txBytes)

	return &txResponse, err
}

// QueryTx reads results of a Tx, to check for errors during execution and receiving its raw log
func QueryTx(ctx context.Context, chainNode *cosmos.ChainNode, txHash string) (transaction Tx, err error) {
	txResp, err := getTxResponse(ctx, chainNode, txHash)
	if err != nil {
		err = errors.Wrapf(err, "failed to get transaction %s", txHash)
		return transaction, err
	}

	transaction.Height = uint64(txResp.Height)
	transaction.TxHash = txHash
	transaction.GasWanted = uint64(txResp.GasWanted)
	transaction.GasUsed = uint64(txResp.GasUsed)

	if txResp.Code != 0 {
		transaction.ErrorCode = txResp.Code
		err = errors.Errorf("%s %d: %s", txResp.Codespace, txResp.Code, txResp.RawLog)
		return transaction, err
	}

	return transaction, nil
}

func getTxResponse(ctx context.Context, chainNode *cosmos.ChainNode, txHash string) (*sdk.TxResponse, error) {
	// Retry because sometimes the tx is not committed to state yet.
	var txResp *sdk.TxResponse

	err := retry.Do(
		func() error {
			var err error
			txResp, err = authtx.QueryTx(chainNode.CliContext(), txHash)
			return err
		},
		// retry for total of 3 seconds
		retry.Attempts(15),
		retry.Delay(200*time.Millisecond),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
	return txResp, err
}

func GetBlock(ctx context.Context, chainNode *cosmos.ChainNode, height int64) (*ctypes.ResultBlock, error) {
	block, err := chainNode.Client.Block(ctx, &height)
	if err != nil {
		return nil, err
	}
	return block, nil
}
