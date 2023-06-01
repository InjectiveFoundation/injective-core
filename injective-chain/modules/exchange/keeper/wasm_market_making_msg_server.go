package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

type WasmMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewWasmMsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper for exchange wasm functions.
func NewWasmMsgServerImpl(keeper Keeper) WasmMsgServer {
	return WasmMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "exch_wasm_msg_h",
		},
	}
}

func (k WasmMsgServer) PrivilegedExecuteContract(
	goCtx context.Context,
	msg *types.MsgPrivilegedExecuteContract,
) (*types.MsgPrivilegedExecuteContractResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.Logger(ctx).Debug("=============== ⭐️ [Start] PrivilegedExecuteContract ⭐️ ===============")

	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	contract, _ := sdk.AccAddressFromBech32(msg.ContractAddress)

	fundsBefore := sdk.Coins(make([]sdk.Coin, 0, len(msg.Funds)))
	totalFunds := sdk.Coins{}

	// Enforce sender has sufficient funds for execution
	if !msg.HasEmptyFunds() {
		coins, err := sdk.ParseCoinsNormalized(msg.Funds)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse coins %s", msg.Funds)
		}

		for _, coin := range coins {
			coinBefore := k.bankKeeper.GetBalance(ctx, sender, coin.Denom)
			fundsBefore = fundsBefore.Add(coinBefore)
		}

		if err := k.bankKeeper.SendCoins(ctx, sender, contract, coins); err != nil {
			return nil, errors.Wrap(err, "failed to send coins")
		}
		totalFunds = coins
	}

	execMsg, err := wasmxtypes.NewInjectiveExecMsg(sender, msg.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create exec msg")
	}

	res, err := k.wasmxExecutionKeeper.InjectiveExec(ctx, contract, totalFunds, execMsg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute msg")
	}

	action, err := types.ParseRequest(res)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute msg")
	}

	if action != nil {
		err = k.handlePrivilegedAction(ctx, contract, sender, action)
		if err != nil {
			return nil, errors.Wrap(err, "failed to execute msg")
		}
	}

	fundsAfter := sdk.Coins(make([]sdk.Coin, 0, len(msg.Funds)))

	for _, coin := range fundsBefore {
		coinAfter := k.bankKeeper.GetBalance(ctx, sender, coin.Denom)
		fundsAfter = fundsAfter.Add(coinAfter)
	}

	fundsDiff, _ := fundsAfter.SafeSub(fundsBefore...)
	filteredFundsDiff := filterNonPositiveCoins(fundsDiff)

	if err != nil {
		k.Logger(ctx).Error("PrivilegedExecuteContract: Unable to parse coins", err)
	}

	k.Logger(ctx).Debug("=============== 🛏️ [End] Exec 🛏️ ===============")
	return &types.MsgPrivilegedExecuteContractResponse{
		FundsDiff: filteredFundsDiff,
	}, nil
}

func filterNonPositiveCoins(coins sdk.Coins) sdk.Coins {
	var filteredCoins sdk.Coins
	for _, coin := range coins {
		if coin.IsPositive() {
			filteredCoins = append(filteredCoins, coin)
		}
	}
	return filteredCoins
}
