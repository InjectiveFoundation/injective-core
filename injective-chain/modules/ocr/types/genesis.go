package types

func NewGenesisState() GenesisState {
	return GenesisState{}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}
