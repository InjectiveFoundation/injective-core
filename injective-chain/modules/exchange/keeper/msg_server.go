package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

var _ types.MsgServer = MsgServer{}

type MsgServer struct {
	*Keeper

	SpotMsgServer
	DerivativesMsgServer
	BinaryOptionsMsgServer
	AccountsMsgServer
	WasmMsgServer
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the exchange MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &MsgServer{
		SpotMsgServer:          NewSpotMsgServerImpl(keeper),
		DerivativesMsgServer:   NewDerivativesMsgServerImpl(keeper),
		BinaryOptionsMsgServer: NewBinaryOptionsMsgServerImpl(keeper),
		AccountsMsgServer:      AccountsMsgServerImpl(keeper),
		WasmMsgServer:          NewWasmMsgServerImpl(keeper),
		Keeper:                 keeper,
		svcTags: metrics.Tags{
			"svc": "exchange_h",
		},
	}
}

func (m MsgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, m.svcTags)
	defer doneFn()

	if msg.Authority != m.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	m.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
