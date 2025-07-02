package helpers

import (
	"context"
	"encoding/json"
	"time"

	retry "github.com/avast/retry-go/v4"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/pkg/errors"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

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

// QueryAccount queries the auth module for account information
func QueryAccount(ctx context.Context, chainNode *cosmos.ChainNode, address string) (*AccountResponse, error) {
	stdout, stderr, err := chainNode.ExecQuery(ctx, "auth", "account", address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query account %s: %s", address, string(stderr))
	}

	var accountResp AccountResponse
	if err := json.Unmarshal(stdout, &accountResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal account response")
	}

	return &accountResp, nil
}

// AccountResponse represents the response structure from auth account query
type AccountResponse struct {
	Account Account `json:"account"`
}

// Account represents the account information with type and value
type Account struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

// EthAccount represents the Injective EthAccount structure
type EthAccount struct {
	BaseAccount BaseAccount `json:"base_account"`
	CodeHash    string      `json:"code_hash"`
}

// BaseAccount represents the base account information
type BaseAccount struct {
	Address       string `json:"address"`
	AccountNumber uint64 `json:"account_number,string"`
	Sequence      uint64 `json:"sequence,string"`
}

// GetBaseAccount unmarshals the account value into a BaseAccount
func (a *Account) GetBaseAccount() (*BaseAccount, error) {
	var baseAccount BaseAccount

	if a.Type != "/cosmos.auth.v1beta1.BaseAccount" {
		return nil, errors.Errorf("account type is not /cosmos.auth.v1beta1.BaseAccount: %s", a.Type)
	}

	if err := json.Unmarshal(a.Value, &baseAccount); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal BaseAccount")
	}

	return &baseAccount, nil
}

// GetEthAccount unmarshals the account value into an EthAccount
func (a *Account) GetEthAccount() (*EthAccount, error) {
	var ethAccount EthAccount

	if a.Type != "/injective.types.v1beta1.EthAccount" {
		return nil, errors.Errorf("account type is not /injective.types.v1beta1.EthAccount: %s", a.Type)
	}

	if err := json.Unmarshal(a.Value, &ethAccount); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal EthAccount")
	}

	return &ethAccount, nil
}
