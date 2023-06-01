package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/metrics"
)

var _ types.MsgServer = MsgServer{}

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

func (m MsgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
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
