package ocr

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/exported"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/client/cli"
	ocrkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

// type check to ensure the interface is properly implemented
var (
	_ module.AppModuleBasic      = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.HasServices         = AppModule{}
	_ module.HasConsensusVersion = AppModule{}

	_ appmodule.AppModule       = AppModule{}
	_ appmodule.HasBeginBlocker = AppModule{}
	_ appmodule.HasEndBlocker   = AppModule{}
)

const ConsensusVersion = 2

// app module Basics object
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the ocr module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers interfaces and implementations of the ocr module.
func (AppModuleBasic) RegisterInterfaces(interfaceRegistry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(interfaceRegistry)
}

// DefaultGenesis returns default genesis state as raw bytes for the ocr
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (b AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genesisState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genesisState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return genesisState.Validate()
}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(c client.Context, serveMux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), serveMux, types.NewQueryClient(c)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the ocr module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd returns no root query command for the ocr module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

type AppModule struct {
	AppModuleBasic

	svcTags        metrics.Tags
	keeper         ocrkeeper.Keeper
	accountKeeper  authkeeper.AccountKeeper
	bankKeeper     bankkeeper.Keeper
	legacySubspace exported.Subspace
}

func (am AppModule) IsOnePerModuleType() {}

func (am AppModule) IsAppModule() {}

// NewAppModule creates a new AppModule Object
func NewAppModule(
	keeper ocrkeeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	legacySubspace exported.Subspace,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		legacySubspace: legacySubspace,
		svcTags: metrics.Tags{
			"svc": "ocr_m",
		},
	}
}

func (am AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

func (AppModule) Name() string {
	return types.ModuleName
}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

func (am AppModule) QuerierRoute() string { return types.RouterKey }

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), ocrkeeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), &am.keeper)

	migrator := ocrkeeper.NewMigrator(am.keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(types.ModuleName, 1, migrator.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate ocr from version 1 to 2: %v", err))
	}
}

func (am AppModule) BeginBlock(ctx context.Context) error {
	am.BeginBlocker(sdk.UnwrapSDKContext(ctx), am.keeper)
	return nil
}

func (am AppModule) EndBlock(ctx context.Context) error {
	am.EndBlocker(sdk.UnwrapSDKContext(ctx), am.keeper)
	return nil
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState types.GenesisState

	cdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.keeper, genesisState)
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

func (am AppModule) GenerateGenesisState(input *module.SimulationState) {
}

func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}

func (am AppModule) RegisterStoreDecoder(decoderRegistry simtypes.StoreDecoderRegistry) {
}

func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{}
}
