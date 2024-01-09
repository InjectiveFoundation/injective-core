package module

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/client/cli"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/exported"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

const (
	// ModuleName defines the module name
	ModuleName = types.ModuleName
	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName
	// StoreKey defines the primary module store key
	StoreKey = ModuleName
	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_" + StoreKey
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

const ConsensusVersion = 1

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface for the capability module.
type AppModuleBasic struct{}

func NewAppModuleBasic() AppModuleBasic {
	return AppModuleBasic{}
}

// Name returns the x/permissions module's name.
func (AppModuleBasic) Name() string {
	return ModuleName
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the x/permissions module's default genesis state.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the x/permissions module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", ModuleName, err)
	}

	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, muxServer *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), muxServer, types.NewQueryClient(clientCtx)) // nolint:errcheck //we don't care
}

// GetTxCmd returns the x/permissions module's root tx command.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the x/permissions module's root query command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface for the capability module.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper

	bankKeeper types.BankKeeper
	tfKeeper   types.TokenFactoryKeeper
	wasmKeeper types.WasmKeeper

	legacySubspace exported.Subspace
}

func NewAppModule(
	k keeper.Keeper,
	bankKeeper types.BankKeeper,
	tfKeeper types.TokenFactoryKeeper,
	wasmKeeper types.WasmKeeper,
	legacySubspace exported.Subspace,
) AppModule {
	// set global bank transfer hook
	bankKeeper.AppendSendRestriction(k.SendRestrictionFn)

	return AppModule{
		AppModuleBasic: NewAppModuleBasic(),
		keeper:         k,
		bankKeeper:     bankKeeper,
		tfKeeper:       tfKeeper,
		wasmKeeper:     wasmKeeper,
		legacySubspace: legacySubspace,
	}
}

// Name returns the x/permissions module's name.
func (am AppModule) Name() string {
	return am.AppModuleBasic.Name()
}

// QuerierRoute returns the x/permissions module's query routing key.
func (AppModule) QuerierRoute() string { return QuerierRoute }

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))
}

// RegisterInvariants registers the x/permissions module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the x/permissions module's genesis initialization. It
// returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)

	am.keeper.InitGenesis(ctx, genState)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the x/permissions module's exported genesis state as raw
// JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion implements ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// ___________________________________________________________________________

// AppModuleSimulation functions

func (am AppModule) GenerateGenesisState(input *module.SimulationState) {
	input.GenState[ModuleName] = input.Cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ProposalMsgs returns all the distribution content functions used to
// simulate governance proposals.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}

// RegisterStoreDecoder registers a decoder for permissions module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
}

// WeightedOperations returns simulator module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}
