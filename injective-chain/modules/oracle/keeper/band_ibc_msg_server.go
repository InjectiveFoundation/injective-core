package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/metrics"
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
	logger := k.logger.WithFields(log.WithFn())

	// cannot process ibc request if band ibc is disabled.
	bandIBCParams := k.GetBandIBCParams(ctx)

	if !bandIBCParams.BandIbcEnabled {
		logger.Error("Cannot process new band ibc request. BandIBC is disabled.")
		return nil, sdkerrors.Wrapf(types.ErrInvalidBandIBCRequest, "BandIBC is disabled")
	}

	bandIBCOracleRequest := k.GetBandIBCOracleRequest(ctx, msg.RequestId)
	if bandIBCOracleRequest == nil {
		return nil, sdkerrors.Wrapf(types.ErrInvalidBandIBCRequest, "BandIBC oracle request not found. Should be created using governance first")
	}

	if err := k.RequestBandIBCOraclePrices(ctx, bandIBCOracleRequest); err != nil {
		logger.Error(err.Error())
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	return &types.MsgRequestBandIBCRatesResponse{}, nil
}
