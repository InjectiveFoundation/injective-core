package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// SetTransientDeposit sets a subaccount's deposit in the transient store for a given denom.
func (k *Keeper) SetTransientDeposit(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
	deposit *v2.Deposit,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	key := types.GetDepositKey(subaccountID, denom)
	bz := k.cdc.MustMarshal(deposit)
	store.Set(key, bz)
}

// EmitAllTransientDepositUpdates emits the EventDepositUpdate events for all of the deposit updates.
func (k *Keeper) EmitAllTransientDepositUpdates(
	ctx sdk.Context,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	depositStore := prefix.NewStore(store, types.DepositsPrefix)

	iterator := depositStore.Iterator(nil, nil)
	defer iterator.Close()

	subaccountDeposits := make(map[string][]*v2.SubaccountDeposit)

	denoms := make([]string, 0)

	for ; iterator.Valid(); iterator.Next() {
		var deposit v2.Deposit
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &deposit)

		subaccountID, denom := types.ParseDepositStoreKey(iterator.Key())

		subaccountDeposit := &v2.SubaccountDeposit{
			SubaccountId: subaccountID.Bytes(),
			Deposit:      &deposit,
		}
		if _, ok := subaccountDeposits[denom]; ok {
			subaccountDeposits[denom] = append(subaccountDeposits[denom], subaccountDeposit)
		} else {
			subaccountDeposits[denom] = []*v2.SubaccountDeposit{subaccountDeposit}
			denoms = append(denoms, denom)
		}
	}
	iterator.Close()

	if len(denoms) > 0 {
		depositUpdates := make([]*v2.DepositUpdate, len(denoms))

		for idx, denom := range denoms {
			depositUpdates[idx] = &v2.DepositUpdate{
				Denom:    denom,
				Deposits: subaccountDeposits[denom],
			}
		}

		depositBatchUpdateEvent := v2.EventBatchDepositUpdate{
			DepositUpdates: depositUpdates,
		}

		k.EmitEvent(ctx, &depositBatchUpdateEvent)
	}
}
