package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	db "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) unmarshalRedemptionSchedule(bz []byte) *types.RedemptionSchedule {
	if bz == nil {
		return nil
	}

	var schedule types.RedemptionSchedule
	err := schedule.Unmarshal(bz)
	if err != nil {
		panic(err)
	}

	return &schedule
}

// ExportNextRedemptionScheduleId returns next redemption schedule Id
func (k *Keeper) ExportNextRedemptionScheduleId(ctx sdk.Context) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var scheduleId uint64
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GlobalRedemptionScheduleIdPrefixKey)
	if bz == nil {
		scheduleId = 1
	} else {
		scheduleId = sdk.BigEndianToUint64(bz)
	}

	return scheduleId
}

func (k *Keeper) SetNextRedemptionScheduleId(ctx sdk.Context, scheduleId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GlobalRedemptionScheduleIdPrefixKey, sdk.Uint64ToBigEndian(scheduleId))
}

// getNextRedemptionScheduleId returns the next redemption schedule id and increase it
func (k *Keeper) getNextRedemptionScheduleId(ctx sdk.Context) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	scheduleId := k.ExportNextRedemptionScheduleId(ctx)
	k.SetNextRedemptionScheduleId(ctx, scheduleId+1)

	return scheduleId
}

// nolint:all
func (k *Keeper) getRedemptionSchedule(ctx sdk.Context, redemptionID uint64, claimTime time.Time) *types.RedemptionSchedule {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	key := types.GetRedemptionScheduleKey(redemptionID, claimTime)
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(key)

	return k.unmarshalRedemptionSchedule(bz)
}

func (k *Keeper) SetRedemptionSchedule(ctx sdk.Context, schedule types.RedemptionSchedule) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bz, err := schedule.Marshal()
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic(err)
	}

	key := schedule.GetRedemptionScheduleKey()
	store.Set(key, bz)
}

func (k *Keeper) deleteRedemptionSchedule(ctx sdk.Context, schedule types.RedemptionSchedule) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := schedule.GetRedemptionScheduleKey()
	store.Delete(key)
}

func (k *Keeper) globalRedemptionIterator(ctx sdk.Context) db.Iterator {
	store := ctx.KVStore(k.storeKey)
	return storetypes.KVStorePrefixIterator(store, types.RedemptionSchedulePrefixKey)
}

func (k *Keeper) getRedemptionAmountFromShare(ctx sdk.Context, marketID common.Hash, fund types.InsuranceFund, shareAmount math.Int) sdk.Coin {
	marketBalance := k.exchangeKeeper.GetMarketBalance(ctx, marketID)
	fundBalance := fund.Balance.ToLegacyDec()

	if marketBalance.IsNegative() {
		fundBalance = fundBalance.Add(marketBalance)
	}

	if fundBalance.IsNegative() {
		metrics.ReportFuncError(k.svcTags)
		return sdk.NewCoin(fund.DepositDenom, math.ZeroInt())
	}

	// defensive programming, should never happen
	if fund.TotalShare.IsZero() {
		metrics.ReportFuncError(k.svcTags)
		return sdk.NewCoin(fund.DepositDenom, math.ZeroInt())
	}

	// defensive programming, should never happen
	product, err := shareAmount.SafeMul(fundBalance.TruncateInt())
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return sdk.NewCoin(fund.DepositDenom, math.ZeroInt())
	}

	redemptionAmount := product.Quo(fund.TotalShare)
	return sdk.NewCoin(fund.DepositDenom, redemptionAmount)
}

// GetAllInsuranceFundRedemptions is used to export all insurance fund redemption requests
func (k *Keeper) GetAllInsuranceFundRedemptions(ctx sdk.Context) []types.RedemptionSchedule {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	schedules := make([]types.RedemptionSchedule, 0)
	iterator := k.globalRedemptionIterator(ctx)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		schedule := k.unmarshalRedemptionSchedule(iterator.Value())
		if schedule == nil {
			panic("redemption schedule unmarshal failure")
		}

		schedules = append(schedules, *schedule)
	}

	return schedules
}

// IterateInsuranceFunds iterates over InsuranceFunds calling process on each insurance fund.
func (k *Keeper) IterateInsuranceFunds(ctx sdk.Context, process func(*types.InsuranceFund) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	fundStore := prefix.NewStore(store, types.InsuranceFundPrefixKey)

	iterator := fundStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var fund types.InsuranceFund
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &fund)
		if process(&fund) {
			return
		}
	}
}

// HasInsuranceFund returns true if InsuranceFund for the given marketID exists.
func (k *Keeper) HasInsuranceFund(ctx sdk.Context, marketID common.Hash) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	fundStore := prefix.NewStore(store, types.InsuranceFundPrefixKey)
	return fundStore.Has(marketID.Bytes())
}

// GetAllInsuranceFunds returns all of the Insurance Funds.
func (k *Keeper) GetAllInsuranceFunds(ctx sdk.Context) []types.InsuranceFund {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	insuranceFunds := make([]types.InsuranceFund, 0)
	appendPair := func(p *types.InsuranceFund) (stop bool) {
		if p == nil {
			metrics.ReportFuncError(k.svcTags)
			panic("invalid insurance fund exists")
		}

		insuranceFunds = append(insuranceFunds, *p)
		return false
	}

	k.IterateInsuranceFunds(ctx, appendPair)
	return insuranceFunds
}

// GetInsuranceFund returns the insurance fund corresponding to the given marketID.
func (k *Keeper) GetInsuranceFund(ctx sdk.Context, marketID common.Hash) *types.InsuranceFund {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	fundStore := prefix.NewStore(store, types.InsuranceFundPrefixKey)
	bz := fundStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var fund types.InsuranceFund
	k.cdc.MustUnmarshal(bz, &fund)

	return &fund
}

// DepositIntoInsuranceFund increments the insurance fund balance by amount.
func (k *Keeper) DepositIntoInsuranceFund(ctx sdk.Context, marketID common.Hash, amount math.Int) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	fund := k.GetInsuranceFund(ctx, marketID)

	if fund == nil {
		metrics.ReportFuncError(k.svcTags)
		return types.ErrInsuranceFundNotFound
	}

	fund.Balance = fund.Balance.Add(amount)
	k.SetInsuranceFund(ctx, fund)
	return nil
}

// WithdrawFromInsuranceFund decrements the insurance fund balance by amount and sends tokens from the insurance module to the exchange module.
func (k *Keeper) WithdrawFromInsuranceFund(ctx sdk.Context, marketID common.Hash, amount math.Int) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	fund := k.GetInsuranceFund(ctx, marketID)

	if fund == nil {
		metrics.ReportFuncError(k.svcTags)
		return types.ErrInsuranceFundNotFound
	} else if amount.GT(fund.Balance) {
		metrics.ReportFuncError(k.svcTags)
		return types.ErrPayoutTooLarge
	}

	fund.Balance = fund.Balance.Sub(amount)
	k.SetInsuranceFund(ctx, fund)
	coinAmount := sdk.NewCoin(fund.DepositDenom, amount)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventInsuranceWithdraw{
		MarketId:     fund.MarketId,
		MarketTicker: fund.MarketTicker,
		Withdrawal:   coinAmount,
	})
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, exchangetypes.ModuleName, sdk.NewCoins(coinAmount))
}

// SetInsuranceFund set insurance into keeper
func (k *Keeper) SetInsuranceFund(ctx sdk.Context, fund *types.InsuranceFund) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	marketID := common.HexToHash(fund.MarketId)

	fundStore := prefix.NewStore(store, types.InsuranceFundPrefixKey)

	bz := k.cdc.MustMarshal(fund)
	fundStore.Set(marketID.Bytes(), bz)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventInsuranceFundUpdate{Fund: fund})
}

// CreateInsuranceFund create insurance fund and mint pool tokens
func (k *Keeper) CreateInsuranceFund(
	ctx sdk.Context,
	sender sdk.AccAddress,
	deposit sdk.Coin,
	ticker, quoteDenom, oracleBase, oracleQuote string,
	oracleType oracletypes.OracleType,
	expiry int64,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var marketID common.Hash
	isBinaryOptions := expiry == types.BinaryOptionsExpiryFlag
	if isBinaryOptions {
		marketID = exchangetypes.NewBinaryOptionsMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType)
	} else {
		marketID = exchangetypes.NewDerivativesMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType, expiry)
	}

	// check if insurance already exist and return error if exist
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInsuranceFundAlreadyExists, "insurance fund %s already exist", marketID.Hex())
	}

	// create insurance fund object
	shareBaseDenom := types.ShareDenomFromId(k.getNextShareDenomId(ctx))

	// use default RedemptionNoticePeriodDuration from params
	redemptionNoticePeriodDuration := k.GetParams(ctx).DefaultRedemptionNoticePeriodDuration
	if isBinaryOptions {
		redemptionNoticePeriodDuration = types.DefaultBinaryOptionsInsurancePeriod
	}
	fund = types.NewInsuranceFund(marketID, deposit.Denom, shareBaseDenom, redemptionNoticePeriodDuration, ticker, oracleBase, oracleQuote, oracleType, expiry)

	// initial deposit shouldn't be zero always as we mint tokens for the first user that deposits
	if deposit.Amount.Equal(math.ZeroInt()) {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalidDepositAmount, "insurance fund initial deposit should not be zero")
	}

	// send coins to module account
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, sdk.Coins{deposit})
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	// record total supply for share tokens
	fund.Balance = fund.Balance.Add(deposit.Amount)

	// mint the minimum protocol owned liquidity to the insurance module
	if err := k.bankKeeper.MintCoins(
		ctx,
		types.ModuleName,
		sdk.Coins{sdk.NewCoin(fund.ShareDenom(), types.InsuranceFundProtocolOwnedLiquiditySupply)},
	); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	fund.AddTotalShare(types.InsuranceFundProtocolOwnedLiquiditySupply)

	fund, err = k.MintShareTokens(ctx, fund, sender, types.InsuranceFundCreatorSupply)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	k.SetInsuranceFund(ctx, fund)

	// set metadata for share denom
	shareDisplayDenom := fmt.Sprintf("INSURANCE-%s", marketID.String())
	k.bankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: fmt.Sprintf("The share token of the insurance fund %s", marketID.Hex()),
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    shareBaseDenom,
				Exponent: 0,
				Aliases:  nil,
			},
			{
				Denom:    shareDisplayDenom,
				Exponent: 6,
				Aliases:  nil,
			},
		},
		Base:    shareBaseDenom,
		Display: shareDisplayDenom,
		Name:    fmt.Sprintf("%s share token", ticker),
		Symbol:  fmt.Sprintf("INSURANCE-%s", ticker),
	})

	return nil
}

// UnderwriteInsuranceFund deposit into insurance fund and mint share tokens
func (k *Keeper) UnderwriteInsuranceFund(ctx sdk.Context, underwriter sdk.AccAddress, marketID common.Hash, deposit sdk.Coin) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// check if insurance already exist and return error if does not exist
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInsuranceFundNotFound, "insurance fund for %s does not exist", marketID.Hex())
	}

	if deposit.Denom != fund.DepositDenom {
		return types.ErrInvalidDepositDenom
	}

	// send coins to module account
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, underwriter, types.ModuleName, sdk.Coins{deposit})
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	var shareTokenAmount math.Int
	if fund.TotalShare.Equal(types.InsuranceFundProtocolOwnedLiquiditySupply) || fund.Balance.LTE(math.ZeroInt()) {
		// when there is only protocol liquidity share left in the fund,
		// we refresh the fund with a new supply of minted share tokens
		if err := k.refreshInsuranceFund(ctx, marketID, fund); err != nil {
			return err
		}
		shareTokenAmount = types.InsuranceFundCreatorSupply
	} else {
		shareTokenAmount = fund.TotalShare.Mul(deposit.Amount).Quo(fund.Balance)
	}

	// increase fund balance
	fund.Balance = fund.Balance.Add(deposit.Amount)

	fund, err = k.MintShareTokens(ctx, fund, underwriter, shareTokenAmount)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	k.SetInsuranceFund(ctx, fund)
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventUnderwrite{
		Underwriter: underwriter.String(),
		MarketId:    marketID.Hex(),
		Deposit:     deposit,
		Shares:      sdk.NewCoin(fund.ShareDenom(), shareTokenAmount),
	})

	return nil
}

func (k *Keeper) refreshInsuranceFund(
	ctx sdk.Context,
	marketID common.Hash,
	fund *types.InsuranceFund,
) error {
	// we change shared denom for insurance fund to start fresh insurance
	nextShareDenomID := k.getNextShareDenomId(ctx)
	fund.InsurancePoolTokenDenom = types.ShareDenomFromId(nextShareDenomID)
	fund.TotalShare = types.InsuranceFundProtocolOwnedLiquiditySupply

	if err := k.bankKeeper.MintCoins(
		ctx,
		types.ModuleName,
		sdk.NewCoins(sdk.NewCoin(fund.ShareDenom(), types.InsuranceFundProtocolOwnedLiquiditySupply)),
	); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	shareDisplayDenom := fmt.Sprintf("INSURANCE-%s", marketID.String())
	k.bankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: fmt.Sprintf("The share token of the insurance fund %s", marketID.Hex()),
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    fund.InsurancePoolTokenDenom,
				Exponent: 0,
				Aliases:  nil,
			},
			{
				Denom:    shareDisplayDenom,
				Exponent: 6,
				Aliases:  nil,
			},
		},
		Base:    fund.InsurancePoolTokenDenom,
		Display: shareDisplayDenom,
		Name:    fmt.Sprintf("%s share token", fund.MarketTicker),
		Symbol:  fmt.Sprintf("INSURANCE-%s", fund.MarketTicker),
	})

	return nil
}

func (k *Keeper) GetEstimatedRedemptions(ctx sdk.Context, sender sdk.AccAddress, marketID common.Hash) sdk.Coins {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// check if insurance already exist
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		return sdk.Coins{}
	}

	shareBaseDenom := fund.ShareDenom()
	shareAmount := k.bankKeeper.GetBalance(ctx, sender, shareBaseDenom)
	redemptionCoin := k.getRedemptionAmountFromShare(ctx, marketID, *fund, shareAmount.Amount)

	return sdk.Coins{redemptionCoin}
}

func (k *Keeper) GetPendingRedemptions(ctx sdk.Context, sender sdk.AccAddress, marketID common.Hash) sdk.Coins {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// check if insurance already exist
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		return sdk.Coins{}
	}

	// iterate all redemptions and sum up pending redemptions
	redemptions := sdk.Coins{}
	iterator := k.globalRedemptionIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		schedule := k.unmarshalRedemptionSchedule(iterator.Value())
		if schedule.MarketId == marketID.String() && schedule.Redeemer == sender.String() {
			shareAmount := schedule.RedemptionAmount.Amount
			redemptions = redemptions.Add(k.getRedemptionAmountFromShare(ctx, marketID, *fund, shareAmount))
		}
	}

	return redemptions
}

// RequestInsuranceFundRedemption withdraw deposit token from insurance fund and burn share tokens
func (k *Keeper) RequestInsuranceFundRedemption(ctx sdk.Context, sender sdk.AccAddress, marketID common.Hash, shares sdk.Coin) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// check if insurance already exist
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInsuranceFundNotFound, "insurance fund %s not found", marketID)
	}

	if shares.Denom != fund.ShareDenom() {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalidShareDenom, "insurance fund share denom %s doesnt match redemption share denom %s", fund.ShareDenom(), shares.Denom)
	}

	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, sdk.Coins{shares})
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	nextScheduleId := k.getNextRedemptionScheduleId(ctx)
	claimTime := ctx.BlockTime().Add(fund.RedemptionNoticePeriodDuration)

	schedule := &types.RedemptionSchedule{
		Id:                      nextScheduleId,
		MarketId:                marketID.Hex(),
		Redeemer:                sender.String(),
		ClaimableRedemptionTime: claimTime,
		RedemptionAmount:        shares,
	}

	k.SetRedemptionSchedule(ctx, *schedule)
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventRequestRedemption{Schedule: schedule})

	if k.isMarketDemolishedOrExpired(ctx, marketID) {
		if err := k.withdrawRedemption(ctx, schedule); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return err
		}
	}

	return nil
}

// WithdrawAllMaturedRedemptions it will be used for automatic withdraw on abci
func (k *Keeper) WithdrawAllMaturedRedemptions(ctx sdk.Context) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// iterate all redemptions and do withdraw
	iterator := k.globalRedemptionIterator(ctx)
	defer iterator.Close()

	// caches result of k.isMarketDemolishedOrExpired
	isMarketDemolishedOrExpiredCache := map[string]bool{}

	for ; iterator.Valid(); iterator.Next() {
		schedule := k.unmarshalRedemptionSchedule(iterator.Value())

		isMarketDemolishedOrExpired, ok := isMarketDemolishedOrExpiredCache[schedule.MarketId]
		if !ok {
			isMarketDemolishedOrExpired = k.isMarketDemolishedOrExpired(ctx, common.HexToHash(schedule.MarketId))
			isMarketDemolishedOrExpiredCache[schedule.MarketId] = isMarketDemolishedOrExpired
		}

		shouldWithdraw := isMarketDemolishedOrExpired || ctx.BlockTime().After(schedule.ClaimableRedemptionTime)
		if shouldWithdraw {
			// use cacheCtx so one redemption failing does not block others
			cacheCtx, writeCache := ctx.CacheContext()
			if err := k.withdrawRedemption(cacheCtx, schedule); err != nil {
				metrics.ReportFuncError(k.svcTags)
				k.Logger(ctx).Error("failed to withdraw redemption", err)
				continue
			}
			writeCache()
		}
	}

	return nil
}

// withdrawRedemption executes withdrawal of the specified redemption schedule
func (k *Keeper) withdrawRedemption(ctx sdk.Context, schedule *types.RedemptionSchedule) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// check if insurance exists
	marketID := common.HexToHash(schedule.MarketId)
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		metrics.ReportFuncError(k.svcTags)
		// Note: insurance fund is never deleted and it should exist if it's put on redemption schedule
		return errors.Wrapf(types.ErrInsuranceFundNotFound, "insurance fund %s does not exist", marketID.Hex())
	}
	// convert string address to bytes
	redeemer, err := sdk.AccAddressFromBech32(schedule.Redeemer)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	// delete schedule
	k.deleteRedemptionSchedule(ctx, *schedule)

	// if redemption share doesn't match the fund's current share denom, burn the shares
	if fund.ShareDenom() != schedule.RedemptionAmount.Denom {
		err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(schedule.RedemptionAmount))
		if err != nil {
			// Note: error can happen when redemption amount is invalid coin or module does not have enough balance
			metrics.ReportFuncError(k.svcTags)
		}
		return nil
	}

	// send deposit tokens to redeemer - this should come before burn for correct calculation
	shareAmount := schedule.RedemptionAmount.Amount

	redeemCoin := k.getRedemptionAmountFromShare(ctx, marketID, *fund, shareAmount)
	if redeemCoin.Amount.IsPositive() {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, redeemer, sdk.Coins{redeemCoin})
		if err != nil {
			// Note: error can happen when redeemCoin is invalid coin or module does not have enough balance
			metrics.ReportFuncError(k.svcTags)
			return nil
		}
	}

	// burn share tokens locked on module
	fund, err = k.BurnShareTokens(ctx, fund, shareAmount)
	if err != nil {
		// Note: error can happen when shareAmount is too big or module does not have enough balance
		metrics.ReportFuncError(k.svcTags)
		return nil
	}

	// record total balance
	fund.Balance = fund.Balance.Sub(redeemCoin.Amount)

	k.SetInsuranceFund(ctx, fund)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventWithdrawRedemption{
		Schedule:   schedule,
		RedeemCoin: redeemCoin,
	})
	return nil
}

// UpdateInsuranceFundOracleParams updates the insurance fund's oracle parameters
func (k *Keeper) UpdateInsuranceFundOracleParams(
	ctx sdk.Context,
	marketID common.Hash,
	oracleParams *exchangetypes.OracleParams,
) error {
	// check if insurance already exists and return error if it doesn't
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrInsuranceFundNotFound, marketID.Hex())
	}
	fund.OracleType = oracleParams.OracleType
	fund.OracleBase = oracleParams.OracleBase
	fund.OracleQuote = oracleParams.OracleQuote
	k.SetInsuranceFund(ctx, fund)
	return nil
}

// isMarketDemolishedOrExpired returns whether the market is demolished or expired.
func (k *Keeper) isMarketDemolishedOrExpired(ctx sdk.Context, marketID common.Hash) bool {
	if market := k.exchangeKeeper.GetDerivativeMarketByID(ctx, marketID); market != nil {
		return isDemolishedOrExpiredMarketStatus(market.Status)
	} else if market := k.exchangeKeeper.GetBinaryOptionsMarketByID(ctx, marketID); market != nil {
		return isDemolishedOrExpiredMarketStatus(market.Status)
	} else if market := k.exchangeKeeper.GetSpotMarketByID(ctx, marketID); market != nil {
		return isDemolishedOrExpiredMarketStatus(market.Status)
	}
	return false
}

func isDemolishedOrExpiredMarketStatus(status exchangetypes.MarketStatus) bool {
	return status == exchangetypes.MarketStatus_Demolished || status == exchangetypes.MarketStatus_Expired
}
