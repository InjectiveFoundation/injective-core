package helpers

import (
	"cosmossdk.io/math"
	tokenfactorytypes "github.com/InjectiveLabs/sdk-go/chain/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/types"
)

func NewMsgCreateDenom(sender, subdenom, name, symbol string, decimals uint32, allowAdminBurn bool) *tokenfactorytypes.MsgCreateDenom {
	return &tokenfactorytypes.MsgCreateDenom{
		Sender:         sender,
		Subdenom:       subdenom,
		Name:           name,
		Symbol:         symbol,
		Decimals:       decimals,
		AllowAdminBurn: allowAdminBurn,
	}
}

func NewMsgMint(sender, denom string) *tokenfactorytypes.MsgMint {
	return &tokenfactorytypes.MsgMint{
		Sender:   sender,
		Amount:   types.NewCoin(denom, math.NewInt(1000000000000000000)),
		Receiver: sender,
	}
}
