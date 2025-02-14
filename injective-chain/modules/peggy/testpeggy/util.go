package testpeggy

import sdk "github.com/cosmos/cosmos-sdk/types"

func GetDefaultValidatorSet() []ValidatorInfo {
	return []ValidatorInfo{
		{
			AccAddr:  AccAddrs[0],
			OrchAddr: sdk.AccAddress(AccAddrs[0]),
			ValAddr:  ValAddrs[0],
			EthAddr:  EthAddrs[0],
			ConsKey:  ConsPubKeys[0],
			PubKey:   AccPubKeys[0],
		},

		{
			AccAddr:  AccAddrs[1],
			OrchAddr: sdk.AccAddress(AccAddrs[1]),
			ValAddr:  ValAddrs[1],
			EthAddr:  EthAddrs[1],
			ConsKey:  ConsPubKeys[1],
			PubKey:   AccPubKeys[1],
		},

		{
			AccAddr:  AccAddrs[2],
			OrchAddr: sdk.AccAddress(AccAddrs[2]),
			ValAddr:  ValAddrs[2],
			EthAddr:  EthAddrs[2],
			ConsKey:  ConsPubKeys[2],
			PubKey:   AccPubKeys[2],
		},

		{
			AccAddr:  AccAddrs[3],
			OrchAddr: sdk.AccAddress(AccAddrs[3]),
			ValAddr:  ValAddrs[3],
			EthAddr:  EthAddrs[3],
			ConsKey:  ConsPubKeys[3],
			PubKey:   AccPubKeys[3],
		},

		{
			AccAddr:  AccAddrs[4],
			OrchAddr: sdk.AccAddress(AccAddrs[4]),
			ValAddr:  ValAddrs[4],
			EthAddr:  EthAddrs[4],
			ConsKey:  ConsPubKeys[4],
			PubKey:   AccPubKeys[4],
		},
	}
}
