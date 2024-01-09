package keeper

import (
	"context"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "wasmx_h",
		},
	}
}

func (m msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(m.svcTags)()

	if msg.Authority != m.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	m.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (m msgServer) ExecuteContractCompat(goCtx context.Context, msg *types.MsgExecuteContractCompat) (*types.MsgExecuteContractCompatResponse, error) {
	wasmMsgServer := wasmkeeper.NewMsgServerImpl(&m.wasmKeeper)

	funds := sdk.Coins{}
	if msg.Funds != "0" {
		funds, _ = sdk.ParseCoinsNormalized(msg.Funds)
	}

	oMsg := &wasmtypes.MsgExecuteContract{
		Sender:   msg.Sender,
		Contract: msg.Contract,
		Msg:      []byte(msg.Msg),
		Funds:    funds,
	}

	res, err := wasmMsgServer.ExecuteContract(goCtx, oMsg)
	if err != nil {
		return nil, err
	}

	return &types.MsgExecuteContractCompatResponse{
		Data: res.Data,
	}, nil
}

func (m msgServer) UpdateRegistryContractParams(goCtx context.Context, msg *types.MsgUpdateContract) (*types.MsgUpdateContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	contractAddr := sdk.MustAccAddressFromBech32(msg.ContractAddress)

	contract, err := m.fetchContractAndCheckAccessControl(ctx, contractAddr, msg)
	if err != nil {
		return nil, err
	}

	m.updateRegisteredContractData(ctx, contractAddr, contract, func(contract *types.RegisteredContract) {
		contract.GasLimit = msg.GasLimit
		contract.GasPrice = msg.GasPrice
		contract.AdminAddress = msg.AdminAddress
	})
	return &types.MsgUpdateContractResponse{}, nil
}

func (m msgServer) ActivateRegistryContract(goCtx context.Context, msg *types.MsgActivateContract) (*types.MsgActivateContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	contractAddr := sdk.MustAccAddressFromBech32(msg.ContractAddress)

	contract, err := m.fetchContractAndCheckAccessControl(ctx, contractAddr, msg)
	if err != nil {
		return nil, err
	}

	m.updateRegisteredContractData(ctx, contractAddr, contract, func(contract *types.RegisteredContract) {
		contract.IsExecutable = true
	})

	return &types.MsgActivateContractResponse{}, nil
}

func (m msgServer) DeactivateRegistryContract(goCtx context.Context, msg *types.MsgDeactivateContract) (*types.MsgDeactivateContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	contractAddr := sdk.MustAccAddressFromBech32(msg.ContractAddress)

	contract, err := m.fetchContractAndCheckAccessControl(ctx, contractAddr, msg)
	if err != nil {
		return nil, err
	}
	if err := m.Keeper.DeactivateContract(ctx, contractAddr, contract); err != nil {
		return nil, err
	}
	return &types.MsgDeactivateContractResponse{}, nil
}

func (m msgServer) RegisterContract(goCtx context.Context, msg *types.MsgRegisterContract) (*types.MsgRegisterContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	accessConfig := m.wasmViewKeeper.GetParams(ctx).CodeUploadAccess
	isRegistrationAllowed := types.IsAllowed(accessConfig, sender)

	if !isRegistrationAllowed {
		return nil, sdkerrors.ErrUnauthorized.Wrap("Unauthorized to register contract")
	}

	if err := m.Keeper.HandleContractRegistration(ctx, params, msg.ContractRegistrationRequest); err != nil {
		return nil, err
	}

	return &types.MsgRegisterContractResponse{}, nil
}

func (m msgServer) fetchContractAndCheckAccessControl(ctx sdk.Context, contractAddr sdk.AccAddress, msg sdk.Msg) (*types.RegisteredContract, error) {
	contract := m.Keeper.GetContractByAddress(ctx, contractAddr)
	if contract == nil {
		return nil, errors.Wrapf(sdkerrors.ErrNotFound, "Contract with address %s not found", contractAddr.String())
	}

	senderAddr := msg.GetSigners()[0]

	if !(senderAddr.Equals(contractAddr) || (len(contract.AdminAddress) > 0 && senderAddr.String() == contract.AdminAddress)) {
		return nil, sdkerrors.ErrUnauthorized.Wrapf("Unauthorized to update contract %v", contractAddr.String())
	}
	return contract, nil
}

func (m msgServer) updateRegisteredContractData(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	registeredContract *types.RegisteredContract,
	updateFn func(contract *types.RegisteredContract),
) {
	updateFn(registeredContract)
	m.Keeper.SetContract(ctx, contractAddr, *registeredContract)
}
