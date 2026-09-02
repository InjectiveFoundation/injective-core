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

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2types "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// Added as additional gas consumption to the end to account for EndBlock processing
const MsgRequestRedemptionGasIncrement = storetypes.Gas(100_000)

const (
	mainnetChainID           = "injective-1"
	redemptionBatchingWindow = 24 * time.Hour
)

func isBinaryOptionsInsuranceDisabled(ctx sdk.Context, expiry int64) bool {
	return ctx.ChainID() == mainnetChainID && expiry == types.BinaryOptionsExpiryFlag
}

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
	defer k.Meter(ctx).FuncTiming(&ctx, "ExportNextRedemptionScheduleId")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "SetNextRedemptionScheduleId")()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GlobalRedemptionScheduleIdPrefixKey, sdk.Uint64ToBigEndian(scheduleId))
}

// getNextRedemptionScheduleId returns the next redemption schedule id and increase it
func (k *Keeper) getNextRedemptionScheduleId(ctx sdk.Context) uint64 {
	defer k.Meter(ctx).FuncTiming(&ctx, "getNextRedemptionScheduleId")()

	scheduleId := k.ExportNextRedemptionScheduleId(ctx)
	k.SetNextRedemptionScheduleId(ctx, scheduleId+1)

	return scheduleId
}

// nolint:all
func (k *Keeper) getRedemptionSchedule(ctx sdk.Context, redemptionID uint64, claimTime time.Time) *types.RedemptionSchedule {
	defer k.Meter(ctx).FuncTiming(&ctx, "getRedemptionSchedule")()

	key := types.GetRedemptionScheduleKey(redemptionID, claimTime)
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(key)

	return k.unmarshalRedemptionSchedule(bz)
}

func (k *Keeper) SetRedemptionSchedule(ctx sdk.Context, schedule types.RedemptionSchedule) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetRedemptionSchedule")()

	store := ctx.KVStore(k.storeKey)
	bz, err := schedule.Marshal()
	if err != nil {
		panic(err)
	}

	// primary index: [prefix][claimTime][redemptionID]
	key := schedule.GetRedemptionScheduleKey()
	store.Set(key, bz)

	// secondary index: [prefix][addr][marketID][claimTime][redemptionID] → empty value
	addrKey, err := schedule.GetRedemptionScheduleByAddrKey()
	if err != nil {
		panic(err)
	}
	store.Set(addrKey, []byte{})
}

func (k *Keeper) deleteRedemptionSchedule(ctx sdk.Context, schedule types.RedemptionSchedule) {
	defer k.Meter(ctx).FuncTiming(&ctx, "deleteRedemptionSchedule")()

	store := ctx.KVStore(k.storeKey)
	key := schedule.GetRedemptionScheduleKey()
	store.Delete(key)

	addrKey, err := schedule.GetRedemptionScheduleByAddrKey()
	if err != nil {
		k.Logger(ctx).Error("failed to compute addr index key for redemption schedule deletion", "error", err)
		return
	}
	store.Delete(addrKey)
}

func (k *Keeper) globalRedemptionIterator(ctx sdk.Context) db.Iterator {
	store := ctx.KVStore(k.storeKey)
	return storetypes.KVStorePrefixIterator(store, types.RedemptionSchedulePrefixKey)
}

// ExportNextFailedRedemptionScheduleId returns the next failed redemption schedule id
func (k *Keeper) ExportNextFailedRedemptionScheduleId(ctx sdk.Context) uint64 {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExportNextFailedRedemptionScheduleId")()

	var id uint64
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GlobalFailedRedemptionScheduleIdPrefixKey)
	if bz == nil {
		id = 1
	} else {
		id = sdk.BigEndianToUint64(bz)
	}

	return id
}

func (k *Keeper) SetNextFailedRedemptionScheduleId(ctx sdk.Context, id uint64) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetNextFailedRedemptionScheduleId")()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GlobalFailedRedemptionScheduleIdPrefixKey, sdk.Uint64ToBigEndian(id))
}

// getNextFailedRedemptionScheduleId returns the next failed redemption schedule id and increments it
func (k *Keeper) getNextFailedRedemptionScheduleId(ctx sdk.Context) uint64 {
	id := k.ExportNextFailedRedemptionScheduleId(ctx)
	k.SetNextFailedRedemptionScheduleId(ctx, id+1)
	return id
}

// GetFailedRedemptionSchedule returns the failed redemption schedule for the given id.
func (k *Keeper) GetFailedRedemptionSchedule(ctx sdk.Context, id uint64) *types.FailedRedemptionSchedule {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetFailedRedemptionSchedule")()

	store := ctx.KVStore(k.storeKey)
	key := types.GetFailedRedemptionScheduleKey(id)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var failed types.FailedRedemptionSchedule
	if err := failed.Unmarshal(bz); err != nil {
		panic(err)
	}

	return &failed
}

// SetFailedRedemptionSchedule persists a failed redemption schedule to the store.
func (k *Keeper) SetFailedRedemptionSchedule(ctx sdk.Context, failed types.FailedRedemptionSchedule) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetFailedRedemptionSchedule")()

	store := ctx.KVStore(k.storeKey)
	bz, err := failed.Marshal()
	if err != nil {
		panic(err)
	}

	key := failed.GetFailedRedemptionScheduleKey()
	store.Set(key, bz)
}

// GetAllFailedRedemptionSchedules returns all failed redemption schedules.
func (k *Keeper) GetAllFailedRedemptionSchedules(ctx sdk.Context) []types.FailedRedemptionSchedule {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllFailedRedemptionSchedules")()

	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.FailedRedemptionSchedulePrefixKey)
	defer iterator.Close()

	results := make([]types.FailedRedemptionSchedule, 0)
	for ; iterator.Valid(); iterator.Next() {
		var failed types.FailedRedemptionSchedule
		if err := failed.Unmarshal(iterator.Value()); err != nil {
			panic(err)
		}
		results = append(results, failed)
	}

	return results
}

func (k *Keeper) getRedemptionAmountFromShare(ctx sdk.Context, marketID common.Hash, fund types.InsuranceFund, shareAmount math.Int) sdk.Coin {
	defer k.Meter(ctx).FuncTiming(&ctx, "getRedemptionAmountFromShare")()

	marketBalance := k.exchangeKeeper.GetMarketBalance(ctx, marketID)
	fundBalance := fund.Balance.ToLegacyDec()

	if marketBalance.IsNegative() {
		fundBalance = fundBalance.Add(marketBalance)
	}

	if fundBalance.IsNegative() {
		return sdk.NewCoin(fund.DepositDenom, math.ZeroInt())
	}

	// defensive programming, should never happen
	if fund.TotalShare.IsZero() {
		return sdk.NewCoin(fund.DepositDenom, math.ZeroInt())
	}

	// defensive programming, should never happen
	product, err := shareAmount.SafeMul(fundBalance.TruncateInt())
	if err != nil {
		return sdk.NewCoin(fund.DepositDenom, math.ZeroInt())
	}

	redemptionAmount := product.Quo(fund.TotalShare)
	return sdk.NewCoin(fund.DepositDenom, redemptionAmount)
}

// GetAllInsuranceFundRedemptions is used to export all insurance fund redemption requests
func (k *Keeper) GetAllInsuranceFundRedemptions(ctx sdk.Context) []types.RedemptionSchedule {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllInsuranceFundRedemptions")()

	schedules := make([]types.RedemptionSchedule, 0)

	chaintypes.IterateSafe(k.globalRedemptionIterator(ctx), func(_, value []byte) bool {
		schedule := k.unmarshalRedemptionSchedule(value)
		if schedule == nil {
			panic("redemption schedule unmarshal failure")
		}
		schedules = append(schedules, *schedule)
		return false
	})

	return schedules
}

// IterateInsuranceFunds iterates over InsuranceFunds calling process on each insurance fund.
func (k *Keeper) IterateInsuranceFunds(ctx sdk.Context, process func(*types.InsuranceFund) (stop bool)) {
	defer k.Meter(ctx).FuncTiming(&ctx, "IterateInsuranceFunds")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "HasInsuranceFund")()

	store := ctx.KVStore(k.storeKey)
	fundStore := prefix.NewStore(store, types.InsuranceFundPrefixKey)
	return fundStore.Has(marketID.Bytes())
}

// GetAllInsuranceFunds returns all of the Insurance Funds.
func (k *Keeper) GetAllInsuranceFunds(ctx sdk.Context) []types.InsuranceFund {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllInsuranceFunds")()

	insuranceFunds := make([]types.InsuranceFund, 0)
	appendPair := func(p *types.InsuranceFund) (stop bool) {
		if p == nil {
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
	defer k.Meter(ctx).FuncTiming(&ctx, "GetInsuranceFund")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "DepositIntoInsuranceFund")()

	fund := k.GetInsuranceFund(ctx, marketID)

	if fund == nil {
		return types.ErrInsuranceFundNotFound
	}

	fund.Balance = fund.Balance.Add(amount)
	k.SetInsuranceFund(ctx, fund)
	return nil
}

// WithdrawFromInsuranceFund decrements the insurance fund balance by amount and sends tokens from the insurance module to the exchange module.
func (k *Keeper) WithdrawFromInsuranceFund(ctx sdk.Context, marketID common.Hash, amount math.Int) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "WithdrawFromInsuranceFund")()

	fund := k.GetInsuranceFund(ctx, marketID)

	if fund == nil {
		return types.ErrInsuranceFundNotFound
	} else if amount.GT(fund.Balance) {
		return types.ErrPayoutTooLarge
	}

	coinAmount := sdk.NewCoin(fund.DepositDenom, amount)
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, exchangetypes.ModuleName, sdk.NewCoins(coinAmount)); err != nil {
		return err
	}

	fund.Balance = fund.Balance.Sub(amount)
	k.SetInsuranceFund(ctx, fund)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventInsuranceWithdraw{
		MarketId:     fund.MarketId,
		MarketTicker: fund.MarketTicker,
		Withdrawal:   coinAmount,
	})
	return nil
}

// SetInsuranceFund set insurance into keeper
func (k *Keeper) SetInsuranceFund(ctx sdk.Context, fund *types.InsuranceFund) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetInsuranceFund")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "CreateInsuranceFund")()

	if isBinaryOptionsInsuranceDisabled(ctx, expiry) {
		return errors.Wrap(exchangetypes.ErrFeatureDisabled, "binary options insurance funds are disabled")
	}

	if deposit.Denom != quoteDenom {
		return errors.Wrapf(
			types.ErrInvalidDepositDenom,
			"quote denom %s does not match deposit denom %s",
			quoteDenom,
			deposit.Denom,
		)
	}

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
		return errors.Wrapf(types.ErrInvalidDepositAmount, "insurance fund initial deposit should not be zero")
	}

	// send coins to module account
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, sdk.Coins{deposit})
	if err != nil {
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
		return err
	}

	fund.AddTotalShare(types.InsuranceFundProtocolOwnedLiquiditySupply)

	fund, err = k.MintShareTokens(ctx, fund, sender, types.InsuranceFundCreatorSupply)
	if err != nil {
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
	defer k.Meter(ctx).FuncTiming(&ctx, "UnderwriteInsuranceFund")()

	// check if insurance already exist and return error if does not exist
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		return errors.Wrapf(types.ErrInsuranceFundNotFound, "insurance fund for %s does not exist", marketID.Hex())
	}

	if isBinaryOptionsInsuranceDisabled(ctx, fund.Expiry) {
		return errors.Wrap(exchangetypes.ErrFeatureDisabled, "binary options insurance fund underwriting is disabled")
	}

	if deposit.Denom != fund.DepositDenom {
		return types.ErrInvalidDepositDenom
	}

	// send coins to module account
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, underwriter, types.ModuleName, sdk.Coins{deposit})
	if err != nil {
		return err
	}

	var shareTokenAmount math.Int
	if fund.Balance.LTE(math.ZeroInt()) {
		// refresh the fund only after all backing balance has been depleted;
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
	defer k.Meter(ctx).FuncTiming(&ctx, "refreshInsuranceFund")()

	// we change shared denom for insurance fund to start fresh insurance
	nextShareDenomID := k.getNextShareDenomId(ctx)
	fund.InsurancePoolTokenDenom = types.ShareDenomFromId(nextShareDenomID)
	fund.TotalShare = types.InsuranceFundProtocolOwnedLiquiditySupply

	if err := k.bankKeeper.MintCoins(
		ctx,
		types.ModuleName,
		sdk.NewCoins(sdk.NewCoin(fund.ShareDenom(), types.InsuranceFundProtocolOwnedLiquiditySupply)),
	); err != nil {
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
	defer k.Meter(ctx).FuncTiming(&ctx, "GetEstimatedRedemptions")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "GetPendingRedemptions")()

	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		return sdk.Coins{}
	}

	redemptions := sdk.Coins{}
	for _, schedule := range k.GetRedemptionSchedulesByAddrAndMarket(ctx, sender, marketID.Hex()) {
		shareAmount := schedule.RedemptionAmount.Amount
		redemptions = redemptions.Add(k.getRedemptionAmountFromShare(ctx, marketID, *fund, shareAmount))
	}

	return redemptions
}

// GetRedemptionSchedulesByAddrAndMarket returns all pending redemption schedules for a
// given redeemer address and market ID, using the secondary address index.
func (k *Keeper) GetRedemptionSchedulesByAddrAndMarket(ctx sdk.Context, sender sdk.AccAddress, marketID string) []types.RedemptionSchedule {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetRedemptionSchedulesByAddrAndMarket")()

	store := ctx.KVStore(k.storeKey)
	idxPrefix := types.GetRedemptionScheduleByAddrPrefix(sender, marketID)

	var schedules []types.RedemptionSchedule

	chaintypes.IterateSafe(storetypes.KVStorePrefixIterator(store, idxPrefix), func(key, _ []byte) bool {
		schedule := k.resolveScheduleFromAddrIndexKey(ctx, idxPrefix, key)
		if schedule != nil {
			schedules = append(schedules, *schedule)
		}
		return false
	})

	return schedules
}

// resolveScheduleFromAddrIndexKey parses a secondary index key to extract the
// claimTime and redemptionID, then looks up the full schedule from the primary index.
func (k *Keeper) resolveScheduleFromAddrIndexKey(ctx sdk.Context, idxPrefix, key []byte) *types.RedemptionSchedule {
	defer k.Meter(ctx).FuncTiming(&ctx, "resolveScheduleFromAddrIndexKey")()

	suffix := key[len(idxPrefix):]
	// suffix = FormatTimeBytes(claimTime) ++ Uint64ToBigEndian(redemptionID)
	if len(suffix) < 8 {
		return nil
	}
	redemptionIDBytes := suffix[len(suffix)-8:]
	redemptionID := sdk.BigEndianToUint64(redemptionIDBytes)
	claimTimeBytes := suffix[:len(suffix)-8]
	claimTime, err := sdk.ParseTimeBytes(claimTimeBytes)
	if err != nil {
		return nil
	}

	return k.getRedemptionSchedule(ctx, redemptionID, claimTime)
}

// findRecentRedemptionSchedule finds an existing pending redemption schedule for the
// given sender and market whose claimable time is within 24h of newClaimTime.
// Uses the secondary (sender, marketID) index so only that address's schedules are scanned.
func (k *Keeper) findRecentRedemptionSchedule(ctx sdk.Context, sender sdk.AccAddress, marketID string, newClaimTime time.Time) *types.RedemptionSchedule {
	defer k.Meter(ctx).FuncTiming(&ctx, "findRecentRedemptionSchedule")()

	store := ctx.KVStore(k.storeKey)
	idxPrefix := types.GetRedemptionScheduleByAddrPrefix(sender, marketID)

	var result *types.RedemptionSchedule

	chaintypes.IterateSafe(storetypes.KVStorePrefixIterator(store, idxPrefix), func(key, _ []byte) bool {
		schedule := k.resolveScheduleFromAddrIndexKey(ctx, idxPrefix, key)
		if schedule == nil {
			return false
		}

		timeDiff := newClaimTime.Sub(schedule.ClaimableRedemptionTime)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff <= redemptionBatchingWindow {
			result = schedule
			return true // stop iteration
		}

		return false
	})

	return result
}

// RequestInsuranceFundRedemption withdraw deposit token from insurance fund and burn share tokens.
// Redemption requests from the same sender for the same market are batched into 24h daily buckets:
// if there is an existing pending redemption created within the last 24 hours, the new request is
// merged with it and the redemption timer is reset.
func (k *Keeper) RequestInsuranceFundRedemption(ctx sdk.Context, sender sdk.AccAddress, marketID common.Hash, shares sdk.Coin) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "RequestInsuranceFundRedemption")()

	ctx.GasMeter().ConsumeGas(MsgRequestRedemptionGasIncrement, "insurance fund redemption EndBlocker processing")

	// check if insurance already exist
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		return errors.Wrapf(types.ErrInsuranceFundNotFound, "insurance fund %s not found", marketID)
	}

	if shares.Denom != fund.ShareDenom() {
		return errors.Wrapf(types.ErrInvalidShareDenom, "insurance fund share denom %s doesnt match redemption share denom %s", fund.ShareDenom(), shares.Denom)
	}

	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, sdk.Coins{shares})
	if err != nil {
		return err
	}

	claimTime := ctx.BlockTime().Add(fund.RedemptionNoticePeriodDuration)

	// Try to find an existing redemption schedule within the 24h batching window
	existingSchedule := k.findRecentRedemptionSchedule(ctx, sender, marketID.Hex(), claimTime)

	var schedule *types.RedemptionSchedule

	canBatch := existingSchedule != nil && existingSchedule.RedemptionAmount.Denom == shares.Denom

	if canBatch {
		k.deleteRedemptionSchedule(ctx, *existingSchedule)

		mergedAmount := existingSchedule.RedemptionAmount.Add(shares)
		schedule = &types.RedemptionSchedule{
			Id:                      existingSchedule.Id,
			MarketId:                marketID.Hex(),
			Redeemer:                sender.String(),
			ClaimableRedemptionTime: claimTime,
			RedemptionAmount:        mergedAmount,
		}
	} else {
		nextScheduleId := k.getNextRedemptionScheduleId(ctx)
		schedule = &types.RedemptionSchedule{
			Id:                      nextScheduleId,
			MarketId:                marketID.Hex(),
			Redeemer:                sender.String(),
			ClaimableRedemptionTime: claimTime,
			RedemptionAmount:        shares,
		}
	}

	k.SetRedemptionSchedule(ctx, *schedule)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventRequestRedemption{Schedule: schedule})

	if k.isMarketDemolishedOrExpired(ctx, marketID) {
		if err := k.processRedemption(ctx, schedule); err != nil {
			return err
		}
	}

	return nil
}

// WithdrawAllMaturedRedemptions it will be used for automatic withdraw on abci
func (k *Keeper) WithdrawAllMaturedRedemptions(ctx sdk.Context) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "WithdrawAllMaturedRedemptions")()

	// caches result of k.isMarketDemolishedOrExpired
	isMarketDemolishedOrExpiredCache := map[string]bool{}

	// first pass: collect all matured schedules
	var maturedSchedules []*types.RedemptionSchedule

	iterator := k.globalRedemptionIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		schedule := k.unmarshalRedemptionSchedule(iterator.Value())

		isMarketDemolishedOrExpired, ok := isMarketDemolishedOrExpiredCache[schedule.MarketId]
		if !ok {
			isMarketDemolishedOrExpired = k.isMarketDemolishedOrExpired(ctx, common.HexToHash(schedule.MarketId))
			isMarketDemolishedOrExpiredCache[schedule.MarketId] = isMarketDemolishedOrExpired
		}

		if isMarketDemolishedOrExpired || ctx.BlockTime().After(schedule.ClaimableRedemptionTime) {
			maturedSchedules = append(maturedSchedules, schedule)
		}
	}

	// second pass: process collected schedules (mutates the store).
	// This cannot be done inside the iterator loop because processRedemption
	// deletes the redemption schedule from the store, which is unsafe during
	// iteration.
	for _, schedule := range maturedSchedules {
		if err := k.processRedemption(ctx, schedule); err != nil {
			k.Logger(ctx).Error("failed to withdraw redemption", err)
		}
	}

	return nil
}

// processRedemption deletes the given redemption schedule and settles the
// withdrawal atomically (burn shares, update fund balance, transfer tokens).
// If the bank transfer fails the coins remain in the module account and a
// voucher is accumulated for the redeemer instead — settlement still
// completes successfully. If any other step fails, all accounting changes are
// reverted and a FailedRedemptionSchedule is persisted to preserve the user's
// ownership claim for future resolution.
func (k *Keeper) processRedemption(ctx sdk.Context, schedule *types.RedemptionSchedule) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processRedemption")(&err)

	// check if insurance exists
	marketID := common.HexToHash(schedule.MarketId)
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		// Note: insurance fund is never deleted and it should exist if it's put on redemption schedule
		err := errors.Wrapf(types.ErrInsuranceFundNotFound, "insurance fund %s does not exist", marketID.Hex())
		return err
	}
	// convert string address to bytes
	redeemer, err := sdk.AccAddressFromBech32(schedule.Redeemer)
	if err != nil {
		return err
	}

	k.deleteRedemptionSchedule(ctx, *schedule)

	err = executeAtomic(ctx, func(ctx sdk.Context) error {
		return k.settleRedemption(ctx, fund, marketID, redeemer, schedule)
	})

	if err != nil {
		emitWithdrawRedemptionFailedEvent(ctx, schedule, err)

		k.Logger(ctx).Error("failed to withdraw redemption", err)

		failedRedemptionSchedule := types.FailedRedemptionSchedule{
			Id:       k.getNextFailedRedemptionScheduleId(ctx),
			Schedule: *schedule,
			Err:      fmt.Sprintf("failed to withdraw redemption: %s", err.Error()),
		}
		k.SetFailedRedemptionSchedule(ctx, failedRedemptionSchedule)

		k.Logger(ctx).Debug("recorded failed redemption schedule", "id", failedRedemptionSchedule.Id)
	}

	return nil
}

// settleRedemption performs all redemption accounting: transferring deposit
// tokens to the redeemer, burning share tokens, and updating the fund balance.
// It must be called within an atomic (cached) context so that a failure at any
// step reverts all prior state changes.
func (k *Keeper) settleRedemption(
	ctx sdk.Context,
	fund *types.InsuranceFund,
	marketID common.Hash,
	redeemer sdk.AccAddress,
	schedule *types.RedemptionSchedule,
) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "settleRedemption")(&err)

	// if redemption share doesn't match the fund's current share denom, burn
	// the shares and return early
	if fund.ShareDenom() != schedule.RedemptionAmount.Denom {
		err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(schedule.RedemptionAmount))
		return err
	}

	shareAmount := schedule.RedemptionAmount.Amount

	// send deposit tokens to redeemer - this should come before burn for correct calculation
	redeemCoin := k.getRedemptionAmountFromShare(ctx, marketID, *fund, shareAmount)
	if redeemCoin.Amount.IsPositive() {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, redeemer, sdk.Coins{redeemCoin}); err != nil {
			// Coins remain in the module account. Record a voucher so the redeemer can
			// claim the amount later. Shares are still burned so fund accounting stays
			// consistent. The accumulation is inside the atomic context, so a subsequent
			// failure rolls it back automatically.
			if vErr := k.vouchersAssistant.AddVoucher(ctx, redeemer, redeemCoin); vErr != nil {
				return vErr
			}
			k.Logger(ctx).Warn("payout rerouted to voucher: bank transfer failed",
				"redeemer", redeemer.String(),
				"coin", redeemCoin.String(),
				"bank_err", err,
			)
		}
	}

	// burn share tokens locked on module
	fund, err = k.BurnShareTokens(ctx, fund, shareAmount)
	if err != nil {
		return err
	}

	// record total balance
	fund.Balance = fund.Balance.Sub(redeemCoin.Amount)

	k.SetInsuranceFund(ctx, fund)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventWithdrawRedemption{
		Schedule:   schedule,
		RedeemCoin: redeemCoin,
	}); err != nil {
		ctx.Logger().Error("failed to emit EventWithdrawRedemption", "schedule_id", schedule.Id, "err", err)
	}

	return nil
}

// executeAtomic executes the given function within a cached (branched) context,
// ensuring atomicity. If f returns an error the changes are discarded and the
// error is returned; otherwise the cached writes and events are committed to
// the parent context via writeCache.
func executeAtomic(ctx sdk.Context, f func(ctx sdk.Context) error) error {
	cacheCtx, writeCache := ctx.CacheContext()
	if err := f(cacheCtx); err != nil {
		return err
	}
	writeCache()
	return nil
}

func emitWithdrawRedemptionFailedEvent(ctx sdk.Context, schedule *types.RedemptionSchedule, err error) {
	// nolint:errcheck // ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventWithdrawRedemptionFailed{
		Schedule:    schedule,
		WithdrawErr: err.Error(),
	})
}

// UpdateInsuranceFundOracleParams updates the insurance fund's oracle parameters
func (k *Keeper) UpdateInsuranceFundOracleParams(
	ctx sdk.Context,
	marketID common.Hash,
	oracleParams *exchangetypes.OracleParams,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "UpdateInsuranceFundOracleParams")()
	// check if insurance already exists and return error if it doesn't
	fund := k.GetInsuranceFund(ctx, marketID)
	if fund == nil {
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
	defer k.Meter(ctx).FuncTiming(&ctx, "isMarketDemolishedOrExpired")()

	if market := k.exchangeKeeper.GetDerivativeMarketByID(ctx, marketID); market != nil {
		return isDemolishedOrExpiredMarketStatus(market.Status)
	} else if market := k.exchangeKeeper.GetBinaryOptionsMarketByID(ctx, marketID); market != nil {
		return isDemolishedOrExpiredMarketStatus(market.Status)
	} else if market := k.exchangeKeeper.GetSpotMarketByID(ctx, marketID); market != nil {
		return isDemolishedOrExpiredMarketStatus(market.Status)
	}
	return false
}

func isDemolishedOrExpiredMarketStatus(status exchangev2types.MarketStatus) bool {
	return status == exchangev2types.MarketStatus_Demolished || status == exchangev2types.MarketStatus_Expired
}
