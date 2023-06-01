package app

import (
	"github.com/cosmos/cosmos-sdk/client"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/InjectiveLabs/injective-core/injective-chain/codec"
)

type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Marshaler         sdkcodec.Codec
	TxConfig          client.TxConfig
	Amino             *sdkcodec.LegacyAmino
}

// MakeEncodingConfig creates an EncodingConfig for testing
func MakeEncodingConfig() EncodingConfig {
	cdc := sdkcodec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()
	marshaler := sdkcodec.NewProtoCodec(interfaceRegistry)
	encodingConfig := EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		TxConfig:          tx.NewTxConfig(marshaler, tx.DefaultSignModes),
		Amino:             cdc,
	}

	codec.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	codec.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	return encodingConfig
}
