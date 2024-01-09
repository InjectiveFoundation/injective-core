package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	k.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil

}

func (k msgServer) CreateDenom(goCtx context.Context, msg *types.MsgCreateDenom) (*types.MsgCreateDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom, err := k.createDenom(ctx, msg.Sender, msg.Subdenom, msg.GetName(), msg.GetSymbol())
	if err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCreateTFDenom{
		Account: getSdkAddressStringOrEmpty(msg.Sender),
		Denom:   denom,
	})

	return &types.MsgCreateDenomResponse{
		NewTokenDenom: denom,
	}, nil
}

func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// pay some extra gas cost to give a better error here.
	_, doesDenomExist := k.bankKeeper.GetDenomMetaData(ctx, msg.Amount.Denom)
	if !doesDenomExist {
		return nil, types.ErrDenomDoesNotExist.Wrapf("denom: %s", msg.Amount.Denom)
	}

	authorityMetadata, err := k.GetAuthorityMetadata(ctx, msg.Amount.Denom)
	if err != nil {
		return nil, err
	}

	if msg.Sender != authorityMetadata.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	err = k.mintTo(ctx, msg.Amount, msg.Sender)
	if err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventMintTFDenom{
		RecipientAddress: getSdkAddressStringOrEmpty(msg.Sender),
		Amount:           msg.Amount,
	})

	return &types.MsgMintResponse{}, nil
}

func (k msgServer) Burn(goCtx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.burnFrom(ctx, msg.Amount, msg.Sender)
	if err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventBurnDenom{
		BurnerAddress: getSdkAddressStringOrEmpty(msg.Sender),
		Amount:        msg.Amount,
	})

	return &types.MsgBurnResponse{}, nil
}

func (k msgServer) ChangeAdmin(goCtx context.Context, msg *types.MsgChangeAdmin) (*types.MsgChangeAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	authorityMetadata, err := k.GetAuthorityMetadata(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	if msg.Sender != authorityMetadata.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	err = k.setAdmin(ctx, msg.Denom, msg.NewAdmin)
	if err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventChangeTFAdmin{
		Denom:           msg.Denom,
		NewAdminAddress: getSdkAddressStringOrEmpty(msg.NewAdmin),
	})

	return &types.MsgChangeAdminResponse{}, nil
}

func (k msgServer) SetDenomMetadata(goCtx context.Context, msg *types.MsgSetDenomMetadata) (*types.MsgSetDenomMetadataResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Defense in depth validation of metadata
	err := msg.Metadata.Validate()
	if err != nil {
		return nil, err
	}

	authorityMetadata, err := k.GetAuthorityMetadata(ctx, msg.Metadata.Base)
	if err != nil {
		return nil, err
	}

	if msg.Sender != authorityMetadata.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	k.bankKeeper.SetDenomMetaData(ctx, msg.Metadata)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetTFDenomMetadata{
		Denom:    msg.Metadata.Base,
		Metadata: msg.Metadata,
	})

	return &types.MsgSetDenomMetadataResponse{}, nil
}

// returns the lowercased Bech32 string of the SDK address (if address is not empty)
func getSdkAddressStringOrEmpty(address string) string {
	if address == "" {
		return ""
	}

	return sdk.MustAccAddressFromBech32(address).String()
}

// nolint:all //omitted for now
// func (server msgServer) ForceTransfer(goCtx context.Context, msg *types.MsgForceTransfer) (*types.MsgForceTransferResponse, error) {
// 	ctx := sdk.UnwrapSDKContext(goCtx)

// 	authorityMetadata, err := server.Keeper.GetAuthorityMetadata(ctx, msg.Amount.GetDenom())
// 	if err != nil {
// 		return nil, err
// 	}

// 	if msg.Sender != authorityMetadata.GetAdmin() {
// 		return nil, types.ErrUnauthorized
// 	}

// 	err = server.Keeper.forceTransfer(ctx, msg.Amount, msg.TransferFromAddress, msg.TransferToAddress)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx.EventManager().EmitEvents(sdk.Events{
// 		sdk.NewEvent(
// 			types.TypeMsgForceTransfer,
// 			sdk.NewAttribute(types.AttributeTransferFromAddress, msg.TransferFromAddress),
// 			sdk.NewAttribute(types.AttributeTransferToAddress, msg.TransferToAddress),
// 			sdk.NewAttribute(types.AttributeAmount, msg.Amount.String()),
// 		),
// 	})

// 	return &types.MsgForceTransferResponse{}, nil
// }
