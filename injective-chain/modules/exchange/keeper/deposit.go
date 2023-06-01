package keeper

import (
	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// MigrateExchangeBalances migrates the subaccount deposits for the new trading from bank balance flow.
func (k *Keeper) MigrateExchangeBalances(ctx sdk.Context, balance types.Balance) {
	subaccountID := common.HexToHash(balance.SubaccountId)
	denom := balance.Denom

	// only migrate default subaccount balances
	if !types.IsDefaultSubaccountID(subaccountID) {
		return
	}

	balanceChangeAmount := balance.Deposits.AvailableBalance.TruncateInt()

	// only migrate if available balance is at least 1, since bank balances are Ints
	if !balanceChangeAmount.IsPositive() {
		return
	}

	newDeposits := &types.Deposit{
		AvailableBalance: balance.Deposits.AvailableBalance.Sub(balanceChangeAmount.ToDec()),
		TotalBalance:     balance.Deposits.TotalBalance.Sub(balanceChangeAmount.ToDec()),
	}
	k.SetDeposit(ctx, subaccountID, denom, newDeposits)

	recipient := types.SubaccountIDToSdkAddress(subaccountID)
	coins := sdk.NewCoins(sdk.NewCoin(denom, balanceChangeAmount))

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, coins); err != nil {
		panic(err.Error())
	}
}

func (k *Keeper) executeDeposit(ctx sdk.Context, msg *types.MsgDeposit) error {

	if !k.IsDenomValid(ctx, msg.Amount.Denom) {
		metrics.ReportFuncError(k.svcTags)
		return sdkerrors.ErrInvalidCoins
	}

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, sdk.NewCoins(msg.Amount)); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("subaccount deposit failed", "senderAddr", senderAddr.String(), "coin", msg.Amount.String())
		return errors.Wrap(err, "deposit failed")
	}

	var subaccountID common.Hash
	var err error

	subaccountID, err = types.GetSubaccountIDOrDeriveFromNonce(senderAddr, msg.SubaccountId)
	if err != nil {
		// allow deposits to externally owned subaccounts
		subaccountID = common.HexToHash(msg.SubaccountId)
	}

	recipientAddr := types.SubaccountIDToSdkAddress(subaccountID)

	// create new account for recipient if it doesn't exist already
	if !k.AccountKeeper.HasAccount(ctx, recipientAddr) {
		defer telemetry.IncrCounter(1, "new", "account")
		k.AccountKeeper.SetAccount(ctx, k.AccountKeeper.NewAccountWithAddress(ctx, recipientAddr))
	}

	if err := k.IncrementDepositForNonDefaultSubaccount(ctx, subaccountID, msg.Amount.Denom, msg.Amount.Amount.ToDec()); err != nil {
		return err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountDeposit{
		SrcAddress:   msg.Sender,
		SubaccountId: subaccountID.Bytes(),
		Amount:       msg.Amount,
	})

	return nil
}

func (k *Keeper) ExecuteWithdraw(ctx sdk.Context, msg *types.MsgWithdraw) error {

	withdrawDestAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(withdrawDestAddr, msg.SubaccountId)

	denom := msg.Amount.Denom
	amount := msg.Amount.Amount.ToDec()

	if !k.IsDenomValid(ctx, denom) {
		metrics.ReportFuncError(k.svcTags)
		return sdkerrors.ErrInvalidCoins
	}

	if err := k.DecrementDeposit(ctx, subaccountID, denom, amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(err, "withdrawal failed")
	}

	err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawDestAddr, sdk.NewCoins(msg.Amount))
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("subaccount withdrawal failed", "senderAddr", withdrawDestAddr.String(), "coin", msg.Amount.String())
		return errors.Wrap(err, "withdrawal failed")
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

func (k *Keeper) GetSpendableFunds(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
) sdk.Dec {
	subaccountDeposits := k.GetDeposit(ctx, subaccountID, denom)
	if !types.IsDefaultSubaccountID(subaccountID) {
		return subaccountDeposits.AvailableBalance
	}

	// combine bankBalance + dust from subaccount deposits to get the total spendable funds
	bankBalance := k.bankKeeper.GetBalance(ctx, types.SubaccountIDToSdkAddress(subaccountID), denom)
	return bankBalance.Amount.ToDec().Add(subaccountDeposits.AvailableBalance)
}

func (k *Keeper) GetAllSpendableFunds(
	ctx sdk.Context,
	subaccountID common.Hash,
) map[string]sdk.Dec {
	spendableFunds := make(map[string]sdk.Dec, 0)
	subaccountDeposits := k.GetDeposits(ctx, subaccountID)
	for denom, deposit := range subaccountDeposits {
		spendableFunds[denom] = deposit.AvailableBalance
	}
	if types.IsDefaultSubaccountID(subaccountID) {
		bankBalances := k.bankKeeper.GetAllBalances(ctx, types.SubaccountIDToSdkAddress(subaccountID))
		for _, balance := range bankBalances {
			funds := spendableFunds[balance.Denom].Add(balance.Amount.ToDec())
			spendableFunds[balance.Denom] = funds
		}
	}
	return spendableFunds
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

// HasSufficientFunds returns true if the bank balances ≥ ceil(amount) for default subaccounts or if the availableBalance ≥ amount
// for non-default subaccounts.
func (k *Keeper) HasSufficientFunds(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) bool {
	isDefaultSubaccountID := types.IsDefaultSubaccountID(subaccountID)

	if isDefaultSubaccountID {
		bankBalance := k.bankKeeper.GetBalance(ctx, types.SubaccountIDToSdkAddress(subaccountID), denom)
		// take the ceiling since we need to round up to the nearest integer due to bank balances being integers
		return bankBalance.Amount.GTE(amount.Ceil().TruncateInt())
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)
	return deposit.AvailableBalance.GTE(amount)
}

// SetDepositOrSendToBank sets the deposit for a given subaccount and denom. If the subaccount is a default subaccount,
// the positive integer part of the availableDeposit is first sent to the account's bank balance and the deposits are
// set with only the remaining funds.
func (k *Keeper) SetDepositOrSendToBank(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	deposit types.Deposit,
	isPreventingBankCharge bool,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	amountToSendToBank := deposit.AvailableBalance.TruncateInt()

	// for default subaccounts, if the integer part of the available deposit funds are non-zero, send them to bank
	// otherwise, simply set the deposit to allow for dust accumulation
	shouldSendFundsToBank := amountToSendToBank.IsPositive() && types.IsDefaultSubaccountID(subaccountID)

	if shouldSendFundsToBank {
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName, // exchange module
			types.SubaccountIDToSdkAddress(subaccountID),
			sdk.NewCoins(sdk.NewCoin(denom, amountToSendToBank)),
		)

		deposit.AvailableBalance = deposit.AvailableBalance.Sub(amountToSendToBank.ToDec())
		deposit.TotalBalance = deposit.TotalBalance.Sub(amountToSendToBank.ToDec())
	} else {
		shouldChargeFromBank := !isPreventingBankCharge && deposit.AvailableBalance.IsNegative() && types.IsDefaultSubaccountID(subaccountID)

		if shouldChargeFromBank {
			amountToChargeFromBank := amountToSendToBank.Abs()

			if availableBalanceAfterCharge := deposit.AvailableBalance.Add(amountToChargeFromBank.ToDec()); availableBalanceAfterCharge.IsNegative() {
				amountToChargeFromBank = amountToChargeFromBank.AddRaw(1)
			}

			if err := k.chargeBank(ctx, types.SubaccountIDToSdkAddress(subaccountID), denom, amountToChargeFromBank); err == nil {
				deposit.AvailableBalance = deposit.AvailableBalance.Add(amountToChargeFromBank.ToDec())
				deposit.TotalBalance = deposit.TotalBalance.Add(amountToChargeFromBank.ToDec())
			}
		}
	}

	k.SetDeposit(ctx, subaccountID, denom, &deposit)
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

// MoveCoinsIntoCurrentAuction moves all the coins from exchange to auction module
func (k *Keeper) MoveCoinsIntoCurrentAuction(
	ctx sdk.Context,
	coinsToSend sdk.Coins,
) {
	if len(coinsToSend) == 0 {
		return
	}

	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, auctiontypes.ModuleName, coinsToSend); err != nil {
		k.Logger(ctx).Error(err.Error())
	}
}

type SendToAuctionCoin struct {
	SubaccountId string
	Denom        string
	Amount       sdkmath.Int
}

func (k *Keeper) WithdrawAllAuctionBalances(ctx sdk.Context) []SendToAuctionCoin {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	depositStore := prefix.NewStore(store, types.GetDepositKeyPrefixBySubaccountID(types.AuctionSubaccountID))
	iterator := depositStore.Iterator(nil, nil)
	defer iterator.Close()

	balances := make([]SendToAuctionCoin, 0)

	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit

		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &deposit)

		if deposit.TotalBalance.IsNil() || deposit.TotalBalance.IsZero() || deposit.TotalBalance.TruncateInt().IsZero() {
			continue
		}

		denom := string(iterator.Key())
		balances = append(balances, SendToAuctionCoin{
			SubaccountId: types.AuctionSubaccountID.Hex(),
			Denom:        denom,
			Amount:       deposit.TotalBalance.TruncateInt(),
		})
	}

	denomDecimals := k.GetAllDenomDecimals(ctx)
	coinsToSendFromAuctionAddress := sdk.NewCoins()

	// transfer funds from auction fee address
	for _, denomDecimal := range denomDecimals {
		balance := k.bankKeeper.GetBalance(ctx, types.AuctionFeesAddress, denomDecimal.Denom)

		if balance.IsNil() || !balance.IsPositive() {
			continue
		}

		balances = append(balances, SendToAuctionCoin{
			SubaccountId: "",
			Denom:        denomDecimal.Denom,
			Amount:       balance.Amount,
		})
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, types.AuctionFeesAddress, types.ModuleName, coinsToSendFromAuctionAddress); err != nil {
		k.Logger(ctx).Error(err.Error())
	}

	coinsToSend := sdk.NewCoins()

	for _, balance := range balances {
		if balance.SubaccountId != "" {
			if err := k.DecrementDeposit(ctx, common.HexToHash(balance.SubaccountId), balance.Denom, balance.Amount.ToDec()); err != nil {
				k.Logger(ctx).Error("WithdrawAllAuctionBalances DecrementDeposit fail:", err)
				continue
			}
		}

		if balance.Denom == chaintypes.InjectiveCoin {
			injBurnAmount := sdk.NewCoins(sdk.NewCoin(chaintypes.InjectiveCoin, balance.Amount))
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, injBurnAmount); err != nil {
				k.Logger(ctx).Error("BurnCoins fail:", err)
			}
		} else {
			coinsToSend = coinsToSend.Add(sdk.NewCoin(balance.Denom, balance.Amount))
		}
	}

	k.MoveCoinsIntoCurrentAuction(ctx, coinsToSend)

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

func (k *Keeper) chargeBank(ctx sdk.Context, account sdk.AccAddress, denom string, amount sdkmath.Int) error {
	if amount.IsZero() {
		return nil
	}

	coin := sdk.NewCoin(denom, amount)

	// ignores "locked" funds in the bank module, but not relevant to us since we don't have locked/vesting bank funds
	balance := k.bankKeeper.GetBalance(ctx, account, denom)
	if balance.Amount.LT(amount) {
		return errors.Wrapf(types.ErrInsufficientFunds, "%s is smaller than %s", balance, coin)
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, account, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("bank charge failed", "account", account.String(), "coin", coin.String())
		return errors.Wrap(err, "bank charge failed")
	}
	return nil
}

func (k *Keeper) chargeAvailableDeposits(ctx sdk.Context, subaccountID common.Hash, denom string, amount sdk.Dec) error {
	deposit := k.GetDeposit(ctx, subaccountID, denom)
	if deposit.IsEmpty() {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInsufficientDeposit, "Deposits for subaccountID %s asset %s not found", subaccountID.Hex(), denom)
	}

	if deposit.AvailableBalance.LT(amount) {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInsufficientDeposit, "Insufficient Deposits for subaccountID %s asset %s. Balance decrement %s exceeds Available Balance %s ", subaccountID.Hex(), denom, amount.String(), deposit.AvailableBalance.String())
	}

	deposit.AvailableBalance = deposit.AvailableBalance.Sub(amount)
	k.SetDeposit(ctx, subaccountID, denom, deposit)
	return nil
}

// chargeAccount transfers the amount from the available balance for non-default subaccounts or the corresponding bank balance if
// the subaccountID is a default subaccount. If bank balances are charged, the total deposits are incremented.
func (k *Keeper) chargeAccount(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if amount.IsZero() {
		return nil
	}

	if amount.IsNegative() {
		return errors.Wrapf(types.ErrInvalidAmount, "amount charged %s for denom %s must not be negative", amount.String(), denom)
	}

	if types.IsDefaultSubaccountID(subaccountID) {
		return k.chargeBankAndIncrementTotalDeposits(ctx, subaccountID, denom, amount)
	}

	return k.chargeAvailableDeposits(ctx, subaccountID, denom, amount)
}

func (k *Keeper) chargeBankAndIncrementTotalDeposits(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) error {
	sender := types.SubaccountIDToSdkAddress(subaccountID)
	// round up decimal portion (if exists) - truncation is fine here since we do Ceil first
	intAmount := amount.Ceil().TruncateInt()
	decAmount := intAmount.ToDec()

	if err := k.chargeBank(ctx, sender, denom, intAmount); err != nil {
		return err
	}

	// increase available balances by the additional decimal amount charged due to ceil(amount).Int() conversion
	// to ensure that the account does not lose dust, since the account may have been slightly overcharged
	extraChargedAmount := decAmount.Sub(amount)

	k.UpdateDepositWithDelta(ctx, subaccountID, denom, &types.DepositDelta{
		AvailableBalanceDelta: extraChargedAmount,
		TotalBalanceDelta:     decAmount,
	})
	return nil
}

func (k *Keeper) incrementAvailableBalanceOrBank(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if amount.IsZero() {
		return
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)
	deposit.AvailableBalance = deposit.AvailableBalance.Add(amount)
	k.SetDepositOrSendToBank(ctx, subaccountID, denom, *deposit, false)
}

// UpdateDepositWithDelta applies a deposit delta to the existing deposit.
func (k *Keeper) UpdateDepositWithDelta(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	depositDelta *types.DepositDelta,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if depositDelta.IsEmpty() {
		return
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)
	deposit.AvailableBalance = deposit.AvailableBalance.Add(depositDelta.AvailableBalanceDelta)
	deposit.TotalBalance = deposit.TotalBalance.Add(depositDelta.TotalBalanceDelta)
	k.SetDepositOrSendToBank(ctx, subaccountID, denom, *deposit, false)
}

// UpdateDepositWithDeltaWithoutBankCharge applies a deposit delta to the existing deposit.
func (k *Keeper) UpdateDepositWithDeltaWithoutBankCharge(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	depositDelta *types.DepositDelta,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if depositDelta.IsEmpty() {
		return
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)
	deposit.AvailableBalance = deposit.AvailableBalance.Add(depositDelta.AvailableBalanceDelta)
	deposit.TotalBalance = deposit.TotalBalance.Add(depositDelta.TotalBalanceDelta)
	k.SetDepositOrSendToBank(ctx, subaccountID, denom, *deposit, true)
}

// IncrementDepositWithCoinOrSendToBank increments a given subaccount's deposits by a given coin amount.
func (k *Keeper) IncrementDepositWithCoinOrSendToBank(
	ctx sdk.Context,
	subaccountID common.Hash,
	coin sdk.Coin,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	decAmount := coin.Amount.ToDec()
	k.IncrementDepositOrSendToBank(ctx, subaccountID, coin.Denom, decAmount)
}

// IncrementDepositForNonDefaultSubaccount increments a given non-default subaccount's deposits by a given dec amount for a given denom
func (k *Keeper) IncrementDepositForNonDefaultSubaccount(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if types.IsDefaultSubaccountID(subaccountID) {
		return errors.Wrap(types.ErrBadSubaccountID, subaccountID.Hex())
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)
	deposit.AvailableBalance = deposit.AvailableBalance.Add(amount)
	deposit.TotalBalance = deposit.TotalBalance.Add(amount)
	k.SetDeposit(ctx, subaccountID, denom, deposit)
	return nil
}

// IncrementDepositOrSendToBank increments a given subaccount's deposits by a given dec amount for a given denom
func (k *Keeper) IncrementDepositOrSendToBank(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	deposit := k.GetDeposit(ctx, subaccountID, denom)
	deposit.AvailableBalance = deposit.AvailableBalance.Add(amount)
	deposit.TotalBalance = deposit.TotalBalance.Add(amount)
	k.SetDepositOrSendToBank(ctx, subaccountID, denom, *deposit, false)
}

// DecrementDeposit decrements a given subaccount's deposits by a given dec amount for a given denom
func (k *Keeper) DecrementDeposit(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if amount.IsZero() {
		return nil
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)

	// usually available balance check is sufficient, but in case of a bug, we check total balance as well
	if deposit.IsEmpty() || deposit.AvailableBalance.LT(amount) || deposit.TotalBalance.LT(amount) {
		metrics.ReportFuncError(k.svcTags)
		return types.ErrInsufficientDeposit
	}
	deposit.AvailableBalance = deposit.AvailableBalance.Sub(amount)
	deposit.TotalBalance = deposit.TotalBalance.Sub(amount)
	k.SetDeposit(ctx, subaccountID, denom, deposit)
	return nil
}

// DecrementDepositOrChargeFromBank decrements a given subaccount's deposits by a given dec amount for a given denom or
// charges the rounded dec amount from the account's bank balance
func (k *Keeper) DecrementDepositOrChargeFromBank(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount sdk.Dec,
) (chargeAmount sdk.Dec, err error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if types.IsDefaultSubaccountID(subaccountID) {
		sender := types.SubaccountIDToSdkAddress(subaccountID)
		// round up decimal portion (if exists) - truncation is fine here since we do Ceil first
		intAmount := amount.Ceil().TruncateInt()
		chargeAmount = intAmount.ToDec()
		err = k.chargeBank(ctx, sender, denom, intAmount)
		return chargeAmount, err
	}

	err = k.DecrementDeposit(ctx, subaccountID, denom, amount)
	return amount, err
}
