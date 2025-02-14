package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type WasmHookMsg struct {
	From    sdk.AccAddress `json:"from_address"`
	To      sdk.AccAddress `json:"to_address"`
	Action  string         `json:"action"`
	Amounts sdk.Coins      `json:"amounts"`
}

func NewWasmHookMsg(fromAddr, toAddr sdk.AccAddress, action Action, amount sdk.Coin) WasmHookMsg {
	return WasmHookMsg{
		From:    fromAddr,
		To:      toAddr,
		Action:  action.String(),
		Amounts: sdk.NewCoins(amount),
	}
}

func GetWasmHookMsgBytes(fromAddr, toAddr sdk.AccAddress, action Action, amount sdk.Coin) ([]byte, error) {
	wasmHookMsg := struct {
		SendRestriction WasmHookMsg `json:"send_restriction"`
	}{NewWasmHookMsg(fromAddr, toAddr, action, amount)}

	bz, err := json.Marshal(wasmHookMsg)
	if err != nil {
		return nil, err
	}

	return bz, nil
}
