package peggy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/exported"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/client/cli"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"

	// "github.com/cosmos/cosmos-sdk/x/gov/simulation"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

const ConsensusVersion = 2

// AppModuleBasic object for module implementation
type AppModuleBasic struct{}

// Name implements app module basic
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec implements app module basic
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis implements app module basic
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis implements app module basic
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return data.ValidateBasic()
}

// GetQueryCmd implements app module basic
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// GetTxCmd implements app module basic
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd(types.StoreKey)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the distribution module.
// also implements app modeul basic
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, serverMux *runtime.ServeMux) {
	err := types.RegisterQueryHandlerClient(context.Background(), serverMux, types.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

// RegisterInterfaces implements app bmodule basic
func (b AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// AppModule object for module implementation
type AppModule struct {
	AppModuleBasic
	keeper         keeper.Keeper
	bankKeeper     bankkeeper.Keeper
	legacySubspace exported.Subspace // used for x/params migration
	blockHandler   *BlockHandler
}

func (am AppModule) ConsensusVersion() uint64 {
	return ConsensusVersion
}

// NewAppModule creates a new AppModule Object
func NewAppModule(
	k keeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	ss exported.Subspace,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		bankKeeper:     bankKeeper,
		legacySubspace: ss,
		blockHandler:   NewBlockHandler(k),
	}
}

// Name implements app module
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterInvariants implements app module
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	// TODO: make some invariants in the peggy module to ensure that
	// coins aren't being fraudlently minted etc...
}

// QuerierRoute implements app module
func (am AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), &am.keeper)

	migrator := keeper.NewMigrator(am.keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(types.ModuleName, 1, migrator.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate peggy from version 1 to 2: %v", err))
	}
}

// InitGenesis initializes the genesis state for this module and implements app module.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	genesisState := new(types.GenesisState)
	cdc.MustUnmarshalJSON(data, genesisState)
	keeper.InitGenesis(ctx, am.keeper, genesisState)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports the current genesis state to a json.RawMessage
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := keeper.ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(&gs)
}

// BeginBlock implements app module
func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock implements app module
func (am AppModule) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	am.blockHandler.EndBlocker(ctx)
	return []abci.ValidatorUpdate{}
}

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the distribution module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	// nolint:all
	// simulation.RandomizedGenState(simState)
}

// ProposalMsgs returns all the distribution content functions used to
// simulate governance proposals.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}

// RegisterStoreDecoder registers a decoder for distribution module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
	// nolint:all
	// sdr[types.StoreKey] = simulation.NewDecodeStore(am.cdc)
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	// nolint:all
	// return simulation.WeightedOperations(
	// simState.AppParams, simState.Cdc, am.accountKeeper, am.bankKeeper, am.keeper, am.stakingKeeper,
	// )
	return nil
}
