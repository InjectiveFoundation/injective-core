package keeper

import (
	"bytes"
	"context"

	"github.com/InjectiveLabs/metrics"
	sdksecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

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

func (k AccountsMsgServer) TransferAndExecute(
	goCtx context.Context,
	msg *types.MsgTransferAndExecute,
) (*types.MsgTransferAndExecuteResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	defaultSubaccountID := types.SdkAddressToSubaccountID(sender)

	switch msg.FundsDirection {
	case types.FundsDirection_BANK_TO_SUBACCOUNT:
		for _, coin := range msg.Funds {
			if err := k.executeDeposit(ctx, &types.MsgDeposit{
				Sender:       msg.Sender,
				SubaccountId: defaultSubaccountID.Hex(),
				Amount:       coin,
			}); err != nil {
				return nil, err
			}

		}

	case types.FundsDirection_SUBACCOUNT_TO_BANK:
		for _, coin := range msg.Funds {
			if err := k.executeWithdraw(ctx, &types.MsgWithdraw{
				Sender:       msg.Sender,
				SubaccountId: defaultSubaccountID.Hex(),
				Amount:       coin,
			}); err != nil {
				return nil, err
			}
		}
	}

	if err := k.executeMsg(ctx, msg.Msg); err != nil {
		return nil, err
	}

	return &types.MsgTransferAndExecuteResponse{}, nil
}

func (k AccountsMsgServer) MultiExecute(
	goCtx context.Context,
	msg *types.MsgMultiExecute,
) (*types.MsgMultiExecuteResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	for idx := range msg.Msgs {
		if err := k.executeMsg(ctx, msg.Msgs[idx]); err != nil {
			return nil, err
		}
	}

	return &types.MsgMultiExecuteResponse{}, nil
}

func (k AccountsMsgServer) executeMsg(
	ctx sdk.Context,
	msg *codectypes.Any,
) error {
	wrappedMsg, ok := msg.GetCachedValue().(sdk.Msg)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "message contains %T which is not a sdk.MsgRequest", wrappedMsg)
	}

	handler := k.router.Handler(wrappedMsg)
	if handler == nil {
		return sdkerrors.ErrUnknownRequest.Wrapf("unrecognized message route: %s", sdk.MsgTypeURL(wrappedMsg))
	}

	msgResp, err := handler(ctx, wrappedMsg)
	if err != nil {
		return sdkerrors.Wrapf(err, "failed to execute message; message %v", msg)
	}

	// emit the events from the dispatched actions
	events := msgResp.Events
	sdkEvents := make([]sdk.Event, 0, len(events))
	for i := 0; i < len(events); i++ {
		sdkEvents = append(sdkEvents, sdk.Event(events[i]))
	}

	ctx.EventManager().EmitEvents(sdkEvents)
	return nil
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
	logger := k.logger.WithFields(log.WithFn())

	if !k.IsDenomValid(ctx, msg.Amount.Denom) {
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.ErrInvalidCoins
	}

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, sdk.NewCoins(msg.Amount)); err != nil {
		metrics.ReportFuncError(k.svcTags)
		logger.Error("subaccount deposit failed", "senderAddr", senderAddr.String(), "coin", msg.Amount.String())
		return nil, sdkerrors.Wrap(err, "deposit failed")
	}

	subaccountID := common.HexToHash(msg.SubaccountId)

	if bytes.Equal(subaccountID.Bytes(), types.ZeroSubaccountID.Bytes()) {
		subaccountID = types.SdkAddressToSubaccountID(senderAddr)
	}

	recipientAddr := types.SubaccountIDToSdkAddress(subaccountID)

	// create new account for recipient if it doesn't exist already
	if !k.AccountKeeper.HasAccount(ctx, recipientAddr) {
		defer telemetry.IncrCounter(1, "new", "account")
		k.AccountKeeper.SetAccount(ctx, k.AccountKeeper.NewAccountWithAddress(ctx, recipientAddr))
	}

	k.IncrementDeposit(ctx, subaccountID, msg.Amount)
	//nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountDeposit{
		SrcAddress:   msg.Sender,
		SubaccountId: subaccountID.Bytes(),
		Amount:       msg.Amount,
	})

	return &types.MsgDepositResponse{}, nil
}

func (k AccountsMsgServer) Withdraw(
	goCtx context.Context,
	msg *types.MsgWithdraw,
) (*types.MsgWithdrawResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.executeWithdraw(ctx, msg); err != nil {
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
	logger := k.logger.WithFields(log.WithFn())

	srcSubaccountID := common.HexToHash(msg.SourceSubaccountId)
	if err := k.Keeper.WithdrawDeposit(ctx, srcSubaccountID, msg.Amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	dstSubaccountID := common.HexToHash(msg.DestinationSubaccountId)
	k.Keeper.IncrementDeposit(ctx, dstSubaccountID, msg.Amount) // convert to hash?

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	logger.Infof("Successfully transferred %s of Coin %s between subaccount %s to %s",
		msg.Amount.Amount.String(), msg.Amount.Denom, srcSubaccountID.Hex(), dstSubaccountID.Hex())

	return &types.MsgSubaccountTransferResponse{}, nil
}

func (k AccountsMsgServer) ExternalTransfer(
	goCtx context.Context,
	msg *types.MsgExternalTransfer,
) (*types.MsgExternalTransferResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	// default subaccount ID:
	srcSubaccountID := common.HexToHash(msg.SourceSubaccountId)
	if err := k.Keeper.WithdrawDeposit(ctx, srcSubaccountID, msg.Amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// withdraw from default subaccount
	dstSubaccountID := common.HexToHash(msg.DestinationSubaccountId)

	recipientAddr := types.SubaccountIDToSdkAddress(dstSubaccountID)

	// create new account for recipient if it doesn't exist already
	if !k.AccountKeeper.HasAccount(ctx, recipientAddr) {
		defer telemetry.IncrCounter(1, "new", "account")
		k.AccountKeeper.SetAccount(ctx, k.AccountKeeper.NewAccountWithAddress(ctx, recipientAddr))
	}

	k.Keeper.IncrementDeposit(ctx, dstSubaccountID, msg.Amount)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	logger.Debugf("Successfully transferred %s of Coin %s to external account %s to %s",
		msg.Amount.Amount.String(), msg.Amount.Denom, srcSubaccountID.Hex(), dstSubaccountID.Hex())

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
