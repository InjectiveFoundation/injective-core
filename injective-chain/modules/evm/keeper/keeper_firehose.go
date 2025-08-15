package keeper

import (
	"math/big"

	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/tracing"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cosmostypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func BlocksBloom(k *Keeper, ctx sdk.Context) *big.Int {
	store := prefix.NewObjStore(ctx.ObjectStore(k.objectKey), types.KeyPrefixObjectBloom)
	it := store.Iterator(nil, nil)
	defer it.Close()

	bloom := new(big.Int)
	for ; it.Valid(); it.Next() {
		bloom.Or(bloom, it.Value().(*big.Int))
	}
	return bloom
}

func ToCosmosStartBlockEvent(k *Keeper, ctx sdk.Context, coinbaseAddr common.Address, blockHeader cmtproto.Header) tracing.CosmosStartBlockEvent {
	// ignore the errors as we are sure that the block header is valid
	h, _ := cosmostypes.HeaderFromProto(&blockHeader)
	h.ValidatorsHash = ctx.CometInfo().GetValidatorsHash()

	gasLimit := uint64(ctx.ConsensusParams().Block.MaxGas)

	finalizedHeaderNumber := h.Height - 1
	if h.Height == 0 {
		finalizedHeaderNumber = 0
	}

	finalizedHeader := &ethtypes.Header{
		Number: big.NewInt(finalizedHeaderNumber),
	}

	return tracing.CosmosStartBlockEvent{
		CosmosHeader: &h,
		GasLimit:     gasLimit,
		Coinbase:     coinbaseAddr,
		Finalized:    finalizedHeader,
	}
}

func ToCosmosEndBlockEvent(k *Keeper, ctx sdk.Context) tracing.CosmosEndBlockEvent {
	return tracing.CosmosEndBlockEvent{
		LogsBloom: BlocksBloom(k, ctx).Bytes(),
	}
}
