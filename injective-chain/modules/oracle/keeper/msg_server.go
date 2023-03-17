package keeper

import (
	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type MsgServer struct {
	BandMsgServer
	BandIBCMsgServer
	PricefeedMsgServer
	CoinbaseMsgServer
	ProviderMsgServer
	PythMsgServer

	Keeper
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the oracle MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {

	return &MsgServer{
		BandMsgServer:      NewBandMsgServerImpl(keeper),
		BandIBCMsgServer:   NewBandIBCMsgServerImpl(keeper),
		PricefeedMsgServer: NewPricefeedMsgServerImpl(keeper),
		CoinbaseMsgServer:  NewCoinbaseMsgServerImpl(keeper),
		ProviderMsgServer:  NewProviderMsgServerImpl(keeper),
		PythMsgServer:      NewPythMsgServerImpl(keeper),
		Keeper:             keeper,
		svcTags: metrics.Tags{
			"svc": "oracle_h",
		},
	}
}

var _ types.MsgServer = MsgServer{}
