package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	tftypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

type BankKeeper interface {
	AppendSendRestriction(restriction banktypes.SendRestrictionFn)
	PrependSendRestriction(restriction banktypes.SendRestrictionFn)
	ClearSendRestriction()
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type TokenFactoryKeeper interface {
	GetAuthorityMetadata(ctx sdk.Context, denom string) (tftypes.DenomAuthorityMetadata, error)
}

type WasmKeeper interface {
	HasContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) bool
	QuerySmart(ctx sdk.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
}
