package types

import (
	"cosmossdk.io/errors"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		TokenPairs: []TokenPair{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	err := gs.Params.Validate()
	if err != nil {
		return err
	}

	seenDenoms := map[string]struct{}{}
	seenErc20Addresses := map[string]struct{}{}

	for _, pair := range gs.GetTokenPairs() {
		// Validate individual TokenPair
		if err := pair.Validate(); err != nil {
			return err
		}

		// Check for duplicate bank denoms
		if _, ok := seenDenoms[pair.GetBankDenom()]; ok {
			return errors.Wrapf(ErrInvalidGenesis, "duplicate bank denom: %s", pair.GetBankDenom())
		}
		seenDenoms[pair.GetBankDenom()] = struct{}{}

		// Check for duplicate ERC20 addresses
		erc20Address := ethcommon.HexToAddress(pair.GetErc20Address())

		if _, ok := seenErc20Addresses[string(erc20Address.Bytes())]; ok {
			return errors.Wrapf(ErrInvalidGenesis, "duplicate ERC20 address: %s", erc20Address.Hex())
		}
		seenErc20Addresses[string(erc20Address.Bytes())] = struct{}{}
	}

	return nil
}
