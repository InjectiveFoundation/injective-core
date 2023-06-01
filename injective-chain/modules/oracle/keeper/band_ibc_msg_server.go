package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type BandIBCMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewBandIBCMsgServerImpl returns an implementation of the band ibc MsgServer interface for the provided Keeper for band ibc oracle functions.
func NewBandIBCMsgServerImpl(keeper Keeper) BandIBCMsgServer {
	return BandIBCMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "band_ibc_msg_h",
		},
	}
}

func (k Keeper) RequestBandIBCRates(goCtx context.Context, msg *types.MsgRequestBandIBCRates) (*types.MsgRequestBandIBCRatesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// cannot process ibc request if band ibc is disabled.
	bandIBCParams := k.GetBandIBCParams(ctx)

	if !bandIBCParams.BandIbcEnabled {
		k.Logger(ctx).Error("Cannot process new band ibc request. BandIBC is disabled.")
		return nil, errors.Wrapf(types.ErrInvalidBandIBCRequest, "BandIBC is disabled")
	}

	bandIBCOracleRequest := k.GetBandIBCOracleRequest(ctx, msg.RequestId)
	if bandIBCOracleRequest == nil {
		return nil, errors.Wrapf(types.ErrInvalidBandIBCRequest, "BandIBC oracle request not found. Should be created using governance first")
	}

	if err := k.RequestBandIBCOraclePrices(ctx, bandIBCOracleRequest); err != nil {
		k.Logger(ctx).Error(err.Error())
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	return &types.MsgRequestBandIBCRatesResponse{}, nil
}
