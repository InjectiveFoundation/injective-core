package governancelane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	skipbase "github.com/skip-mev/block-sdk/v2/block/base"

	"github.com/InjectiveLabs/injective-core/injective-chain/lanes/helpers"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "governance"
)

type GovernanceMatchHandler struct {
	ExchangeKeeper  *exchangekeeper.Keeper
	SignerExtractor signerextraction.Adapter
}

// GovernanceMatchHandler returns the governance match handler for the governance lane.
func (h *GovernanceMatchHandler) GovernanceMatchHandler() skipbase.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		sigTx, ok := tx.(signing.SigVerifiableTx)
		if !ok {
			ctx.Logger().Error("Error converting to sigTx")
			return false
		}

		sigs, err := sigTx.GetSignaturesV2()
		if err != nil {
			ctx.Logger().Error("Error getting signatures", "error", err)
			return false
		}

		if len(sigs) == 0 {
			return false
		}

		// only check first signature for performance reasons
		firstSigner := helpers.NewAccAddress(sigs[0].PubKey.Address()).String()
		return h.ExchangeKeeper.IsAdmin(ctx, firstSigner)
	}
}

// NewGovernanceLane returns a new governance lane. The governance lane orders transactions by the transaction fees.
// The governance lane accepts any transaction. The governance lane builds and verifies blocks
// in a similar fashion to how the CometBFT/Tendermint consensus engine builds and verifies
// blocks pre SDK version 0.47.0.
func NewGovernanceLane(exchangeKeeper *exchangekeeper.Keeper, cfg skipbase.LaneConfig) *skipbase.BaseLane {
	governanceMatchHandler := GovernanceMatchHandler{
		ExchangeKeeper:  exchangeKeeper,
		SignerExtractor: cfg.SignerExtractor,
	}

	options := []skipbase.LaneOption{
		skipbase.WithMatchHandler(governanceMatchHandler.GovernanceMatchHandler()),
	}

	lane, err := skipbase.NewBaseLane(
		cfg,
		LaneName,
		options...,
	)
	if err != nil {
		panic(err)
	}

	return lane
}
