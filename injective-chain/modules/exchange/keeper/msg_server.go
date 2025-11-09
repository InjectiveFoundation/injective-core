package keeper

import (
	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

var _ v2.MsgServer = MsgServer{}

type MsgServer struct {
	SpotMsgServer
	DerivativesMsgServer
	BinaryOptionsMsgServer
	AccountsMsgServer
	GeneralMsgServer
	WasmMsgServer
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the exchange MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) v2.MsgServer {
	return &MsgServer{
		SpotMsgServer:          NewSpotMsgServerImpl(keeper),
		DerivativesMsgServer:   NewDerivativesMsgServerImpl(keeper),
		BinaryOptionsMsgServer: NewBinaryOptionsMsgServerImpl(keeper),
		AccountsMsgServer:      AccountsMsgServerImpl(keeper),
		GeneralMsgServer:       NewGeneralMsgServerImpl(keeper),
		WasmMsgServer:          NewWasmMsgServerImpl(keeper),
		svcTags: metrics.Tags{
			"svc": "exchange_h",
		},
	}
}

var _ types.MsgServer = v1MsgServer{}

type v1MsgServer struct {
	SpotV1MsgServer
	DerivativesV1MsgServer
	BinaryOptionsV1MsgServer
	AccountsV1MsgServer
	WasmV1MsgServer
	*Keeper
	server  v2.MsgServer
	svcTags metrics.Tags
}

// NewV1MsgServerImpl returns an implementation of the exchange MsgServer interface
// for the provided Keeper.
func NewV1MsgServerImpl(keeper *Keeper, server v2.MsgServer) types.MsgServer {
	return &v1MsgServer{
		SpotV1MsgServer:          NewSpotV1MsgServerImpl(*keeper, server),
		DerivativesV1MsgServer:   NewDerivativesV1MsgServerImpl(*keeper, server),
		BinaryOptionsV1MsgServer: NewBinaryOptionsV1MsgServerImpl(*keeper, server),
		AccountsV1MsgServer:      AccountsV1MsgServerImpl(*keeper, server),
		WasmV1MsgServer:          NewWasmV1MsgServerImpl(*keeper, server),
		Keeper:                   keeper,
		server:                   server,
		svcTags: metrics.Tags{
			"svc": "exchange_v1_h",
		},
	}
}
