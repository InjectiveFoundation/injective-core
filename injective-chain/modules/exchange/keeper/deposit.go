package keeper

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/metrics"

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k *Keeper) executeDeposit(ctx sdk.Context, msg *types.MsgDeposit) error {
	logger := k.logger.WithFields(log.WithFn())

	if !k.IsDenomValid(ctx, msg.Amount.Denom) {
		metrics.ReportFuncError(k.svcTags)
		return sdkerrors.ErrInvalidCoins
	}

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, sdk.NewCoins(msg.Amount)); err != nil {
		metrics.ReportFuncError(k.svcTags)
		logger.Error("subaccount deposit failed", "senderAddr", senderAddr.String(), "coin", msg.Amount.String())
		return sdkerrors.Wrap(err, "deposit failed")
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
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountDeposit{
		SrcAddress:   msg.Sender,
		SubaccountId: subaccountID.Bytes(),
		Amount:       msg.Amount,
	})

	return nil
}

func (k *Keeper) executeWithdraw(ctx sdk.Context, msg *types.MsgWithdraw) error {
	logger := k.logger.WithFields(log.WithFn())

	subaccountID := common.HexToHash(msg.SubaccountId)

	if !k.IsDenomValid(ctx, msg.Amount.Denom) {
		metrics.ReportFuncError(k.svcTags)
		return sdkerrors.ErrInvalidCoins
	}

	if err := k.WithdrawDeposit(ctx, subaccountID, msg.Amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return sdkerrors.Wrap(err, "withdrawal failed")
	}

	withdrawDestAddr, _ := sdk.AccAddressFromBech32(msg.Sender)

	err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawDestAddr, sdk.NewCoins(msg.Amount))
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		logger.Error("subaccount withdrawal failed", "senderAddr", withdrawDestAddr.String(), "coin", msg.Amount.String())
		return sdkerrors.Wrap(err, "withdrawal failed")
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountWithdraw{
		SubaccountId: subaccountID.Bytes(),
		DstAddress:   msg.Sender,
		Amount:       msg.Amount,
	})

	return nil
}

// IsDenomValid checks if denom is a valid denomination in the bank module supply.
func (k *Keeper) IsDenomValid(ctx sdk.Context, denom string) bool {
	return k.bankKeeper.GetSupply(ctx, denom).Amount.IsPositive()
}

// GetDeposit gets a subaccount's deposit for a given denom.
func (k *Keeper) GetDeposit(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
) *types.Deposit {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetDepositKey(subaccountID, denom)

	bz := store.Get(key)
	if bz == nil {
		return types.NewDeposit()
	}

	var deposit types.Deposit
	k.cdc.MustUnmarshal(bz, &deposit)

	if deposit.TotalBalance.IsNil() {
		deposit.TotalBalance = sdk.ZeroDec()
	}

	if deposit.AvailableBalance.IsNil() {
		deposit.AvailableBalance = sdk.ZeroDec()
	}

	return &deposit
}

// SetDeposit sets a subaccount's deposit for a given denom.
func (k *Keeper) SetDeposit(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	deposit *types.Deposit,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetDepositKey(subaccountID, denom)
	bz := k.cdc.MustMarshal(deposit)
	store.Set(key, bz)

	k.SetTransientDeposit(ctx, subaccountID, denom, deposit)
}

// GetDeposits gets all the deposits for all of the subaccount's denoms.
func (k *Keeper) GetDeposits(
	ctx sdk.Context,
	subaccountID common.Hash,
) map[string]*types.Deposit {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	keyPrefix := types.GetDepositKeyPrefixBySubaccountID(subaccountID)
	depositStore := prefix.NewStore(store, keyPrefix)

	iterator := depositStore.Iterator(nil, nil)
	defer iterator.Close()

	deposits := make(map[string]*types.Deposit)

	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &deposit)
		denom := string(iterator.Key())
		deposits[denom] = &deposit
	}
	return deposits
}

func (k *Keeper) WithdrawAllAuctionBalances(ctx sdk.Context) []types.Balance {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	depositStore := prefix.NewStore(store, types.GetDepositKeyPrefixBySubaccountID(types.AuctionSubaccountID))
	iterator := depositStore.Iterator(nil, nil)
	defer iterator.Close()
	balances := make([]types.Balance, 0)

	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &deposit)
		if deposit.TotalBalance.IsNil() || deposit.TotalBalance.IsZero() || deposit.TotalBalance.TruncateInt().IsZero() {
			continue
		}
		denom := string(iterator.Key())
		balances = append(balances, types.Balance{
			SubaccountId: types.AuctionSubaccountID.Hex(),
			Denom:        denom,
			Deposits:     &deposit,
		})
	}

	// transfer funds from distribution module address
	distributionModuleAddr := k.AccountKeeper.GetModuleAddress(distributiontypes.ModuleName)
	if distributionModuleAddr != nil {
		distributionSubaccountID := types.SdkAddressToSubaccountID(distributionModuleAddr)
		deposits := k.GetDeposits(ctx, distributionSubaccountID)

		denoms := types.GetSortedBalanceKeys(deposits)

		for _, denom := range denoms {
			deposit := deposits[denom]
			if deposit.TotalBalance.IsNil() || deposit.TotalBalance.IsZero() || deposit.TotalBalance.TruncateInt().IsZero() {
				continue
			}
			// impossible, but just in case
			if deposit.AvailableBalance.LT(deposit.TotalBalance) {
				continue
			}
			balances = append(balances, types.Balance{
				SubaccountId: distributionSubaccountID.Hex(),
				Denom:        denom,
				Deposits:     deposit,
			})
		}
	}

	// transfer funds from marketID "subaccounts" to each market's respective insurance fund
	markets := k.GetAllActiveDerivativeMarkets(ctx)
	for _, market := range markets {
		marketID := market.MarketID()
		marketSubaccountID := types.SdkAddressToSubaccountID(types.SubaccountIDToSdkAddress(marketID))
		deposit := k.GetDeposit(ctx, marketSubaccountID, market.QuoteDenom)

		depositAmount := deposit.AvailableBalance.TruncateInt()
		if depositAmount.IsZero() {
			continue
		}

		k.decrementBothDeposits(ctx, marketSubaccountID, market.QuoteDenom, depositAmount.ToDec())

		if err := k.insuranceKeeper.DepositIntoInsuranceFund(ctx, marketID, depositAmount); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}

		coinAmount := sdk.NewCoins(sdk.NewCoin(market.QuoteDenom, depositAmount))
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, insurancetypes.ModuleName, coinAmount); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}

	coinsToSend := sdk.NewCoins()
	for _, balance := range balances {
		withdrawIntAmount := balance.Deposits.TotalBalance.TruncateInt()
		withdrawAmount := withdrawIntAmount.ToDec()
		k.decrementBothDeposits(ctx, common.HexToHash(balance.SubaccountId), balance.Denom, withdrawAmount)
		if balance.Denom == chaintypes.InjectiveCoin {
			injBurnAmount := sdk.NewCoins(sdk.NewCoin(chaintypes.InjectiveCoin, withdrawIntAmount))
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, injBurnAmount); err != nil {
				k.logger.Warningln("BurnCoins fail:", err)
			}
		} else {
			coinsToSend = coinsToSend.Add(sdk.NewCoin(balance.Denom, withdrawIntAmount))
		}
	}

	if len(coinsToSend) > 0 {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, auctiontypes.ModuleName, coinsToSend); err != nil {
			k.logger.Errorln(err)
		}
	}

	return balances
}

// GetAllExchangeBalances returns the exchange balances.
func (k *Keeper) GetAllExchangeBalances(
	ctx sdk.Context,
) []types.Balance {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	depositStore := prefix.NewStore(store, types.DepositsPrefix)
	iterator := depositStore.Iterator(nil, nil)
	defer iterator.Close()

	balances := make([]types.Balance, 0)

	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &deposit)
		subaccountID, denom := types.ParseDepositStoreKey(iterator.Key())
		balances = append(balances, types.Balance{
			SubaccountId: subaccountID.Hex(),
			Denom:        denom,
			Deposits:     &deposit,
		})
	}

	return balances
}

func (k *Keeper) decrementBothDeposits(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	absoluteDecrementAmount sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if absoluteDecrementAmount.IsZero() {
		return
	}

	k.UpdateDepositWithDelta(ctx, subaccountID, denom, &types.DepositDelta{
		AvailableBalanceDelta: absoluteDecrementAmount.Neg(),
		TotalBalanceDelta:     absoluteDecrementAmount.Neg(),
	})
}

// UpdateDepositWithDelta applies a deposit delta to the existing deposit.
func (k *Keeper) UpdateDepositWithDelta(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	depositDelta *types.DepositDelta,
) *types.Deposit {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if depositDelta == nil {
		return nil
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)

	if depositDelta.TotalBalanceDelta.IsNil() {
		depositDelta.TotalBalanceDelta = sdk.ZeroDec()
	}

	if depositDelta.AvailableBalanceDelta.IsNil() {
		depositDelta.AvailableBalanceDelta = sdk.ZeroDec()
	}

	if depositDelta.TotalBalanceDelta.IsZero() && depositDelta.AvailableBalanceDelta.IsZero() {
		return deposit
	}

	deposit.AvailableBalance = deposit.AvailableBalance.Add(depositDelta.AvailableBalanceDelta)
	deposit.TotalBalance = deposit.TotalBalance.Add(depositDelta.TotalBalanceDelta)
	k.SetDeposit(ctx, subaccountID, denom, deposit)

	return deposit
}

// IncrementDeposit increments a given subaccount's deposits by a given coin amount.
func (k *Keeper) IncrementDeposit(
	ctx sdk.Context,
	subaccountID common.Hash,
	coin sdk.Coin,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	deposit := k.GetDeposit(ctx, subaccountID, coin.Denom)
	decAmount := coin.Amount.ToDec()
	deposit.AvailableBalance = deposit.AvailableBalance.Add(decAmount)
	deposit.TotalBalance = deposit.TotalBalance.Add(decAmount)
	k.SetDeposit(ctx, subaccountID, coin.Denom, deposit)
}

// WithdrawDeposit withdraws funds from decrements a given subaccount's deposits by a given coin amount.
func (k *Keeper) WithdrawDeposit(
	ctx sdk.Context,
	subaccountID common.Hash,
	coin sdk.Coin,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	deposit := k.GetDeposit(ctx, subaccountID, coin.Denom)
	withdrawAmount := coin.Amount.ToDec()

	// usually available balance check is sufficient, but in case of a bug, we check total balance as well
	if deposit.IsEmpty() || deposit.AvailableBalance.LT(withdrawAmount) || deposit.TotalBalance.LT(withdrawAmount) {
		metrics.ReportFuncError(k.svcTags)
		return types.ErrInsufficientDeposit
	}

	deposit.AvailableBalance = deposit.AvailableBalance.Sub(withdrawAmount)
	deposit.TotalBalance = deposit.TotalBalance.Sub(withdrawAmount)
	k.SetDeposit(ctx, subaccountID, coin.Denom, deposit)
	return nil
}
