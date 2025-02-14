package types

import (
	"cosmossdk.io/errors"
)

// DefaultGenesis returns the default Permissions genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		Namespaces: []Namespace{},
		Vouchers:   []*AddressVoucher{},
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

	for i := range gs.GetNamespaces() {
		ns := gs.GetNamespaces()[i]
		if _, ok := seenDenoms[ns.GetDenom()]; ok {
			return errors.Wrapf(ErrInvalidGenesis, "duplicate denom: %s", ns.GetDenom())
		}
		seenDenoms[ns.GetDenom()] = struct{}{}
	}

	return nil
}
