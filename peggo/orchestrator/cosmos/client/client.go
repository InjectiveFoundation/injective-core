package client

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	log "github.com/InjectiveLabs/suplog"
)

const (
	DefaultGasPrice = "160000000inj"
)

type Options struct {
	GasPrices string
}

type ChainClient struct {
	ctx                   sdkclient.Context
	logger                log.Logger
	txFactory             tx.Factory
	tendermintQueryClient cmtservice.ServiceClient
	txClient              txtypes.ServiceClient
	accNum                uint64
	accSeq                uint64
}

func NewChainClient(ctx sdkclient.Context, opts *Options) (*ChainClient, error) {
	txf := tx.Factory{}
	txf = txf.WithKeybase(ctx.Keyring)
	txf = txf.WithTxConfig(ctx.TxConfig)
	txf = txf.WithAccountRetriever(ctx.AccountRetriever)
	txf = txf.WithSimulateAndExecute(true)
	txf = txf.WithGasAdjustment(1.5)
	txf = txf.WithChainID(ctx.ChainID)
	txf = txf.WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
	txf = txf.WithFromName(ctx.GetFromName())
	txf = txf.WithGasPrices(DefaultGasPrice)

	if opts != nil {
		if _, err := sdk.ParseDecCoins(opts.GasPrices); err != nil {
			return nil, errors.Wrapf(err, "failed to parse gas prices %s", opts.GasPrices)
		}

		txf = txf.WithGasPrices(opts.GasPrices)
	}

	acc, err := txf.AccountRetriever().GetAccount(ctx, ctx.GetFromAddress())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get account: %s", ctx.GetFromAddress())
	}

	cc := &ChainClient{
		ctx:                   ctx,
		logger:                log.WithFields(log.Fields{"svc": "cosmos_client"}),
		txFactory:             txf,
		txClient:              txtypes.NewServiceClient(ctx.GRPCClient),
		tendermintQueryClient: cmtservice.NewServiceClient(ctx.GRPCClient),
		accNum:                acc.GetAccountNumber(),
		accSeq:                acc.GetSequence(),
	}

	return cc, nil
}

func (c *ChainClient) syncNonce() {
	num, seq, err := c.txFactory.AccountRetriever().GetAccountNumberSequence(c.ctx, c.ctx.GetFromAddress())
	if err != nil {
		c.logger.WithError(err).Errorln("failed to get account seq")
		return
	}

	if num != c.accNum {
		c.logger.WithFields(log.Fields{
			"expected": c.accNum,
			"actual":   num,
		}).Panic("account number changed during nonce sync")
	}

	c.accSeq = seq
}

// PrepareFactory ensures the account defined by ctx.GetFromAddress() exists and
// if the account number and/or the account sequence number are zero (not set),
// they will be queried for and set on the provided Factory. A new Factory with
// the updated fields will be returned.
func prepareFactory(clientCtx sdkclient.Context, txf tx.Factory) (tx.Factory, error) {
	from := clientCtx.GetFromAddress()
	if err := txf.AccountRetriever().EnsureExists(clientCtx, from); err != nil {
		return txf, err
	}

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err := txf.AccountRetriever().GetAccountNumberSequence(clientCtx, from)
		if err != nil {
			return txf, err
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}

	return txf, nil
}

func (c *ChainClient) getAccSeq() uint64 {
	defer func() { c.accSeq++ }()
	return c.accSeq
}

func (c *ChainClient) FromAddress() sdk.AccAddress {
	return c.ctx.FromAddress
}

type APICall[Q any, R any] func(ctx context.Context, in *Q, opts ...grpc.CallOption) (*R, error)

func callAPI[Q any, R any](ctx context.Context, call APICall[Q, R], in *Q) (*R, error) {
	localCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs())

	var header metadata.MD
	response, err := call(localCtx, in, grpc.Header(&header))

	return response, err
}

func (c *ChainClient) FetchLatestBlock(ctx context.Context) (*cmtservice.GetLatestBlockResponse, error) {
	req := &cmtservice.GetLatestBlockRequest{}
	res, err := callAPI(ctx, c.tendermintQueryClient.GetLatestBlock, req)

	return res, err
}

// BroadcastMsg submits a group of messages in one transaction to the chain
// The function uses the broadcast mode specified with the broadcastMode parameter
func (c *ChainClient) BroadcastMsg(
	broadcastMode txtypes.BroadcastMode,
	msgs ...sdk.Msg,
) (*txtypes.BroadcastTxRequest, *txtypes.BroadcastTxResponse, error) {
	sequence := c.getAccSeq()
	c.txFactory = c.txFactory.WithSequence(sequence)
	c.txFactory = c.txFactory.WithAccountNumber(c.accNum)

	req, res, err := c.broadcastTx(broadcastMode, msgs...)
	if err != nil {
		if strings.Contains(err.Error(), "account sequence mismatch") {
			c.syncNonce()
			sequence := c.getAccSeq()
			c.txFactory = c.txFactory.WithSequence(sequence)
			c.txFactory = c.txFactory.WithAccountNumber(c.accNum)
			c.logger.Debugln("retrying broadcastTx with nonce", sequence)
			req, res, err = c.broadcastTx(broadcastMode, msgs...)
		}
		if err != nil {
			resJSON, _ := json.MarshalIndent(res, "", "\t")
			c.logger.WithField("size", len(msgs)).WithError(err).Errorln(
				"failed to asynchronously broadcast messages:", string(resJSON),
			)

			return nil, nil, err
		}
	}

	return req, res, nil
}

func (c *ChainClient) broadcastTx(
	broadcastMode txtypes.BroadcastMode,
	msgs ...sdk.Msg,
) (*txtypes.BroadcastTxRequest, *txtypes.BroadcastTxResponse, error) {
	txBytes, err := c.buildSignedTx(c.ctx, c.txFactory, msgs...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to build signed Tx")
	}

	req := txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    broadcastMode,
	}

	res, err := callAPI(context.Background(), c.txClient.BroadcastTx, &req)
	return &req, res, err
}

func (c *ChainClient) buildSignedTx(clientCtx sdkclient.Context, txf tx.Factory, msgs ...sdk.Msg) ([]byte, error) {
	ctx := context.Background()
	if clientCtx.Simulate {
		simTxBytes, err := txf.BuildSimTx(msgs...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build sim tx bytes")
		}

		req := &txtypes.SimulateRequest{TxBytes: simTxBytes}
		simRes, err := callAPI(ctx, c.txClient.Simulate, req)

		if err != nil {
			return nil, errors.Wrap(err, "failed to CalculateGas")
		}

		adjustedGas := uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed))
		txf = txf.WithGas(adjustedGas)
	}

	txf, err := prepareFactory(clientCtx, txf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepareFactory")
	}

	txn, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to BuildUnsignedTx")
	}

	txn.SetFeeGranter(clientCtx.GetFeeGranterAddress())

	if err = tx.Sign(ctx, txf, clientCtx.GetFromName(), txn, true); err != nil {
		return nil, errors.Wrap(err, "failed to Sign Tx")
	}

	return clientCtx.TxConfig.TxEncoder()(txn.GetTx())
}
