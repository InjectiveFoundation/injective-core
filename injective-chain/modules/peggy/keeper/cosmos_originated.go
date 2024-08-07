package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

func (k *Keeper) GetCosmosOriginatedDenom(ctx sdk.Context, tokenContract common.Address) (string, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetERC20ToCosmosDenomKey(tokenContract))
	if bz == nil {
		return "", false
	}

	return string(bz), true
}

func (k *Keeper) GetCosmosOriginatedERC20(ctx sdk.Context, denom string) (common.Address, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetCosmosDenomToERC20Key(denom))
	if bz == nil {
		return common.Address{}, false
	}

	return common.BytesToAddress(bz), true
}

func (k *Keeper) SetCosmosOriginatedDenomToERC20(ctx sdk.Context, denom string, tokenContract common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetCosmosDenomToERC20Key(denom), tokenContract.Bytes())
	store.Set(types.GetERC20ToCosmosDenomKey(tokenContract), []byte(denom))
}

// DenomToERC20 returns if an asset is native to Cosmos or Ethereum, and get its corresponding ERC20 address
// This will return an error if it cant parse the denom as a peggy denom, and then also can't find the denom
// in an index of ERC20 contracts deployed on Ethereum to serve as synthetic Cosmos assets.
func (k *Keeper) DenomToERC20Lookup(ctx sdk.Context, denomStr string) (isCosmosOriginated bool, tokenContract common.Address, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// First try parsing the ERC20 out of the denom
	peggyDenom, denomErr := types.NewPeggyDenomFromString(denomStr)
	if denomErr == nil {
		// This is an Ethereum-originated asset
		tokenContractFromDenom, _ := peggyDenom.TokenContract()
		return false, tokenContractFromDenom, nil
	}

	// If denom is native cosmos coin denom, return Cosmos coin ERC20 contract address from Params
	if denomStr == k.GetCosmosCoinDenom(ctx) {
		// isCosmosOriginated assumed to be false, since the native cosmos coin
		// expected to be mapped from Ethereum mainnet in first place, i.e. its origin
		// is still from Ethereum.
		return false, k.GetCosmosCoinERC20Contract(ctx), nil
	}

	// Look up ERC20 contract in index and error if it's not in there
	tokenContract, exists := k.GetCosmosOriginatedERC20(ctx, denomStr)
	if !exists {
		err = errors.Errorf(
			"denom (%s) not a peggy voucher coin (parse error: %s), and also not in cosmos-originated ERC20 index",
			denomStr, denomErr.Error(),
		)

		metrics.ReportFuncError(k.svcTags)
		return false, common.Address{}, err
	}

	isCosmosOriginated = true
	return isCosmosOriginated, tokenContract, nil
}

// RewardToERC20Lookup is a specialized function wrapping DenomToERC20Lookup designed to validate
// the validator set reward any time we generate a validator set
func (k *Keeper) RewardToERC20Lookup(ctx sdk.Context, coin sdk.Coin) (common.Address, math.Int) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if coin.Denom == "" || coin.Amount.BigInt() == nil || coin.Amount == math.NewInt(0) {
		metrics.ReportFuncError(k.svcTags)
		panic("Bad validator set relaying reward!")
	} else {
		// reward case, pass to DenomToERC20Lookup
		_, addressStr, err := k.DenomToERC20Lookup(ctx, coin.Denom)
		if err != nil {
			// This can only ever happen if governance sets a value for the reward
			// which is not a valid ERC20 that as been bridged before (either from or to Cosmos)
			// We'll classify that as operator error and just panic
			metrics.ReportFuncError(k.svcTags)
			panic("Invalid Valset reward! Correct or remove the paramater value")
		}
		err = types.ValidateEthAddress(addressStr.Hex())
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			panic("Invalid Valset reward! Correct or remove the paramater value")
		}
		return addressStr, coin.Amount
	}
}

// Validate if claim data can be unmarshalled into authorised sdk msgs
// Enfore if it is signed by valid signer
// Enfore if validateBasic is successful.
func (k *Keeper) ValidateClaimData(ctx sdk.Context, claimData string, ethereumSigner sdk.AccAddress) (msg sdk.Msg, err error) {
	// Check if the claim data is a valid sdk msg
	if err := k.cdc.UnmarshalInterfaceJSON([]byte(claimData), &msg); err != nil {
		return msg, errors.Errorf("claim data is not a valid sdk msg: %s", err.Error())
	}

	message, ok := msg.(sdk.HasValidateBasic)
	if !ok {
		return msg, errors.Errorf("claim data is not a valid sdk.Msg: %v", err)
	}

	// Enforce that msg.ValidateBasic() succeeds
	if err := message.ValidateBasic(); err != nil {
		return msg, errors.Errorf("claim data is not a valid sdk.Msg: %v", err)
	}

	legacyMsg, ok := msg.(sdk.LegacyMsg)
	if !ok {
		return msg, errors.Errorf("claim data is not a valid sdk.Msg: %v", err)
	}
	// Enforce that the claim data is signed by the ethereum signer
	if !legacyMsg.GetSigners()[0].Equals(ethereumSigner) {
		return msg, errors.Errorf("claim data is not signed by ethereum signer: %s", ethereumSigner.String())
	}

	return msg, nil
}

// ERC20ToDenom returns if an ERC20 address represents an asset is native to Cosmos or Ethereum,
// and get its corresponding peggy denom.
func (k *Keeper) ERC20ToDenomLookup(ctx sdk.Context, tokenContract common.Address) (isCosmosOriginated bool, denom string) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// First try looking up tokenContract in index
	denomStr, exists := k.GetCosmosOriginatedDenom(ctx, tokenContract)
	if exists {
		isCosmosOriginated = true
		return isCosmosOriginated, denomStr
	} else if tokenContract == k.GetCosmosCoinERC20Contract(ctx) {
		return false, k.GetCosmosCoinDenom(ctx)
	}

	// If it is not in there, it is not a cosmos originated token, turn the ERC20 into a peggy denom
	return false, types.NewPeggyDenom(tokenContract).String()
}

// IterateERC20ToDenom iterates over erc20 to denom relations
func (k *Keeper) IterateERC20ToDenom(ctx sdk.Context, cb func(k []byte, v *types.ERC20ToDenom) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ERC20ToDenomKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		erc20ToDenom := types.ERC20ToDenom{
			Erc20: common.BytesToAddress(iter.Key()).Hex(),
			Denom: string(iter.Value()),
		}

		if cb(iter.Key(), &erc20ToDenom) {
			break
		}
	}
}
