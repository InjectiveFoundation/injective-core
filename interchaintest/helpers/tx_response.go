package helpers

import (
	"context"
	"time"

	retry "github.com/avast/retry-go/v4"
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

// QueryTx reads results of a Tx, to check for errors during execution and receiving its raw log
func QueryTx(ctx context.Context, chainNode *cosmos.ChainNode, txHash string) (tx Tx, err error) {
	txResp, err := getTxResponse(ctx, chainNode, txHash)
	if err != nil {
		err = errors.Wrapf(err, "failed to get transaction %s", txHash)
		return tx, err
	}

	tx.Height = uint64(txResp.Height)
	tx.TxHash = txHash
	tx.GasWanted = uint64(txResp.GasWanted)
	tx.GasUsed = uint64(txResp.GasUsed)

	if txResp.Code != 0 {
		tx.ErrorCode = txResp.Code
		err = errors.Errorf("%s %d: %s", txResp.Codespace, txResp.Code, txResp.RawLog)
		return tx, err
	}

	return tx, nil
}

func getTxResponse(ctx context.Context, chainNode *cosmos.ChainNode, txHash string) (*sdk.TxResponse, error) {
	// Retry because sometimes the tx is not committed to state yet.
	var txResp *sdk.TxResponse

	err := retry.Do(func() error {
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
