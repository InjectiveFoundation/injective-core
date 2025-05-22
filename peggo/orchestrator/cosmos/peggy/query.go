package peggy

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/metrics"
)

var ErrNotFound = errors.New("not found")

type QueryClient interface {
	PeggyParams(ctx context.Context) (*peggytypes.Params, error)
	LastClaimEventByAddr(ctx context.Context, validatorAccountAddress cosmostypes.AccAddress) (*peggytypes.LastClaimEvent, error)
	GetValidatorAddress(ctx context.Context, addr gethcommon.Address) (cosmostypes.AccAddress, error)

	ValsetAt(ctx context.Context, nonce uint64) (*peggytypes.Valset, error)
	CurrentValset(ctx context.Context) (*peggytypes.Valset, error)
	OldestUnsignedValsets(ctx context.Context, valAccountAddress cosmostypes.AccAddress) ([]*peggytypes.Valset, error)
	LatestValsets(ctx context.Context) ([]*peggytypes.Valset, error)
	AllValsetConfirms(ctx context.Context, nonce uint64) ([]*peggytypes.MsgValsetConfirm, error)

	OldestUnsignedTransactionBatch(ctx context.Context, valAccountAddress cosmostypes.AccAddress) (*peggytypes.OutgoingTxBatch, error)
	LatestTransactionBatches(ctx context.Context) ([]*peggytypes.OutgoingTxBatch, error)
	UnbatchedTokensWithFees(ctx context.Context) ([]*peggytypes.BatchFees, error)
	TransactionBatchSignatures(ctx context.Context, nonce uint64, tokenContract gethcommon.Address) ([]*peggytypes.MsgConfirmBatch, error)
}

type queryClient struct {
	peggytypes.QueryClient

	svcTags metrics.Tags
}

func NewQueryClient(client peggytypes.QueryClient) QueryClient {
	return queryClient{
		QueryClient: client,
		svcTags:     metrics.Tags{"svc": "peggy_query"},
	}
}

func (c queryClient) ValsetAt(ctx context.Context, nonce uint64) (*peggytypes.Valset, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	req := &peggytypes.QueryValsetRequestRequest{Nonce: nonce}

	resp, err := c.QueryClient.ValsetRequest(ctx, req)
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query ValsetRequest from client")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Valset, nil
}

func (c queryClient) CurrentValset(ctx context.Context) (*peggytypes.Valset, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	resp, err := c.QueryClient.CurrentValset(ctx, &peggytypes.QueryCurrentValsetRequest{})
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query CurrentValset from client")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Valset, nil
}

func (c queryClient) OldestUnsignedValsets(ctx context.Context, valAccountAddress cosmostypes.AccAddress) ([]*peggytypes.Valset, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	req := &peggytypes.QueryLastPendingValsetRequestByAddrRequest{
		Address: valAccountAddress.String(),
	}

	resp, err := c.QueryClient.LastPendingValsetRequestByAddr(ctx, req)
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query LastPendingValsetRequestByAddr from client")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Valsets, nil
}

func (c queryClient) LatestValsets(ctx context.Context) ([]*peggytypes.Valset, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	resp, err := c.QueryClient.LastValsetRequests(ctx, &peggytypes.QueryLastValsetRequestsRequest{})
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query LastValsetRequests from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Valsets, nil
}

func (c queryClient) AllValsetConfirms(ctx context.Context, nonce uint64) ([]*peggytypes.MsgValsetConfirm, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	resp, err := c.QueryClient.ValsetConfirmsByNonce(ctx, &peggytypes.QueryValsetConfirmsByNonceRequest{Nonce: nonce})
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query ValsetConfirmsByNonce from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Confirms, nil
}

func (c queryClient) OldestUnsignedTransactionBatch(ctx context.Context, valAccountAddress cosmostypes.AccAddress) (*peggytypes.OutgoingTxBatch, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	req := &peggytypes.QueryLastPendingBatchRequestByAddrRequest{
		Address: valAccountAddress.String(),
	}

	resp, err := c.QueryClient.LastPendingBatchRequestByAddr(ctx, req)
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query LastPendingBatchRequestByAddr from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Batch, nil
}

func (c queryClient) LatestTransactionBatches(ctx context.Context) ([]*peggytypes.OutgoingTxBatch, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	resp, err := c.QueryClient.OutgoingTxBatches(ctx, &peggytypes.QueryOutgoingTxBatchesRequest{})
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query OutgoingTxBatches from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Batches, nil
}

func (c queryClient) UnbatchedTokensWithFees(ctx context.Context) ([]*peggytypes.BatchFees, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	resp, err := c.QueryClient.BatchFees(ctx, &peggytypes.QueryBatchFeeRequest{})
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query BatchFees from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.BatchFees, nil
}

func (c queryClient) TransactionBatchSignatures(ctx context.Context, nonce uint64, tokenContract gethcommon.Address) ([]*peggytypes.MsgConfirmBatch, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	req := &peggytypes.QueryBatchConfirmsRequest{
		Nonce:           nonce,
		ContractAddress: tokenContract.String(),
	}

	resp, err := c.QueryClient.BatchConfirms(ctx, req)
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query BatchConfirms from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.Confirms, nil
}

func (c queryClient) LastClaimEventByAddr(ctx context.Context, validatorAccountAddress cosmostypes.AccAddress) (*peggytypes.LastClaimEvent, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	req := &peggytypes.QueryLastEventByAddrRequest{
		Address: validatorAccountAddress.String(),
	}

	resp, err := c.QueryClient.LastEventByAddr(ctx, req)
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query LastEventByAddr from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return resp.LastClaimEvent, nil
}

func (c queryClient) PeggyParams(ctx context.Context) (*peggytypes.Params, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	resp, err := c.QueryClient.Params(ctx, &peggytypes.QueryParamsRequest{})
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query PeggyParams from daemon")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	return &resp.Params, nil
}

func (c queryClient) GetValidatorAddress(ctx context.Context, addr gethcommon.Address) (cosmostypes.AccAddress, error) {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	req := &peggytypes.QueryDelegateKeysByEthAddress{
		EthAddress: addr.Hex(),
	}

	resp, err := c.QueryClient.GetDelegateKeyByEth(ctx, req)
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, errors.Wrap(err, "failed to query GetDelegateKeyByEth from client")
	}

	if resp == nil {
		metrics.ReportFuncError(c.svcTags)
		return nil, ErrNotFound
	}

	valAddr, err := cosmostypes.AccAddressFromBech32(resp.ValidatorAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode validator address: %v", resp.ValidatorAddress)
	}

	return valAddr, nil
}
