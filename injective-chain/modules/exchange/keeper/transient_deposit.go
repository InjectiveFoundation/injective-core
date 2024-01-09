package keeper

import (
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// SetTransientDeposit sets a subaccount's deposit in the transient store for a given denom.
func (k *Keeper) SetTransientDeposit(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	deposit *types.Deposit,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	key := types.GetDepositKey(subaccountID, denom)
	bz := k.cdc.MustMarshal(deposit)
	store.Set(key, bz)
}

// EmitAllTransientDepositUpdates emits the EventDepositUpdate events for all of the deposit updates.
func (k *Keeper) EmitAllTransientDepositUpdates(
	ctx sdk.Context,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	depositStore := prefix.NewStore(store, types.DepositsPrefix)

	iterator := depositStore.Iterator(nil, nil)
	defer iterator.Close()

	subaccountDeposits := make(map[string][]*types.SubaccountDeposit)

	denoms := make([]string, 0)

	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &deposit)

		subaccountID, denom := types.ParseDepositStoreKey(iterator.Key())

		subaccountDeposit := &types.SubaccountDeposit{
			SubaccountId: subaccountID.Bytes(),
			Deposit:      &deposit,
		}
		if _, ok := subaccountDeposits[denom]; ok {
			subaccountDeposits[denom] = append(subaccountDeposits[denom], subaccountDeposit)
		} else {
			subaccountDeposits[denom] = []*types.SubaccountDeposit{subaccountDeposit}
			denoms = append(denoms, denom)
		}
	}

	if len(denoms) > 0 {
		depositUpdates := make([]*types.DepositUpdate, len(denoms))

		for idx, denom := range denoms {
			depositUpdates[idx] = &types.DepositUpdate{
				Denom:    denom,
				Deposits: subaccountDeposits[denom],
			}
		}

		depositBatchUpdateEvent := types.EventBatchDepositUpdate{
			DepositUpdates: depositUpdates,
		}

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&depositBatchUpdateEvent)
	}
}
