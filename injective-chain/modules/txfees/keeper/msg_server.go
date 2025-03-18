package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	keeper  *Keeper
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{
		keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "txfees_h",
		},
	}
}

func (m msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, m.svcTags)
	defer doneFn()

	if msg.Authority != m.keeper.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", m.keeper.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	m.keeper.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
