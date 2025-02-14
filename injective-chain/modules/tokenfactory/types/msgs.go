package types

import (
	"strings"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// constants
const (
	TypeMsgCreateDenom      = "create_denom"
	TypeMsgMint             = "tf_mint"
	TypeMsgBurn             = "tf_burn"
	TypeMsgChangeAdmin      = "change_admin"
	TypeMsgSetDenomMetadata = "set_denom_metadata"
	TypeMsgUpdateParams     = "update_params"
)

var _ sdk.Msg = &MsgCreateDenom{}
var _ sdk.Msg = &MsgMint{}
var _ sdk.Msg = &MsgBurn{}
var _ sdk.Msg = &MsgSetDenomMetadata{}
var _ sdk.Msg = &MsgChangeAdmin{}
var _ sdk.Msg = &MsgUpdateParams{}

func (m MsgUpdateParams) Route() string { return RouterKey }

func (m MsgUpdateParams) Type() string { return TypeMsgUpdateParams }

func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	if err := m.Params.Validate(); err != nil {
		return err
	}

	return nil
}

func (m *MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(m))
}

func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// NewMsgCreateDenom creates a msg to create a new denom
func NewMsgCreateDenom(sender, subdenom, name, symbol string, decimals uint32, allowAdminBurn bool) *MsgCreateDenom {
	return &MsgCreateDenom{
		Sender:         sender,
		Subdenom:       subdenom,
		Name:           name,
		Symbol:         symbol,
		Decimals:       decimals,
		AllowAdminBurn: allowAdminBurn,
	}
}

func (m MsgCreateDenom) Route() string { return RouterKey }
func (m MsgCreateDenom) Type() string  { return TypeMsgCreateDenom }
func (m MsgCreateDenom) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	_, err = GetTokenDenom(m.Sender, m.Subdenom)
	if err != nil {
		return errors.Wrap(ErrInvalidDenom, err.Error())
	}

	if len(m.Name) > MaxNameLength {
		return errors.Wrapf(ErrInvalidDenom, "name cannot exceed %d characters", MaxNameLength)
	}
	
	if m.Decimals > MaxDecimals {
		return errors.Wrapf(ErrInvalidDenom, "decimals cannot exceed %d", MaxDecimals)
	}

	if len(m.Symbol) > MaxSymbolLength {
		return errors.Wrapf(ErrInvalidDenom, "symbol cannot exceed %d characters", MaxSymbolLength)
	}

	return nil
}

func (m *MsgCreateDenom) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

func (m MsgCreateDenom) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// NewMsgMint creates a message to mint tokens
func NewMsgMint(sender string, amount sdk.Coin, receiver string) *MsgMint {
	return &MsgMint{
		Sender:   sender,
		Amount:   amount,
		Receiver: receiver,
	}
}

func (m MsgMint) Route() string { return RouterKey }
func (m MsgMint) Type() string  { return TypeMsgMint }
func (m MsgMint) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	if !m.Amount.IsValid() || m.Amount.IsZero() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, m.Amount.String())
	}

	if m.Receiver != "" {
		_, err = sdk.AccAddressFromBech32(m.Receiver)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid recevier address (%s)", err)
		}
	}

	return nil
}

func (m MsgMint) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgMint) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// NewMsgBurn creates a message to burn tokens
func NewMsgBurn(sender string, amount sdk.Coin, burnFrom string) *MsgBurn {
	return &MsgBurn{
		Sender:          sender,
		Amount:          amount,
		BurnFromAddress: burnFrom,
	}
}

func (m MsgBurn) Route() string { return RouterKey }
func (m MsgBurn) Type() string  { return TypeMsgBurn }
func (m MsgBurn) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	if !m.Amount.IsValid() || m.Amount.IsZero() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, m.Amount.String())
	}

	if m.BurnFromAddress != "" {
		if _, err := sdk.AccAddressFromBech32(m.BurnFromAddress); err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid burn from address (%s)", err)
		}
	}

	return nil
}

func (m *MsgBurn) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

func (m MsgBurn) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// NewMsgChangeAdmin creates a message to change the admin
func NewMsgChangeAdmin(sender, denom, newAdmin string) *MsgChangeAdmin {
	return &MsgChangeAdmin{
		Sender:   sender,
		Denom:    denom,
		NewAdmin: newAdmin,
	}
}

func (m MsgChangeAdmin) Route() string { return RouterKey }
func (m MsgChangeAdmin) Type() string  { return TypeMsgChangeAdmin }
func (m MsgChangeAdmin) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(m.NewAdmin)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid address (%s)", err)
	}

	_, _, err = DeconstructDenom(m.Denom)
	if err != nil {
		return err
	}

	return nil
}

func (m *MsgChangeAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

func (m MsgChangeAdmin) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// NewMsgSetDenomMetadata creates a message to set the denom metadata
func NewMsgSetDenomMetadata(sender string, metadata banktypes.Metadata, adminBurnDisabled *MsgSetDenomMetadata_AdminBurnDisabled) *MsgSetDenomMetadata {
	return &MsgSetDenomMetadata{
		Sender:            sender,
		Metadata:          metadata,
		AdminBurnDisabled: adminBurnDisabled,
	}
}

func (m MsgSetDenomMetadata) Route() string { return RouterKey }
func (m MsgSetDenomMetadata) Type() string  { return TypeMsgSetDenomMetadata }
func (m MsgSetDenomMetadata) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	err = m.Metadata.Validate()
	if err != nil {
		return err
	}

	if m.Metadata.Base == types.InjectiveCoin {
		return errors.Wrap(ErrInvalidDenom, "cannot set metadata for INJ")
	}

	err = sdk.ValidateDenom(m.Metadata.Base)
	if err != nil {
		return err
	}

	// If denom metadata is for a TokenFactory denom, run the different components validations
	strParts := strings.Split(m.Metadata.Base, "/")
	if len(strParts) > 2 {
		_, _, err = DeconstructDenom(m.Metadata.Base)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MsgSetDenomMetadata) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

func (m MsgSetDenomMetadata) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}
