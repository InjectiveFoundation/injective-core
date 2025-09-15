package types

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = &Params{}

// Auction params default values
var (
	// DefaultAuctionPeriod represents the number of seconds in 1 week
	DefaultAuctionPeriod int64 = 60 * 60 * 24 * 7
	// DefaultMinNextBidIncrementRate represents default min increment rate 0.25%
	DefaultMinNextBidIncrementRate = math.LegacyNewDecWithPrec(25, 4)
	// DefaultInjBasketMaxCap represents default inj basket max cap
	DefaultInjBasketMaxCap = math.NewIntWithDecimal(10_000, 18)
	// DefaultBiddersWhitelist represents default bidders whitelist (empty = all allowed)
	DefaultBiddersWhitelist = []string{}
)

// Parameter keys
var (
	KeyAuctionPeriod           = []byte("AuctionPeriod")
	KeyMinNextBidIncrementRate = []byte("MinNextBidIncrementRate")
	KeyInjBasketMaxCap         = []byte("InjBasketMaxCap")
	KeyBiddersWhitelist        = []byte("BiddersWhitelist")
)

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	auctionPeriod int64,
	minNextBidIncrementRate math.LegacyDec,
	injBasketMaxCap math.Int,
	biddersWhitelist []string,
) Params {
	return Params{
		AuctionPeriod:           auctionPeriod,
		MinNextBidIncrementRate: minNextBidIncrementRate,
		InjBasketMaxCap:         injBasketMaxCap,
		BiddersWhitelist:        biddersWhitelist,
	}
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyAuctionPeriod, &p.AuctionPeriod, validateAuctionPeriodDuration),
		paramtypes.NewParamSetPair(KeyMinNextBidIncrementRate, &p.MinNextBidIncrementRate, validateMinNextBidIncrementRate),
		paramtypes.NewParamSetPair(KeyInjBasketMaxCap, &p.InjBasketMaxCap, validateInjBasketMaxCap),
		paramtypes.NewParamSetPair(KeyBiddersWhitelist, &p.BiddersWhitelist, validateBiddersWhitelist),
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return Params{
		AuctionPeriod:           DefaultAuctionPeriod,
		MinNextBidIncrementRate: DefaultMinNextBidIncrementRate,
		InjBasketMaxCap:         DefaultInjBasketMaxCap,
		BiddersWhitelist:        DefaultBiddersWhitelist,
	}
}

// Validate performs basic validation on auction parameters.
func (p Params) Validate() error {
	if err := validateAuctionPeriodDuration(p.AuctionPeriod); err != nil {
		return err
	}

	if err := validateMinNextBidIncrementRate(p.MinNextBidIncrementRate); err != nil {
		return err
	}

	if err := validateInjBasketMaxCap(p.InjBasketMaxCap); err != nil {
		return err
	}

	return validateBiddersWhitelist(p.BiddersWhitelist)
}

func validateAuctionPeriodDuration(i interface{}) error {
	v, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("AuctionPeriodDuration must be positive: %d", v)
	}

	return nil
}

func validateMinNextBidIncrementRate(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("MinNextBidIncrementRate cannot be nil")
	}

	if v.Equal(math.LegacyZeroDec()) {
		return fmt.Errorf("MinNextBidIncrementRate must be positive: %s", v.String())
	}

	if v.GT(math.LegacyNewDecWithPrec(2, 1)) { // > 20%
		return fmt.Errorf("MinNextBidIncrementRate must be equal or less than 20 percent: %s", v.String())
	}

	return nil
}

func validateInjBasketMaxCap(i interface{}) error {
	v, ok := i.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("InjBasketMaxCap cannot be nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("InjBasketMaxCap cannot be negative")
	}
	return nil
}

func validateBiddersWhitelist(i any) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// Check for duplicate addresses and validate address format
	seen := make(map[string]bool)
	for _, addr := range v {
		if addr == "" {
			return errors.New("empty address in bidders whitelist")
		}

		// Validate address format
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return fmt.Errorf("invalid address format in bidders whitelist: %s", addr)
		}

		if seen[addr] {
			return fmt.Errorf("duplicate address in bidders whitelist: %s", addr)
		}
		seen[addr] = true
	}

	return nil
}
