package helpers

import (
	"context"
	"strings"
	"time"

	retry "github.com/avast/retry-go/v4"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pkg/errors"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetBlock(ctx context.Context, chainNode *cosmos.ChainNode, height int64) (*ctypes.ResultBlock, error) {
	block, err := chainNode.Client.Block(ctx, &height)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func GetBlockResults(ctx context.Context, chainNode *cosmos.ChainNode, height int64) (*ctypes.ResultBlockResults, error) {
	block, err := chainNode.Client.BlockResults(ctx, &height)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// getTxResponseRPC queries a transaction using gRPC.
// This function retries the query to handle cases where the transaction is not yet committed to state.
func getTxResponseRPC(ctx context.Context, chainNode *cosmos.ChainNode, txHash string) (*sdk.TxResponse, error) {
	// Normalize hash - remove 0x prefix if present
	hash := strings.TrimPrefix(txHash, "0x")

	chain := chainNode.Chain.(*cosmos.CosmosChain)
	conn, err := grpc.NewClient(chain.GetHostGRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create gRPC connection")
	}
	defer conn.Close()

	txClient := txtypes.NewServiceClient(conn)

	var txResp *txtypes.GetTxResponse

	// Retry because sometimes the tx is not committed to state yet
	err = retry.Do(
		func() error {
			var err error
			txResp, err = QueryRPC(ctx, txClient.GetTx, &txtypes.GetTxRequest{Hash: hash})
			return err
		},
		// retry for total of 3 seconds
		retry.Attempts(15),
		retry.Delay(200*time.Millisecond),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query transaction %s via gRPC", txHash)
	}

	return txResp.TxResponse, nil
}

// QueryTxRPC queries a transaction using gRPC, avoiding the need to start a docker container.
// This function retries the query to handle cases where the transaction is not yet committed to state.
func QueryTxRPC(ctx context.Context, chainNode *cosmos.ChainNode, txHash string) (transaction Tx, err error) {
	txResp, err := getTxResponseRPC(ctx, chainNode, txHash)
	if err != nil {
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

// QueryCall is a type alias for gRPC query functions, similar to APICall in sdk-go.
type QueryCall[Q any, R any] func(ctx context.Context, in *Q, opts ...grpc.CallOption) (*R, error)

// QueryRPC is a generic helper function to execute gRPC queries.
// It simply executes the provided query call with the given request.
// This pattern follows the sdk-go api_request_assistant ExecuteCall.
func QueryRPC[Q any, R any](
	ctx context.Context,
	call QueryCall[Q, R],
	request *Q,
) (*R, error) {
	return call(ctx, request)
}
