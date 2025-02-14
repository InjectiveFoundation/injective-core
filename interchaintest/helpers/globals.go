package helpers

import (
	"testing"

	ctypes "github.com/InjectiveLabs/sdk-go/chain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var SomeoneAddress sdk.AccAddress

func init() {
	addr, err := sdk.AccAddressFromHexUnsafe("0000000000000000000000000000000000000000")
	if err != nil {
		panic("failed to init address")
	}

	SomeoneAddress = addr
}

const (
	Bech32MainPrefix          = "inj"
	InjectiveBondDenom        = "inj"
	InjectiveCoinType         = 60
	InjectiveSigningAlgorithm = "eth_secp256k1"
)

var InjectiveCoinDecimals = int64(18)

func SetAccountPrefixes(accountAddressPrefix string) {
	// Set prefixes
	accountPubKeyPrefix := accountAddressPrefix + "pub"
	validatorAddressPrefix := accountAddressPrefix + "valoper"
	validatorPubKeyPrefix := accountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := accountAddressPrefix + "valcons"
	consNodePubKeyPrefix := accountAddressPrefix + "valconspub"

	// Set and seal config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(accountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)

	ctypes.SetBech32Prefixes(config)
	ctypes.SetBip44CoinType(config)
}

func debugOutput(t *testing.T, stdout string) {
	if len(stdout) == 0 {
		return
	}

	if true {
		t.Log(stdout)
	}
}
