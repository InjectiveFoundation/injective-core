package types

import (
	"encoding/hex"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrortypes "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

func (l *RateLimit) TotalInflow() sdkmath.Int {
	sum := sdkmath.ZeroInt()
	for _, transfer := range l.Transfers {
		if transfer.IsDeposit {
			sum = sum.Add(transfer.Amount)
		}
	}

	return sum
}

func (l *RateLimit) TotalOutflow() sdkmath.Int {
	sum := sdkmath.ZeroInt()
	for _, transfer := range l.Transfers {
		if !transfer.IsDeposit {
			sum = sum.Add(transfer.Amount)
		}
	}

	return sum
}

var (
	_ sdk.Msg = &MsgCreateRateLimit{}
	_ sdk.Msg = &MsgUpdateRateLimit{}
	_ sdk.Msg = &MsgRemoveRateLimit{}
)

func (*MsgCreateRateLimit) Route() string { return RouterKey }

func (*MsgCreateRateLimit) Type() string { return "create_rate_limit" }

func (msg *MsgCreateRateLimit) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg *MsgCreateRateLimit) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.MustAccAddressFromBech32(msg.Authority)}
}

//revive:disable:cyclomatic //no comply
func (msg *MsgCreateRateLimit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidAddress, "invalid authority address: %s", msg.Authority)
	}

	if msg.TokenAddress == "" || !gethcommon.IsHexAddress(msg.TokenAddress) {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidAddress, "invalid token_address: %s", msg.TokenAddress)
	}

	if msg.TokenDecimals == 0 {
		return sdkerrors.Wrap(sdkerrortypes.ErrInvalidRequest, "token_decimals cannot be zero")
	}

	if !isValidPythID(msg.TokenPriceId) {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidRequest, "invalid token_price_id: %s", msg.TokenPriceId)
	}

	if msg.RateLimitUsd.IsNil() || msg.RateLimitUsd.IsZero() {
		return sdkerrors.Wrap(sdkerrortypes.ErrInvalidRequest, "rate_limit_usd cannot be zero")
	}

	if msg.RateLimitWindow == 0 {
		return sdkerrors.Wrap(sdkerrortypes.ErrInvalidRequest, "rate_limit_window cannot be zero")
	}

	if msg.AbsoluteMintLimit.IsNil() || msg.AbsoluteMintLimit.IsZero() {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidRequest, "absolute_mint_limit cannot be zero")
	}

	return nil
}

func (*MsgUpdateRateLimit) Route() string { return RouterKey }

func (*MsgUpdateRateLimit) Type() string { return "update_rate_limit" }

func (msg *MsgUpdateRateLimit) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg *MsgUpdateRateLimit) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.MustAccAddressFromBech32(msg.Authority)}
}

func (msg *MsgUpdateRateLimit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidAddress, "invalid authority address: %s", msg.Authority)
	}

	if msg.TokenAddress == "" || !gethcommon.IsHexAddress(msg.TokenAddress) {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidAddress, "invalid token_address: %s", msg.TokenAddress)
	}

	if !isValidPythID(msg.NewTokenPriceId) {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidRequest, "invalid new_token_price_id: %s", msg.NewTokenPriceId)
	}

	if msg.NewRateLimitUsd.IsNil() || msg.NewRateLimitUsd.IsZero() {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidRequest, "new_rate_limit_usd cannot be zero")
	}

	if msg.NewRateLimitWindow == 0 {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidRequest, "new_rate_limit_window cannot be zero")
	}

	return nil
}

func (*MsgRemoveRateLimit) Route() string { return RouterKey }

func (*MsgRemoveRateLimit) Type() string { return "remove_rate_limit" }

func (msg *MsgRemoveRateLimit) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg *MsgRemoveRateLimit) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.MustAccAddressFromBech32(msg.Authority)}
}

func (msg *MsgRemoveRateLimit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidAddress, "invalid authority address: %s", msg.Authority)
	}

	if msg.TokenAddress == "" || !gethcommon.IsHexAddress(msg.TokenAddress) {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInvalidAddress, "invalid token_address: %s", msg.TokenAddress)
	}

	return nil
}

func isValidPythID(s string) bool {
	_, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	return err == nil
}
