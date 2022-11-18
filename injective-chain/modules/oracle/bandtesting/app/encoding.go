package band

import (
	"github.com/cosmos/cosmos-sdk/std"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/bandtesting/app/params"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
)

// MakeEncodingConfig creates an EncodingConfig for testing
func MakeEncodingConfig() simappparams.EncodingConfig {
	encodingConfig := params.MakeEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
