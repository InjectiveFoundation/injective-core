package types

import (
	"context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
)

var (
	_ authztypes.Authorization         = &ContractExecutionCompatAuthorization{}
	_ cdctypes.UnpackInterfacesMessage = &ContractExecutionCompatAuthorization{}
)

// NewContractExecutionCompatAuthorization constructor
func NewContractExecutionCompatAuthorization(grants ...wasmtypes.ContractGrant) *ContractExecutionCompatAuthorization {
	return &ContractExecutionCompatAuthorization{
		Grants: grants,
	}
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (ContractExecutionCompatAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgExecuteContractCompat{})
}

// NewAuthz factory method to create an Authorization with updated grants
func (ContractExecutionCompatAuthorization) NewAuthz(g []wasmtypes.ContractGrant) authztypes.Authorization {
	return NewContractExecutionCompatAuthorization(g...)
}

// Accept implements Authorization.Accept.
func (a *ContractExecutionCompatAuthorization) Accept(goCtx context.Context, msg sdk.Msg) (authztypes.AcceptResponse, error) {
	wasmxMsg, ok := msg.(*MsgExecuteContractCompat)
	if !ok {
		return authztypes.AcceptResponse{}, sdkerrors.ErrInvalidRequest.Wrap("msg is not a MsgExecuteContractCompat")
	}

	// convert MsgExecuteContractCompat to MsgExecuteContract
	funds := sdk.Coins{}
	if wasmxMsg.Funds != "0" {
		funds, _ = sdk.ParseCoinsNormalized(wasmxMsg.Funds)
	}

	wasmMsg := &wasmtypes.MsgExecuteContract{
		Sender:   wasmxMsg.Sender,
		Contract: wasmxMsg.Contract,
		Msg:      []byte(wasmxMsg.Msg),
		Funds:    funds,
	}

	// convert wasmx auth to wasm auth
	wasmAuth := wasmtypes.NewContractExecutionAuthorization(a.Grants...)

	// and check via converted values
	res, err := wasmAuth.Accept(goCtx, wasmMsg)

	if res.Updated != nil { // hot-patch the updated authorization type back to "Compat" version
		res.Updated = &ContractExecutionCompatAuthorization{
			Grants: res.Updated.(*wasmtypes.ContractExecutionAuthorization).Grants,
		}
	}

	return res, err
}

// ValidateBasic implements Authorization.ValidateBasic.
func (a ContractExecutionCompatAuthorization) ValidateBasic() error {
	wasmAuth := wasmtypes.NewContractExecutionAuthorization(a.Grants...)
	return wasmAuth.ValidateBasic()
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (a ContractExecutionCompatAuthorization) UnpackInterfaces(unpacker cdctypes.AnyUnpacker) error {
	for _, g := range a.Grants {
		if err := g.UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	return nil
}
