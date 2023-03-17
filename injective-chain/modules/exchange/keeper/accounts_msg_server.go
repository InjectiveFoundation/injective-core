package keeper

import (
	"context"

	"github.com/InjectiveLabs/metrics"
	sdksecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type AccountsMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// AccountsMsgServerImpl returns an implementation of the bank MsgServer interface for the provided Keeper for account functions.
func AccountsMsgServerImpl(keeper Keeper) AccountsMsgServer {
	return AccountsMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "acc_msg_h",
		},
	}
}

func (k AccountsMsgServer) BatchUpdateOrders(
	goCtx context.Context,
	msg *types.MsgBatchUpdateOrders,
) (*types.MsgBatchUpdateOrdersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	return k.ExecuteBatchUpdateOrders(
		ctx,
		sender,
		msg.SubaccountId,
		msg.SpotMarketIdsToCancelAll,
		msg.DerivativeMarketIdsToCancelAll,
		msg.BinaryOptionsMarketIdsToCancelAll,
		msg.SpotOrdersToCancel,
		msg.DerivativeOrdersToCancel,
		msg.BinaryOptionsOrdersToCancel,
		msg.SpotOrdersToCreate,
		msg.DerivativeOrdersToCreate,
		msg.BinaryOptionsOrdersToCreate,
	)
}

func (k AccountsMsgServer) Deposit(
	goCtx context.Context,
	msg *types.MsgDeposit,
) (*types.MsgDepositResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.executeDeposit(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgDepositResponse{}, nil
}

func (k AccountsMsgServer) Withdraw(
	goCtx context.Context,
	msg *types.MsgWithdraw,
) (*types.MsgWithdrawResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.ExecuteWithdraw(ctx, msg); err != nil {
		return nil, err
	}
	return &types.MsgWithdrawResponse{}, nil
}

func (k AccountsMsgServer) SubaccountTransfer(
	goCtx context.Context,
	msg *types.MsgSubaccountTransfer,
) (*types.MsgSubaccountTransferResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	srcSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
	dstSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.DestinationSubaccountId)

	denom := msg.Amount.Denom
	amount := msg.Amount.Amount.ToDec()

	if err := k.Keeper.DecrementDeposit(ctx, srcSubaccountID, denom, amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if err := k.Keeper.IncrementDepositForNonDefaultSubaccount(ctx, dstSubaccountID, denom, amount); err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	return &types.MsgSubaccountTransferResponse{}, nil
}

func (k AccountsMsgServer) ExternalTransfer(
	goCtx context.Context,
	msg *types.MsgExternalTransfer,
) (*types.MsgExternalTransferResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	srcSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
	dstSubaccountID := common.HexToHash(msg.DestinationSubaccountId)

	denom := msg.Amount.Denom
	amount := msg.Amount.Amount.ToDec()

	if err := k.Keeper.DecrementDeposit(ctx, srcSubaccountID, denom, amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	recipientAddr := types.SubaccountIDToSdkAddress(dstSubaccountID)

	// create new account for recipient if it doesn't exist already
	if !k.AccountKeeper.HasAccount(ctx, recipientAddr) {
		defer telemetry.IncrCounter(1, "new", "account")
		k.AccountKeeper.SetAccount(ctx, k.AccountKeeper.NewAccountWithAddress(ctx, recipientAddr))
	}

	if err := k.Keeper.IncrementDepositForNonDefaultSubaccount(ctx, dstSubaccountID, denom, amount); err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	return &types.MsgExternalTransferResponse{}, nil
}

func (k AccountsMsgServer) RewardsOptOut(
	goCtx context.Context,
	msg *types.MsgRewardsOptOut,
) (*types.MsgRewardsOptOutResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	account, _ := sdk.AccAddressFromBech32(msg.Sender)
	isAlreadyOptedOut := k.GetIsOptedOutOfRewards(ctx, account)

	if isAlreadyOptedOut {
		return nil, types.ErrAlreadyOptedOutOfRewards
	}

	k.SetIsOptedOutOfRewards(ctx, account, true)

	return &types.MsgRewardsOptOutResponse{}, nil
}

func (k AccountsMsgServer) ReclaimLockedFunds(
	goCtx context.Context,
	msg *types.MsgReclaimLockedFunds,
) (*types.MsgReclaimLockedFundsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	lockedPubKey := sdksecp256k1.PubKey{
		Key: msg.LockedAccountPubKey,
	}
	lockedAddress := sdk.AccAddress(lockedPubKey.Address())
	lockedAccount := k.AccountKeeper.GetAccount(ctx, lockedAddress)

	// only allow unlocking the funds if the locked account has never been used, since otherwise, this indicates
	// the funds aren't actually locked
	if lockedAccount == nil || lockedAccount.GetSequence() > 0 {
		return nil, types.ErrInvalidAddress
	}

	balances := k.bankKeeper.GetAllBalances(ctx, lockedAddress)
	balancesToUnlock := sdk.NewCoins()

	for _, coin := range balances {
		// for security, don't transfer peggy denoms since these were likely transferred from Ethereum
		if types.IsPeggyToken(coin.Denom) || coin.IsZero() {
			continue
		}
		balancesToUnlock = balancesToUnlock.Add(coin)
	}

	if balancesToUnlock.IsZero() {
		return nil, types.ErrNoFundsToUnlock
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, lockedAddress, types.ModuleName, balancesToUnlock); err != nil {
		return nil, err
	}

	correctPubKey := ethsecp256k1.PubKey{
		Key: lockedPubKey.Bytes(),
	}
	recipient := sdk.AccAddress(correctPubKey.Address())

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, balancesToUnlock); err != nil {
		return nil, err
	}

	return &types.MsgReclaimLockedFundsResponse{}, nil
}
