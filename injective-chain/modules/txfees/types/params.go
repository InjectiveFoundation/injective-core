package types

import (
	"errors"

	"cosmossdk.io/math"
	mempool1559 "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper/mempool-1559"
)

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return Params{
		MaxGasWantedPerTx:                    uint64(30_000_000),
		HighGasTxThreshold:                   uint64(25_000_000),
		MinGasPriceForHighGasTx:              math.LegacyZeroDec(),
		Mempool1559Enabled:                   false,
		MinGasPrice:                          math.LegacyNewDec(mempool1559.DefaultMinGasPrice),
		DefaultBaseFeeMultiplier:             math.LegacyMustNewDecFromStr("1.5"),
		MaxBaseFeeMultiplier:                 math.LegacyMustNewDecFromStr("1000"),
		ResetInterval:                        72_000,
		TargetBlockSpacePercentRate:          math.LegacyMustNewDecFromStr("0.625"),
		RecheckFeeLowBaseFee:                 math.LegacyMustNewDecFromStr("3"),
		RecheckFeeHighBaseFee:                math.LegacyMustNewDecFromStr("2.3"),
		RecheckFeeBaseFeeThresholdMultiplier: math.LegacyMustNewDecFromStr("4"),
		MaxBlockChangeRate:                   math.LegacyMustNewDecFromStr("0.1"),
	}
}

func (p Params) Validate() error {
	if err := p.validateGasParameters(); err != nil {
		return err
	}

	if err := p.validateGasPriceParameters(); err != nil {
		return err
	}

	if err := p.validateMultipliers(); err != nil {
		return err
	}

	return p.validateExecutionTimeParameters()
}

func (p Params) validateGasParameters() error {
	if p.MaxGasWantedPerTx == 0 {
		return errors.New("max_gas_wanted_per_tx must be greater than 0")
	}

	if p.HighGasTxThreshold == 0 {
		return errors.New("high_gas_tx_threshold must be greater than 0")
	}

	if p.HighGasTxThreshold >= p.MaxGasWantedPerTx {
		return errors.New("high_gas_tx_threshold must be less than max_gas_wanted_per_tx")
	}

	return nil
}

func (p Params) validateGasPriceParameters() error {
	if p.MinGasPriceForHighGasTx.IsNegative() {
		return errors.New("min_gas_price_for_high_gas_tx must be positive")
	}

	if p.MinGasPrice.IsNegative() || p.MinGasPrice.IsZero() {
		return errors.New("min_gas_price must be greater than 0")
	}

	return nil
}

func (p Params) validateMultipliers() error {
	if p.DefaultBaseFeeMultiplier.IsNegative() || p.DefaultBaseFeeMultiplier.IsZero() {
		return errors.New("default_base_fee_multiplier must be greater than 0")
	}

	if p.DefaultBaseFeeMultiplier.LT(math.LegacyOneDec()) {
		return errors.New("default_base_fee_multiplier must be greater than or equal to 1")
	}

	if p.MaxBaseFeeMultiplier.LT(p.DefaultBaseFeeMultiplier) {
		return errors.New("max_base_fee_multiplier must be greater than or equal to default_base_fee_multiplier")
	}

	if p.RecheckFeeBaseFeeThresholdMultiplier.IsNegative() || p.RecheckFeeBaseFeeThresholdMultiplier.IsZero() {
		return errors.New("recheck_fee_base_fee_threshold_multiplier must be greater than 0")
	}

	return nil
}

func (p Params) validateExecutionTimeParameters() error {
	if p.ResetInterval <= 0 {
		return errors.New("reset_interval must be greater than 0")
	}

	if p.TargetBlockSpacePercentRate.IsNegative() || p.TargetBlockSpacePercentRate.GT(math.LegacyOneDec()) {
		return errors.New("target_block_space_percent_rate must be between 0 and 1")
	}

	if p.MaxBlockChangeRate.IsNegative() || p.MaxBlockChangeRate.GT(math.LegacyOneDec()) {
		return errors.New("max_block_change_rate must be between 0 and 1")
	}

	return nil
}
