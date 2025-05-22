package simulation

import (
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/staking"

	codectypes "github.com/InjectiveLabs/injective-core/injective-chain/codec/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm"
)

func MakeSimEncodingConfig(moduleManager module.BasicManager) *codectypes.EncodingConfig {
	if moduleManager == nil {
		moduleManager = module.NewBasicManager(
			auth.AppModuleBasic{},
			bank.AppModuleBasic{},
			distr.AppModuleBasic{},
			gov.NewAppModuleBasic([]govclient.ProposalHandler{paramsclient.ProposalHandler}),
			staking.AppModuleBasic{},

			// EVM related modules
			evm.AppModuleBasic{},
		)
	}

	encCfg := codectypes.MakeEncodingConfig()
	moduleManager.RegisterInterfaces(encCfg.InterfaceRegistry)
	moduleManager.RegisterLegacyAminoCodec(encCfg.Amino)

	return &encCfg
}
