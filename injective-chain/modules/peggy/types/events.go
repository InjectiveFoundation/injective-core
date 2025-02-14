package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func NewEventDepositReceived(sender common.Address, receiver sdk.AccAddress, amount sdk.Coin) *EventDepositReceived {
	return &EventDepositReceived{
		Sender:   sender.String(),
		Receiver: receiver.String(),
		Amount:   amount,
	}
}
