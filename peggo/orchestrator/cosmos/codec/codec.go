package codec

import (
	"cosmossdk.io/x/tx/signing"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cosmoscdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	injcodec "github.com/InjectiveLabs/injective-core/injective-chain/codec"
	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func init() {
	// set the address prefixes
	config := sdktypes.GetConfig()

	// This is specific to Injective chain
	chaintypes.SetBech32Prefixes(config)
	chaintypes.SetBip44CoinType(config)
}

func Codec() sdkcodec.Codec {
	reg, err := cosmoscdctypes.NewInterfaceRegistryWithOptions(cosmoscdctypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec: address.Bech32Codec{
				Bech32Prefix: sdktypes.GetConfig().GetBech32AccountAddrPrefix(),
			},
			ValidatorAddressCodec: address.Bech32Codec{
				Bech32Prefix: sdktypes.GetConfig().GetBech32ValidatorAddrPrefix(),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	injcodec.RegisterInterfaces(reg)
	injcodec.RegisterLegacyAminoCodec(sdkcodec.NewLegacyAmino())

	// orchestrator only needs peggy types
	peggytypes.RegisterInterfaces(reg)

	return sdkcodec.NewProtoCodec(reg)
}
