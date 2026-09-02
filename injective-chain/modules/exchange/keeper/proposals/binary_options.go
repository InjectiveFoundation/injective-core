package proposals

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

const mainnetChainID = "injective-1"

func (k *ProposalKeeper) HandleBinaryOptionsMarketLaunchProposal(
	ctx sdk.Context,
	p *v2.BinaryOptionsMarketLaunchProposal,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.HandleBinaryOptionsMarketLaunchProposal")()

	if ctx.ChainID() == mainnetChainID {
		return errors.Wrap(types.ErrFeatureDisabled, "binary options are disabled")
	}

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	_, err := k.BinaryOptionsMarketLaunch(
		ctx,
		p.Ticker,
		p.OracleSymbol,
		p.OracleProvider,
		p.OracleType,
		p.OracleScaleFactor,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.ExpirationTimestamp,
		p.SettlementTimestamp,
		p.Admin,
		p.QuoteDenom,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
		p.OpenNotionalCap,
	)
	if err != nil {
		return err
	}

	return nil
}

func (k *ProposalKeeper) HandleBinaryOptionsMarketParamUpdateProposal(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.HandleBinaryOptionsMarketParamUpdateProposal")()

	if ctx.ChainID() == mainnetChainID {
		return errors.Wrap(types.ErrFeatureDisabled, "binary options are disabled")
	}

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	market, err := k.validateMarketExists(ctx, p.MarketId)
	if err != nil {
		return err
	}

	_, _, err = validateTimestamps(ctx, market, p)
	if err != nil {
		return err
	}

	if err := k.validateAdmin(ctx, p); err != nil {
		return err
	}

	if err := k.validateOracleParams(ctx, p); err != nil {
		return err
	}

	if p.HasDisabledMinimalProtocolFee != v2.DisableMinimalProtocolFeeUpdate_NoUpdate {
		market.HasDisabledMinimalProtocolFee = p.HasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_True
	}

	if err := k.validateFeeRates(ctx, market, p); err != nil {
		return err
	}

	if p.Ticker == "" {
		p.Ticker = market.Ticker
	}

	// schedule market param change in transient store
	return k.ScheduleBinaryOptionsMarketParamUpdate(ctx, p)
}

func (k *ProposalKeeper) validateMarketExists(ctx sdk.Context, marketId string) (*v2.BinaryOptionsMarket, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.validateMarketExists")()

	marketID := common.HexToHash(marketId)
	market, _ := k.GetBinaryOptionsMarketAndStatus(ctx, marketID)

	if market == nil {
		return nil, types.ErrBinaryOptionsMarketNotFound
	}

	if market.Status == v2.MarketStatus_Demolished {
		return nil, types.ErrInvalidMarketStatus
	}

	return market, nil
}

func validateTimestamps(
	ctx sdk.Context,
	market *v2.BinaryOptionsMarket,
	p *v2.BinaryOptionsMarketParamUpdateProposal,
) (expirationTimestamp, settlementTimestamp int64, err error) {
	expTimestamp, settlementTimestamp := market.ExpirationTimestamp, market.SettlementTimestamp
	currentTime := ctx.BlockTime().Unix()

	// Handle expiration timestamp update
	if p.ExpirationTimestamp != 0 {
		// Check if market is already expired
		if market.ExpirationTimestamp <= currentTime {
			return 0, 0, errors.Wrap(types.ErrInvalidExpiry, "cannot change expiration time of an expired market")
		}

		// Check if new expiration is in the past
		if p.ExpirationTimestamp <= currentTime {
			return 0, 0, errors.Wrapf(types.ErrInvalidExpiry, "expiration timestamp %d is in the past", p.ExpirationTimestamp)
		}

		expTimestamp = p.ExpirationTimestamp
	}

	// Handle settlement timestamp update
	if p.SettlementTimestamp != 0 {
		// Check if new settlement is in the past
		if p.SettlementTimestamp <= currentTime {
			return 0, 0, errors.Wrapf(types.ErrInvalidSettlement, "expiration timestamp %d is in the past", p.SettlementTimestamp)
		}

		settlementTimestamp = p.SettlementTimestamp
	}

	// Validate relationship between timestamps
	if expTimestamp >= settlementTimestamp {
		return 0, 0, errors.Wrap(types.ErrInvalidExpiry, "expiration timestamp should be prior to settlement timestamp")
	}

	return expTimestamp, settlementTimestamp, nil
}

func (k *ProposalKeeper) validateAdmin(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.validateAdmin")()
	// Enforce that the admin account exists, if specified
	if p.Admin != "" {
		admin, _ := sdk.AccAddressFromBech32(p.Admin)
		if !k.account.HasAccount(ctx, admin) {
			return errors.Wrapf(types.ErrAccountDoesntExist, "admin %s", p.Admin)
		}
	}

	return nil
}

func (k *ProposalKeeper) validateOracleParams(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.validateOracleParams")()

	if p.OracleParams != nil {
		// Enforce that the provider exists, but not necessarily that the oracle price for the symbol exists
		if k.oracle.GetProviderInfo(ctx, p.OracleParams.Provider) == nil {
			return errors.Wrapf(types.ErrInvalidOracle, "oracle provider %s does not exist", p.OracleParams.Provider)
		}
	}

	return nil
}

func (k *ProposalKeeper) validateFeeRates(
	ctx sdk.Context,
	market *v2.BinaryOptionsMarket,
	p *v2.BinaryOptionsMarketParamUpdateProposal,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.validateFeeRates")()
	// Skip validation if no fee rates are being updated
	if p.MakerFeeRate == nil && p.TakerFeeRate == nil && p.RelayerFeeShareRate == nil && p.HasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_NoUpdate {
		return nil
	}

	// Use default values from market if not provided in proposal
	makerFeeRate := market.MakerFeeRate
	takerFeeRate := market.TakerFeeRate
	relayerFeeShareRate := market.RelayerFeeShareRate

	// Override with provided values
	if p.MakerFeeRate != nil {
		makerFeeRate = *p.MakerFeeRate
	}

	if p.TakerFeeRate != nil {
		takerFeeRate = *p.TakerFeeRate
	}

	if p.RelayerFeeShareRate != nil {
		relayerFeeShareRate = *p.RelayerFeeShareRate
	}

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx, market)

	if p.HasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_True {
		minimalProtocolFeeRate = math.LegacyZeroDec()
	}

	// must use `if` not `else` here due to `DisableMinimalProtocolFeeUpdate_NoUpdate`
	if p.HasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_False {
		minimalProtocolFeeRate = k.GetCachedParams(ctx).MinimalProtocolFeeRate
	}

	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	return v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate,
		takerFeeRate,
		relayerFeeShareRate,
		minimalProtocolFeeRate,
		discountSchedule,
	)
}
