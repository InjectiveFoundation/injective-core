package testutil

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app"
)

// Commit commits a block at a given time. Reminder: At the end of each
// Tendermint Consensus round the following methods are run
//  1. BeginBlock
//  2. DeliverTx
//  3. EndBlock
//  4. Commit
func Commit(ctx sdk.Context, injApp *app.InjectiveApp, t time.Duration, vs *cmttypes.ValidatorSet) (sdk.Context, error) {
	header := ctx.BlockHeader()
	req := abci.FinalizeBlockRequest{Height: header.Height}

	if vs != nil {
		res, err := injApp.FinalizeBlock(&req)
		if err != nil {
			return ctx, err
		}

		nextVals, err := applyValSetChanges(vs, res.ValidatorUpdates)
		if err != nil {
			return ctx, err
		}
		header.ValidatorsHash = vs.Hash()
		header.NextValidatorsHash = nextVals.Hash()
	} else {
		if _, err := injApp.EndBlocker(ctx); err != nil {
			return ctx, err
		}
	}

	if _, err := injApp.Commit(); err != nil {
		return ctx, err
	}

	header.Height++
	header.Time = header.Time.Add(t)
	header.AppHash = injApp.LastCommitID().Hash

	if _, err := injApp.BeginBlocker(ctx); err != nil {
		return ctx, err
	}

	return ctx.WithBlockHeader(header), nil
}

// applyValSetChanges takes in cmtypes.ValidatorSet and []abci.ValidatorUpdate and will return a new cmtypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func applyValSetChanges(valSet *cmttypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) (*cmttypes.ValidatorSet, error) {
	updates, err := cmttypes.PB2TM.ValidatorUpdates(valUpdates)
	if err != nil {
		return nil, err
	}

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	if err != nil {
		return nil, err
	}

	return newVals, nil
}
