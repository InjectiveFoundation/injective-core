package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
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
		AvailableBalance: balance.Deposits.AvailableBalance.Sub(balanceChangeAmount.ToLegacyDec()),
		TotalBalance:     balance.Deposits.TotalBalance.Sub(balanceChangeAmount.ToLegacyDec()),
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

	if err := k.IncrementDepositForNonDefaultSubaccount(ctx, subaccountID, msg.Amount.Denom, msg.Amount.Amount.ToLegacyDec()); err != nil {
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
	amount := msg.Amount.Amount.ToLegacyDec()

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
) math.LegacyDec {
	subaccountDeposits := k.GetDeposit(ctx, subaccountID, denom)
	if !types.IsDefaultSubaccountID(subaccountID) {
		return subaccountDeposits.AvailableBalance
	}

	// combine bankBalance + dust from subaccount deposits to get the total spendable funds
	bankBalance := k.bankKeeper.GetBalance(ctx, types.SubaccountIDToSdkAddress(subaccountID), denom)
	return bankBalance.Amount.ToLegacyDec().Add(subaccountDeposits.AvailableBalance)
}

func (k *Keeper) GetAllSpendableFunds(
	ctx sdk.Context,
	subaccountID common.Hash,
) map[string]math.LegacyDec {
	spendableFunds := make(map[string]math.LegacyDec, 0)
	subaccountDeposits := k.GetDeposits(ctx, subaccountID)
	for denom, deposit := range subaccountDeposits {
		spendableFunds[denom] = deposit.AvailableBalance
	}
	if types.IsDefaultSubaccountID(subaccountID) {
		bankBalances := k.bankKeeper.GetAllBalances(ctx, types.SubaccountIDToSdkAddress(subaccountID))
		for _, balance := range bankBalances {
			funds := spendableFunds[balance.Denom].Add(balance.Amount.ToLegacyDec())
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetDepositKey(subaccountID, denom)

	bz := store.Get(key)
	if bz == nil {
		return types.NewDeposit()
	}

	var deposit types.Deposit
	k.cdc.MustUnmarshal(bz, &deposit)

	if deposit.TotalBalance.IsNil() {
		deposit.TotalBalance = math.LegacyZeroDec()
	}

	if deposit.AvailableBalance.IsNil() {
		deposit.AvailableBalance = math.LegacyZeroDec()
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.SetTransientDeposit(ctx, subaccountID, denom, deposit)

	store := k.getStore(ctx)
	key := types.GetDepositKey(subaccountID, denom)

	// prune from store if deposit is empty
	if deposit == nil || deposit.IsEmpty() {
		store.Delete(key)
		return
	}

	bz := k.cdc.MustMarshal(deposit)
	store.Set(key, bz)
}

// HasSufficientFunds returns true if the bank balances ≥ ceil(amount) for default subaccounts or if the availableBalance ≥ amount
// for non-default subaccounts.
func (k *Keeper) HasSufficientFunds(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount math.LegacyDec,
) bool {
	isDefaultSubaccountID := types.IsDefaultSubaccountID(subaccountID)

	if isDefaultSubaccountID {
		bankBalance := k.bankKeeper.GetBalance(ctx, types.SubaccountIDToSdkAddress(subaccountID), denom)
		// take the ceiling since we need to round up to the nearest integer due to bank balances being integers
		return bankBalance.Amount.GTE(amount.Ceil().TruncateInt())
	}

	deposit := k.GetDeposit(ctx, subaccountID, denom)
	// usually available balance check is sufficient, but in case of a bug, we check total balance as well
	return deposit.AvailableBalance.GTE(amount) && deposit.TotalBalance.GTE(amount)
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	amountToSendToBank := deposit.AvailableBalance.TruncateInt()

	// for default subaccounts, if the integer part of the available deposit funds are non-zero, send them to bank
	// otherwise, simply set the deposit to allow for dust accumulation
	shouldSendFundsToBank := amountToSendToBank.IsPositive() && types.IsDefaultSubaccountID(subaccountID)

	if shouldSendFundsToBank {
		// NOTE: AvailableBalance should never be GT TotalBalance, but since in some tests the scenario happened
		// we are adding a check to prevent sending more funds to the bank than the total balance
		truncatedTotalBalance := math.MaxInt(deposit.TotalBalance.TruncateInt(), math.NewInt(0))
		amountToSendToBank := math.MinInt(amountToSendToBank, truncatedTotalBalance)
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName, // exchange module
			types.SubaccountIDToSdkAddress(subaccountID),
			sdk.NewCoins(sdk.NewCoin(denom, amountToSendToBank)),
		)

		deposit.AvailableBalance = deposit.AvailableBalance.Sub(amountToSendToBank.ToLegacyDec())
		deposit.TotalBalance = deposit.TotalBalance.Sub(amountToSendToBank.ToLegacyDec())
	} else {
		shouldChargeFromBank := !isPreventingBankCharge && deposit.AvailableBalance.IsNegative() && types.IsDefaultSubaccountID(subaccountID)

		if shouldChargeFromBank {
			amountToChargeFromBank := amountToSendToBank.Abs()

			if availableBalanceAfterCharge := deposit.AvailableBalance.Add(amountToChargeFromBank.ToLegacyDec()); availableBalanceAfterCharge.IsNegative() {
				amountToChargeFromBank = amountToChargeFromBank.AddRaw(1)
			}

			if err := k.chargeBank(ctx, types.SubaccountIDToSdkAddress(subaccountID), denom, amountToChargeFromBank); err == nil {
				deposit.AvailableBalance = deposit.AvailableBalance.Add(amountToChargeFromBank.ToLegacyDec())
				deposit.TotalBalance = deposit.TotalBalance.Add(amountToChargeFromBank.ToLegacyDec())
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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

func (k *Keeper) WithdrawAllAuctionBalances(ctx sdk.Context) sdk.Coins {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	denomDecimals := k.GetAllDenomDecimals(ctx)
	coinsToSendToAuction := sdk.NewCoins()

	injAuctionSubaccountAmount := math.ZeroInt()
	injSendCap := k.GetInjAuctionMaxCap(ctx)

	// collect balances from auction subaccount
	for idx := range denomDecimals {
		denom := denomDecimals[idx].Denom
		deposit := k.GetDeposit(ctx, types.AuctionSubaccountID, denom)

		if deposit.TotalBalance.IsNil() || deposit.TotalBalance.IsZero() || deposit.TotalBalance.TruncateInt().IsZero() {
			continue
		}

		amount := deposit.TotalBalance.TruncateInt()

		if denom == chaintypes.InjectiveCoin {
			amount = math.MinInt(amount, injSendCap)
			injAuctionSubaccountAmount = injAuctionSubaccountAmount.Add(amount)
		}

		if err := k.DecrementDeposit(ctx, types.AuctionSubaccountID, denom, amount.ToLegacyDec()); err != nil {
			k.Logger(ctx).Error("WithdrawAllAuctionBalances DecrementDeposit fail:", err)
			continue
		}
		coinsToSendToAuction = coinsToSendToAuction.Add(sdk.NewCoin(denom, amount))
	}

	balancesToSendFromAuctionAddress := sdk.NewCoins()

	// transfer balances from auction fee address to exchange module
	for idx := range denomDecimals {
		denom := denomDecimals[idx].Denom
		balance := k.bankKeeper.GetBalance(ctx, types.AuctionFeesAddress, denom)

		if balance.IsNil() || !balance.IsPositive() {
			continue
		}

		amount := balance.Amount
		if balance.Denom == chaintypes.InjectiveCoin {
			if injAuctionSubaccountAmount.GTE(injSendCap) {
				amount = math.ZeroInt()
			} else if amount.Add(injAuctionSubaccountAmount).GT(injSendCap) {
				amount = injSendCap.Sub(injAuctionSubaccountAmount)
			}
		}

		coin := sdk.NewCoin(denom, amount)
		balancesToSendFromAuctionAddress = balancesToSendFromAuctionAddress.Add(coin)
		coinsToSendToAuction = coinsToSendToAuction.Add(coin)
	}

	if len(balancesToSendFromAuctionAddress) > 0 {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, types.AuctionFeesAddress, types.ModuleName, balancesToSendFromAuctionAddress); err != nil {
			k.Logger(ctx).Error(err.Error())
		}
	}

	k.MoveCoinsIntoCurrentAuction(ctx, coinsToSendToAuction)
	return coinsToSendToAuction
}

// GetAllExchangeBalances returns the exchange balances.
func (k *Keeper) GetAllExchangeBalances(
	ctx sdk.Context,
) []types.Balance {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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

func (k *Keeper) chargeBank(ctx sdk.Context, account sdk.AccAddress, denom string, amount math.Int) error {
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

func (k *Keeper) chargeAvailableDeposits(ctx sdk.Context, subaccountID common.Hash, denom string, amount math.LegacyDec) error {
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
	amount math.LegacyDec,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	amount math.LegacyDec,
) error {
	sender := types.SubaccountIDToSdkAddress(subaccountID)
	// round up decimal portion (if exists) - truncation is fine here since we do Ceil first
	intAmount := amount.Ceil().TruncateInt()
	decAmount := intAmount.ToLegacyDec()

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
	amount math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	decAmount := coin.Amount.ToLegacyDec()
	k.IncrementDepositOrSendToBank(ctx, subaccountID, coin.Denom, decAmount)
}

// IncrementDepositForNonDefaultSubaccount increments a given non-default subaccount's deposits by a given dec amount for a given denom
func (k *Keeper) IncrementDepositForNonDefaultSubaccount(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	amount math.LegacyDec,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	amount math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	amount math.LegacyDec,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	amount math.LegacyDec,
) (chargeAmount math.LegacyDec, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if types.IsDefaultSubaccountID(subaccountID) {
		sender := types.SubaccountIDToSdkAddress(subaccountID)
		// round up decimal portion (if exists) - truncation is fine here since we do Ceil first
		intAmount := amount.Ceil().TruncateInt()
		chargeAmount = intAmount.ToLegacyDec()
		err = k.chargeBank(ctx, sender, denom, intAmount)
		return chargeAmount, err
	}

	err = k.DecrementDeposit(ctx, subaccountID, denom, amount)
	return amount, err
}
