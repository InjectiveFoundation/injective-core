package keeper

import (
	"bytes"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

func (k *Keeper) SetContract(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	contract types.RegisteredContract,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	store := k.getStore(ctx)

	indexKey := types.GetContractsIndexKey(contractAddress)
	gasBz := store.Get(indexKey)
	var oldContractsKey []byte
	if gasBz != nil {
		oldGasPrice := sdk.BigEndianToUint64(gasBz)
		oldContractsKey = types.GetContractsByGasPriceKey(oldGasPrice, contractAddress)
	}

	// set main contracts store
	gasPrice := uint64(0)
	if contract.IsExecutable {
		gasPrice = contract.GasPrice
	}
	contractsKey := types.GetContractsByGasPriceKey(gasPrice, contractAddress)
	if oldContractsKey != nil && !bytes.Equal(contractsKey, oldContractsKey) {
		store.Delete(oldContractsKey)
	}

	bz := k.cdc.MustMarshal(&contract)
	store.Set(contractsKey, bz)

	// set index
	store.Set(indexKey, sdk.Uint64ToBigEndian(gasPrice))
}

func (k *Keeper) DeleteContract(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	store := k.getStore(ctx)

	indexKey := types.GetContractsIndexKey(contractAddress)
	gasBz := store.Get(indexKey)
	if gasBz == nil {
		return
	}
	gasPrice := sdk.BigEndianToUint64(gasBz)

	contractsKey := types.GetContractsByGasPriceKey(gasPrice, contractAddress)

	store.Delete(contractsKey)
	store.Delete(indexKey)
}

func (k *Keeper) GetContractByAddress(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
) *types.RegisteredContract {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	indexKey := types.GetContractsIndexKey(contractAddress)
	gasBz := store.Get(indexKey)
	if gasBz == nil {
		return nil
	}
	gasPrice := sdk.BigEndianToUint64(gasBz)

	contractsKey := types.GetContractsByGasPriceKey(gasPrice, contractAddress)

	bz := store.Get(contractsKey)
	if bz == nil {
		return nil
	}

	var contract types.RegisteredContract
	k.cdc.MustUnmarshal(bz, &contract)
	return &contract
}

func (k *Keeper) IterateContractsByGasPrice(
	ctx sdk.Context,
	minGasPrice uint64,
	callback func(contractAddress sdk.AccAddress, contractInfo types.RegisteredContract) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	store := k.getStore(ctx)
	contractsStore := prefix.NewStore(store, types.ContractsByGasPricePrefix)
	startKey := sdk.Uint64ToBigEndian(minGasPrice)
	iter := contractsStore.ReverseIterator(startKey, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		bz := iter.Value()
		var contract types.RegisteredContract

		k.cdc.MustUnmarshal(bz, &contract)

		addrBytes := iter.Key()[8:] // first 8 bytes is uint64 gas price
		contractAddr := sdk.AccAddress(addrBytes)
		if callback(contractAddr, contract) {
			break
		}
	}
}

func (k *Keeper) GetAllRegisteredContracts(
	ctx sdk.Context,
) []types.RegisteredContractWithAddress {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	allContracts := make([]types.RegisteredContractWithAddress, 0)

	k.IterateContractsByGasPrice(ctx, 0, func(contractAddress sdk.AccAddress, contractInfo types.RegisteredContract) (stop bool) {
		contractWithAddress := types.RegisteredContractWithAddress{
			Address:            contractAddress.String(),
			RegisteredContract: &contractInfo,
		}
		allContracts = append(allContracts, contractWithAddress)
		return false
	})

	return allContracts
}
