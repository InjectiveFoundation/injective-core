package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	permissionstypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
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

	denom, err := k.createDenom(ctx, msg.Sender, msg.Subdenom, msg.GetName(), msg.GetSymbol(), msg.GetDecimals(), msg.GetAllowAdminBurn())
	if err != nil {
		return nil, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCreateDenom{
		Account: msg.Sender,
		Denom:   denom,
	})

	return &types.MsgCreateDenomResponse{
		NewTokenDenom: denom,
	}, nil
}

func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom := msg.Amount.Denom
	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	// pay some extra gas cost to give a better error here.
	_, doesDenomExist := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !doesDenomExist {
		return nil, types.ErrDenomDoesNotExist.Wrapf("denom: %s", denom)
	}

	authorityMetadata, err := k.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return nil, err
	}

	hasPermissionsNamespace := k.permissionsKeeper.HasNamespace(ctx, denom)

	// for non-permissioned tokens, only the admin can mint
	if !hasPermissionsNamespace && msg.Sender != authorityMetadata.GetAdmin() {
		return nil, types.ErrUnauthorized
	}
	if hasPermissionsNamespace && !k.permissionsKeeper.HasPermissionsForAction(ctx, denom, sender, permissionstypes.Action_MINT) {
		return nil, types.ErrUnauthorized.Wrapf("sender %s, for %s action on denom: %s", sender, permissionstypes.Action_MINT, denom)
	}

	receiver := sender
	if msg.Receiver != "" {
		receiver = sdk.MustAccAddressFromBech32(msg.Receiver)
	}

	err = k.mintTo(ctx, msg.Amount, receiver)
	if err != nil {
		return nil, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.EventMint{
		Minter:   msg.Sender,
		Amount:   msg.Amount,
		Receiver: receiver.String(),
	})

	return &types.MsgMintResponse{}, nil
}

func (k msgServer) Burn(goCtx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	denom := msg.Amount.Denom
	burnFromAddr := sender

	// for backwards compatibility, keep burnFromAddr as the sender if it's unspecified
	if msg.BurnFromAddress != "" {
		burnFromAddr = sdk.MustAccAddressFromBech32(msg.BurnFromAddress)
	}

	hasPermissionsNamespace := k.permissionsKeeper.HasNamespace(ctx, denom)

	err := k.verifyBurnPermissions(ctx, denom, sender, burnFromAddr, hasPermissionsNamespace)
	if err != nil {
		return nil, err
	}

	err = k.burnFrom(ctx, msg.Amount, burnFromAddr)
	if err != nil {
		return nil, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.EventBurn{
		Burner:   msg.Sender,
		Amount:   msg.Amount,
		BurnFrom: burnFromAddr.String(),
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

	_ = ctx.EventManager().EmitTypedEvent(&types.EventChangeAdmin{
		Denom:           msg.Denom,
		NewAdminAddress: msg.NewAdmin,
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

	isAdminAllowed := authorityMetadata.GetAdmin() != "" && msg.Sender == authorityMetadata.GetAdmin()
	isGovernanceAllowed := authorityMetadata.GetAdmin() == "" && msg.Sender == k.authority
	if !(isAdminAllowed || isGovernanceAllowed) {
		return nil, types.ErrUnauthorized
	}

	existingMetadata, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Metadata.Base)
	if found {
		if existingMetadata.Decimals != 0 && existingMetadata.Decimals != msg.Metadata.Decimals {
			return nil, fmt.Errorf("cannot update denom metadata decimals")
		}
	}

	k.bankKeeper.SetDenomMetaData(ctx, msg.Metadata)

	// only allow disabling admin burn if it was previously enabled
	if msg.AdminBurnDisabled != nil && authorityMetadata.AdminBurnAllowed {
		authorityMetadata.AdminBurnAllowed = !msg.AdminBurnDisabled.ShouldDisable
		err = k.SetAuthorityMetadata(ctx, msg.Metadata.Base, authorityMetadata)
		if err != nil {
			return nil, err
		}
	} else if msg.AdminBurnDisabled != nil && !authorityMetadata.AdminBurnAllowed {
		return nil, fmt.Errorf("cannot enable AdminBurnAllowed if it was previously disabled")
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSetDenomMetadata{
		Denom:    msg.Metadata.Base,
		Metadata: msg.Metadata,
	})

	return &types.MsgSetDenomMetadataResponse{}, nil
}
