package keeper

import (
	"math"
	"sort"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// Keeper maintains the link to storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc      codec.Codec         // The wire codec for binary encoding/decoding.
	storeKey storetypes.StoreKey // Unexposed key to access store from sdk.Context

	StakingKeeper     types.StakingKeeper
	bankKeeper        types.BankKeeper
	DistKeeper        types.DistributionKeeper
	SlashingKeeper    types.SlashingKeeper
	exchangeMsgServer exchangetypes.MsgServer

	AttestationHandler interface {
		Handle(sdk.Context, types.EthereumClaim) error
	}

	svcTags  metrics.Tags
	grpcTags metrics.Tags

	// address authorized to execute MsgUpdateParams. Default: gov module
	authority string
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// NewKeeper returns a new instance of the peggy keeper
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	stakingKeeper types.StakingKeeper,
	bankKeeper types.BankKeeper,
	slashingKeeper types.SlashingKeeper,
	distKeeper types.DistributionKeeper,
	exchangeKeeper exchangekeeper.Keeper,
	authority string,
) Keeper {
	exchangeMsgServer := exchangekeeper.NewMsgServerImpl(exchangeKeeper)

	k := Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		StakingKeeper:     stakingKeeper,
		bankKeeper:        bankKeeper,
		DistKeeper:        distKeeper,
		SlashingKeeper:    slashingKeeper,
		exchangeMsgServer: exchangeMsgServer,
		authority:         authority,
		svcTags: metrics.Tags{
			"svc": "peggy_k",
		},
		grpcTags: metrics.Tags{
			"svc": "peggy_grpc",
		},
	}

	k.AttestationHandler = NewAttestationHandler(bankKeeper, k)

	return k
}

/////////////////////////////
//     VALSET REQUESTS     //
/////////////////////////////

// SetValsetRequest returns a new instance of the Peggy BridgeValidatorSet
// i.e. {"nonce": 1, "memebers": [{"eth_addr": "foo", "power": 11223}]}
func (k *Keeper) SetValsetRequest(ctx sdk.Context) *types.Valset {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	valset := k.GetCurrentValset(ctx)

	// If none of the bonded validators has registered eth key, then valset.Members = 0.
	if len(valset.Members) == 0 {
		return nil
	}

	k.StoreValset(ctx, valset)
	// Store the checkpoint as a legit past valset
	checkpoint := valset.GetCheckpoint(k.GetPeggyID(ctx))
	k.SetPastEthSignatureCheckpoint(ctx, checkpoint)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventValsetUpdateRequest{
		ValsetNonce:   valset.Nonce,
		ValsetHeight:  valset.Height,
		ValsetMembers: valset.Members,
		RewardAmount:  valset.RewardAmount,
		RewardToken:   valset.RewardToken,
	})

	return valset
}

// StoreValset is for storing a valiator set at a given height
func (k *Keeper) StoreValset(ctx sdk.Context, valset *types.Valset) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	valset.Height = uint64(ctx.BlockHeight())
	store.Set(types.GetValsetKey(valset.Nonce), k.cdc.MustMarshal(valset))
	k.SetLatestValsetNonce(ctx, valset.Nonce)
}

// SetLatestValsetNonce sets the latest valset nonce
func (k *Keeper) SetLatestValsetNonce(ctx sdk.Context, nonce uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LatestValsetNonce, types.UInt64Bytes(nonce))
}

// StoreValsetUnsafe is for storing a valiator set at a given height
func (k *Keeper) StoreValsetUnsafe(ctx sdk.Context, valset *types.Valset) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetValsetKey(valset.Nonce), k.cdc.MustMarshal(valset))
	k.SetLatestValsetNonce(ctx, valset.Nonce)
}

// HasValsetRequest returns true if a valset defined by a nonce exists
func (k *Keeper) HasValsetRequest(ctx sdk.Context, nonce uint64) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GetValsetKey(nonce))
}

// DeleteValset deletes the valset at a given nonce from state
func (k *Keeper) DeleteValset(ctx sdk.Context, nonce uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx.KVStore(k.storeKey).Delete(types.GetValsetKey(nonce))
}

// GetLatestValsetNonce returns the latest valset nonce
func (k *Keeper) GetLatestValsetNonce(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.LatestValsetNonce)

	if len(bytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(bytes)
}

// GetValset returns a valset by nonce
func (k *Keeper) GetValset(ctx sdk.Context, nonce uint64) *types.Valset {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetValsetKey(nonce))
	if bz == nil {
		return nil
	}

	var valset types.Valset
	k.cdc.MustUnmarshal(bz, &valset)

	return &valset
}

// IterateValsets retruns all valsetRequests
func (k *Keeper) IterateValsets(ctx sdk.Context, cb func(key []byte, val *types.Valset) bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetRequestKey)
	iter := prefixStore.ReverseIterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var valset types.Valset
		k.cdc.MustUnmarshal(iter.Value(), &valset)
		// cb returns true to stop early
		if cb(iter.Key(), &valset) {
			break
		}
	}
}

// GetValsets returns all the validator sets in state
func (k *Keeper) GetValsets(ctx sdk.Context) (out []*types.Valset) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.IterateValsets(ctx, func(_ []byte, val *types.Valset) bool {
		out = append(out, val)
		return false
	})

	sort.Sort(types.Valsets(out))

	return
}

// GetLatestValset returns the latest validator set in state
func (k *Keeper) GetLatestValset(ctx sdk.Context) (out *types.Valset) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	latestValsetNonce := k.GetLatestValsetNonce(ctx)
	out = k.GetValset(ctx, latestValsetNonce)

	return
}

// setLastSlashedValsetNonce sets the latest slashed valset nonce
func (k *Keeper) SetLastSlashedValsetNonce(ctx sdk.Context, nonce uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LastSlashedValsetNonce, types.UInt64Bytes(nonce))
}

// GetLastSlashedValsetNonce returns the latest slashed valset nonce
func (k *Keeper) GetLastSlashedValsetNonce(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.LastSlashedValsetNonce)

	if len(bytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(bytes)
}

// SetLastUnbondingBlockHeight sets the last unbonding block height
func (k *Keeper) SetLastUnbondingBlockHeight(ctx sdk.Context, unbondingBlockHeight uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LastUnbondingBlockHeight, types.UInt64Bytes(unbondingBlockHeight))
}

// GetLastUnbondingBlockHeight returns the last unbonding block height
func (k *Keeper) GetLastUnbondingBlockHeight(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.LastUnbondingBlockHeight)

	if len(bytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(bytes)
}

// GetUnslashedValsets returns all the unslashed validator sets in state
func (k *Keeper) GetUnslashedValsets(ctx sdk.Context, maxHeight uint64) (out []*types.Valset) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	lastSlashedValsetNonce := k.GetLastSlashedValsetNonce(ctx)

	k.IterateValsetBySlashedValsetNonce(ctx, lastSlashedValsetNonce, maxHeight, func(_ []byte, valset *types.Valset) bool {
		if valset.Nonce > lastSlashedValsetNonce {
			out = append(out, valset)
		}
		return false
	})

	return
}

// IterateValsetBySlashedValsetNonce iterates through all valset by last slashed valset nonce in ASC order
func (k *Keeper) IterateValsetBySlashedValsetNonce(
	ctx sdk.Context,
	lastSlashedValsetNonce uint64,
	maxHeight uint64,
	cb func(k []byte, v *types.Valset) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetRequestKey)
	iter := prefixStore.Iterator(types.UInt64Bytes(lastSlashedValsetNonce), types.UInt64Bytes(maxHeight))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		valset := types.Valset{}
		k.cdc.MustUnmarshal(iter.Value(), &valset)

		if cb(iter.Key(), &valset) {
			break
		}
	}
}

/////////////////////////////
//     VALSET CONFIRMS     //
/////////////////////////////

// GetValsetConfirm returns a valset confirmation by a nonce and validator address
func (k *Keeper) GetValsetConfirm(ctx sdk.Context, nonce uint64, validator sdk.AccAddress) *types.MsgValsetConfirm {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	entity := store.Get(types.GetValsetConfirmKey(nonce, validator))
	if entity == nil {
		return nil
	}

	valset := types.MsgValsetConfirm{}
	k.cdc.MustUnmarshal(entity, &valset)

	return &valset
}

// SetValsetConfirm sets a valset confirmation
func (k *Keeper) SetValsetConfirm(ctx sdk.Context, valset *types.MsgValsetConfirm) []byte {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	addr, err := sdk.AccAddressFromBech32(valset.Orchestrator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic(err)
	}

	key := types.GetValsetConfirmKey(valset.Nonce, addr)
	store.Set(key, k.cdc.MustMarshal(valset))

	return key
}

// GetValsetConfirms returns all validator set confirmations by nonce
func (k *Keeper) GetValsetConfirms(ctx sdk.Context, nonce uint64) (valsetConfirms []*types.MsgValsetConfirm) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetConfirmKey)
	start, end := PrefixRange(types.UInt64Bytes(nonce))
	iterator := prefixStore.Iterator(start, end)

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		valset := types.MsgValsetConfirm{}

		k.cdc.MustUnmarshal(iterator.Value(), &valset)
		valsetConfirms = append(valsetConfirms, &valset)
	}

	return valsetConfirms
}

// IterateValsetConfirmByNonce iterates through all valset confirms by validator set nonce in ASC order
func (k *Keeper) IterateValsetConfirmByNonce(ctx sdk.Context, nonce uint64, cb func(k []byte, v *types.MsgValsetConfirm) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetConfirmKey)
	iter := prefixStore.Iterator(PrefixRange(types.UInt64Bytes(nonce)))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		valset := types.MsgValsetConfirm{}
		k.cdc.MustUnmarshal(iter.Value(), &valset)

		if cb(iter.Key(), &valset) {
			break
		}
	}
}

/////////////////////////////
//      BATCH CONFIRMS     //
/////////////////////////////

// GetBatchConfirm returns a batch confirmation given its nonce, the token contract, and a validator address
func (k *Keeper) GetBatchConfirm(ctx sdk.Context, nonce uint64, tokenContract common.Address, validator sdk.AccAddress) *types.MsgConfirmBatch {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	entity := store.Get(types.GetBatchConfirmKey(tokenContract, nonce, validator))
	if entity == nil {
		return nil
	}

	batch := types.MsgConfirmBatch{}
	k.cdc.MustUnmarshal(entity, &batch)

	return &batch
}

// SetBatchConfirm sets a batch confirmation by a validator
func (k *Keeper) SetBatchConfirm(ctx sdk.Context, batch *types.MsgConfirmBatch) []byte {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// convert eth signer to hex string lol
	batch.EthSigner = common.HexToAddress(batch.EthSigner).Hex()
	tokenContract := common.HexToAddress(batch.TokenContract)
	store := ctx.KVStore(k.storeKey)

	acc, err := sdk.AccAddressFromBech32(batch.Orchestrator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic(err)
	}

	key := types.GetBatchConfirmKey(tokenContract, batch.Nonce, acc)
	store.Set(key, k.cdc.MustMarshal(batch))

	return key
}

// IterateBatchConfirmByNonceAndTokenContract iterates through all batch confirmations
func (k *Keeper) IterateBatchConfirmByNonceAndTokenContract(
	ctx sdk.Context,
	nonce uint64,
	tokenContract common.Address,
	cb func(k []byte, v *types.MsgConfirmBatch) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.BatchConfirmKey)
	batchPrefix := append(tokenContract.Bytes(), types.UInt64Bytes(nonce)...)
	iter := prefixStore.Iterator(PrefixRange(batchPrefix))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		confirm := types.MsgConfirmBatch{}
		k.cdc.MustUnmarshal(iter.Value(), &confirm)

		if cb(iter.Key(), &confirm) {
			break
		}
	}
}

// GetBatchConfirmByNonceAndTokenContract returns the batch confirms
func (k *Keeper) GetBatchConfirmByNonceAndTokenContract(ctx sdk.Context, nonce uint64, tokenContract common.Address) (out []*types.MsgConfirmBatch) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.IterateBatchConfirmByNonceAndTokenContract(ctx, nonce, tokenContract, func(_ []byte, msg *types.MsgConfirmBatch) (stop bool) {
		out = append(out, msg)
		return false
	})

	return
}

/////////////////////////////
//    ADDRESS DELEGATION   //
/////////////////////////////

// SetOrchestratorValidator sets the Orchestrator key for a given validator
func (k *Keeper) SetOrchestratorValidator(ctx sdk.Context, val sdk.ValAddress, orch sdk.AccAddress) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetOrchestratorAddressKey(orch), val.Bytes())
}

// GetOrchestratorValidator returns the validator key associated with an orchestrator key
func (k *Keeper) GetOrchestratorValidator(ctx sdk.Context, orch sdk.AccAddress) (sdk.ValAddress, bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetOrchestratorAddressKey(orch))
	if bz == nil {
		return nil, false
	}

	return sdk.ValAddress(bz), true
}

/////////////////////////////
//       ETH ADDRESS       //
/////////////////////////////

// SetEthAddressForValidator sets the ethereum address for a given validator
func (k *Keeper) SetEthAddressForValidator(ctx sdk.Context, validator sdk.ValAddress, ethAddr common.Address) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetEthAddressByValidatorKey(validator), ethAddr.Bytes())
	store.Set(types.GetValidatorByEthAddressKey(ethAddr), validator.Bytes())
}

// GetEthAddressByValidator returns the eth address for a given peggy validator
func (k *Keeper) GetEthAddressByValidator(ctx sdk.Context, validator sdk.ValAddress) (common.Address, bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetEthAddressByValidatorKey(validator))
	if bz == nil {
		return common.Address{}, false
	}

	return common.BytesToAddress(bz), true
}

// GetValidatorByEthAddress returns the validator for a given eth address
func (k *Keeper) GetValidatorByEthAddress(ctx sdk.Context, ethAddr common.Address) (validator stakingtypes.Validator, found bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	valAddr := store.Get(types.GetValidatorByEthAddressKey(ethAddr))
	if valAddr == nil {
		return stakingtypes.Validator{}, false
	}
	validator, found = k.StakingKeeper.GetValidator(ctx, valAddr)
	if !found {
		return stakingtypes.Validator{}, false
	}

	return validator, true
}

// GetCurrentValset gets powers from the store and normalizes them
// into an integer percentage with a resolution of uint32 Max meaning
// a given validators 'Peggy power' is computed as
// Cosmos power for that validator / total cosmos power = x / uint32 Max
// where x is the voting power on the Peggy contract. This allows us
// to only use integer division which produces a known rounding error
// from truncation equal to the ratio of the validators
// Cosmos power / total cosmos power ratio, leaving us at uint32 Max - 1
// total voting power. This is an acceptable rounding error since floating
// point may cause consensus problems if different floating point unit
// implementations are involved.
//
// 'total cosmos power' has an edge case, if a validator has not set their
// Ethereum key they are not included in the total. If they where control
// of the bridge could be lost in the following situation.
//
// If we have 100 total power, and 100 total power joins the validator set
// the new validators hold more than 33% of the bridge power, if we generate
// and submit a valset and they don't have their eth keys set they can never
// update the validator set again and the bridge and all its' funds are lost.
// For this reason we exclude validators with unset eth keys from validator sets
func (k *Keeper) GetCurrentValset(ctx sdk.Context) *types.Valset {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	validators := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
	// allocate enough space for all validators, but len zero, we then append
	// so that we have an array with extra capacity but the correct length depending
	// on how many validators have keys set.
	bridgeValidators := make([]*types.BridgeValidator, 0, len(validators))
	var totalPower uint64
	for i := range validators {
		val := validators[i].GetOperator()
		p := uint64(k.StakingKeeper.GetLastValidatorPower(ctx, val))

		if ethAddress, found := k.GetEthAddressByValidator(ctx, val); found {
			bv := &types.BridgeValidator{Power: p, EthereumAddress: ethAddress.Hex()}
			bridgeValidators = append(bridgeValidators, bv)
			totalPower += p
		}
	}

	// normalize power values
	for i := range bridgeValidators {
		bridgeValidators[i].Power = sdk.NewUint(bridgeValidators[i].Power).MulUint64(math.MaxUint32).QuoUint64(totalPower).Uint64()
	}

	// get the reward from the params store
	reward := k.GetParams(ctx).ValsetReward
	var rewardToken common.Address
	var rewardAmount sdkmath.Int
	if reward.Denom == "" {
		// the case where a validator has 'no reward'. The 'no reward' value is interpreted as having a zero
		// address for the ERC20 token and a zero value for the reward amount. Since we store a coin with the
		// params, a coin with a blank denom and/or zero amount is interpreted in this way.
		rewardToken = common.Address{0x0000000000000000000000000000000000000000}
		rewardAmount = sdk.NewIntFromUint64(0)

	} else {
		rewardToken, rewardAmount = k.RewardToERC20Lookup(ctx, reward)
	}
	// TODO: make the nonce an incrementing one (i.e. fetch last nonce from state, increment, set here)
	return types.NewValset(uint64(ctx.BlockHeight()), uint64(ctx.BlockHeight()), bridgeValidators, rewardAmount, rewardToken)
}

/////////////////////////////
//       HELPERS           //
/////////////////////////////

func (k *Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

// SendToCommunityPool handles incorrect SendToCosmos calls to the community pool, since the calls
// have already been made on Ethereum there's nothing we can do to reverse them, and we should at least
// make use of the tokens which would otherwise be lost
func (k Keeper) SendToCommunityPool(ctx sdk.Context, coins sdk.Coins) error {
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, distrtypes.ModuleName, coins); err != nil {
		return errors.Wrap(err, "transfer to community pool failed")
	}
	feePool := k.DistKeeper.GetFeePool(ctx)
	feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(coins...)...)
	k.DistKeeper.SetFeePool(ctx, feePool)
	return nil
}

/////////////////////////////
//       PARAMS        //
/////////////////////////////

// GetParams returns the parameters from the store
func (k *Keeper) GetParams(ctx sdk.Context) *types.Params {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.ParamKey)
	if bz == nil {
		return nil
	}

	params := &types.Params{}
	k.cdc.MustUnmarshal(bz, params)

	return params
}

// SetParams sets the parameters in the store
func (k *Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(params)
	store.Set(types.ParamKey, bz)
}

// GetBridgeContractAddress returns the bridge contract address on ETH
func (k *Keeper) GetBridgeContractAddress(ctx sdk.Context) common.Address {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	params := k.GetParams(ctx)
	if params == nil {
		return common.Address{}
	}

	return common.HexToAddress(params.BridgeEthereumAddress)
}

// GetBridgeChainID returns the chain id of the ETH chain we are running against
func (k *Keeper) GetBridgeChainID(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	params := k.GetParams(ctx)
	if params == nil {
		return 0
	}

	return params.BridgeChainId
}

func (k *Keeper) GetPeggyID(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	params := k.GetParams(ctx)
	if params == nil {
		return ""
	}

	return params.PeggyId
}

// GetCosmosCoinDenom returns native cosmos coin denom
func (k *Keeper) GetCosmosCoinDenom(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	params := k.GetParams(ctx)
	if params == nil {
		return ""
	}

	return params.CosmosCoinDenom
}

// GetCosmosCoinERC20Contract returns the Cosmos coin ERC20 contract address
func (k *Keeper) GetCosmosCoinERC20Contract(ctx sdk.Context) common.Address {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	params := k.GetParams(ctx)
	if params == nil {
		return common.Address{}
	}

	return common.HexToAddress(params.CosmosCoinErc20Contract)
}

func (k *Keeper) UnpackAttestationClaim(attestation *types.Attestation) (types.EthereumClaim, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var msg types.EthereumClaim

	err := k.cdc.UnpackAny(attestation.Claim, &msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		err = errors.Wrap(err, "failed to unpack EthereumClaim")
		return nil, err
	} else {
		return msg, nil
	}
}

// GetOrchestratorAddresses iterates both the EthAddress and Orchestrator address indexes to produce
// a vector of MsgSetOrchestratorAddresses entires containing all the delgate keys for state
// export / import. This may seem at first glance to be excessively complicated, why not combine
// the EthAddress and Orchestrator address indexes and simply iterate one thing? The answer is that
// even though we set the Eth and Orchestrator address in the same place we use them differently we
// always go from Orchestrator address to Validator address and from validator address to Ethereum address
// we want to keep looking up the validator address for various reasons, so a direct Orchestrator to Ethereum
// address mapping will mean having to keep two of the same data around just to provide lookups.
//
// For the time being this will serve
func (k *Keeper) GetOrchestratorAddresses(ctx sdk.Context) []*types.MsgSetOrchestratorAddresses {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	storePrefix := types.EthAddressByValidatorKey
	iter := store.Iterator(PrefixRange(storePrefix))
	defer iter.Close()

	ethAddresses := make(map[string]common.Address)

	for ; iter.Valid(); iter.Next() {
		// the 'key' contains both the prefix and the value, so we need
		// to cut off the starting bytes, if you don't do this a valid
		// cosmos key will be made out of EthAddressByValidatorKey + the startin bytes
		// of the actual key
		key := iter.Key()[len(types.EthAddressByValidatorKey):]
		value := iter.Value()
		ethAddress := common.BytesToAddress(value)
		validatorAccount := sdk.AccAddress(key)
		ethAddresses[validatorAccount.String()] = ethAddress
	}

	store = ctx.KVStore(k.storeKey)
	storePrefix = types.KeyOrchestratorAddress
	iter = store.Iterator(PrefixRange(storePrefix))
	defer iter.Close()

	orchestratorAddresses := make(map[string]sdk.AccAddress)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()[len(types.KeyOrchestratorAddress):]
		value := iter.Value()
		orchestratorAccount := sdk.AccAddress(key)
		validatorAccount := sdk.AccAddress(value)
		orchestratorAddresses[validatorAccount.String()] = orchestratorAccount
	}

	result := make([]*types.MsgSetOrchestratorAddresses, 0)

	for validatorAccount, ethAddress := range ethAddresses {
		orchestratorAccount, ok := orchestratorAddresses[validatorAccount]
		if !ok {
			metrics.ReportFuncError(k.svcTags)
			panic("cannot find validator account in orchestrator addresses mapping")
		}

		result = append(result, &types.MsgSetOrchestratorAddresses{
			Sender:       validatorAccount,
			Orchestrator: orchestratorAccount.String(),
			EthAddress:   ethAddress.Hex(),
		})
	}

	// we iterated over a map, so now we have to sort to ensure the
	// output here is deterministic, eth address chosen for no particular
	// reason
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].EthAddress < result[j].EthAddress
	})

	return result
}

// DeserializeValidatorIterator returns validators from the validator iterator.
// Adding here in gravity keeper as cdc is not available inside endblocker.
func (k *Keeper) DeserializeValidatorIterator(vals []byte) stakingtypes.ValAddresses {
	validators := stakingtypes.ValAddresses{}
	k.cdc.MustUnmarshal(vals, &validators)
	return validators
}

type PrefixStart []byte
type PrefixEnd []byte

// PrefixRange turns a prefix into a (start, end) range. The start is the given prefix value and
// the end is calculated by adding 1 bit to the start value. Nil is not allowed as prefix.
//
//	Example: []byte{1, 3, 4} becomes []byte{1, 3, 5}
//			 []byte{15, 42, 255, 255} becomes []byte{15, 43, 0, 0}
//
// In case of an overflow the end is set to nil.
//
//	Example: []byte{255, 255, 255, 255} becomes nil
//
// MARK finish-batches: this is where some crazy shit happens
func PrefixRange(proposedPrefix []byte) (PrefixStart, PrefixEnd) {
	if proposedPrefix == nil {
		panic("nil key not allowed")
	}

	// special case: no prefix is whole range
	if len(proposedPrefix) == 0 {
		return nil, nil
	}

	// copy the prefix and update last byte
	end := make([]byte, len(proposedPrefix))
	copy(end, proposedPrefix)
	l := len(end) - 1
	end[l]++

	// wait, what if that overflowed?....
	for end[l] == 0 && l > 0 {
		l--
		end[l]++
	}

	// okay, funny guy, you gave us FFF, no end to this range...
	if l == 0 && end[0] == 0 {
		end = nil
	}

	return proposedPrefix, end
}

// IsOnBlacklist checks that the Ethereum Address is black listed.
func (k *Keeper) IsOnBlacklist(ctx sdk.Context, addr common.Address) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.getStore(ctx).Has(types.GetEthereumBlacklistStoreKey(addr))
}

// SetEthereumBlacklistAddress sets the ethereum blacklist address.
func (k *Keeper) SetEthereumBlacklistAddress(ctx sdk.Context, addr common.Address) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// set boolean indicator
	k.getStore(ctx).Set(types.GetEthereumBlacklistStoreKey(addr), []byte{})
}

// GetAllEthereumBlacklistAddresses fetches all etheruem blacklisted addresses.
func (k *Keeper) GetAllEthereumBlacklistAddresses(ctx sdk.Context) []string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	blacklistedAddresses := make([]string, 0)
	store := ctx.KVStore(k.storeKey)
	blacklistAddressStore := prefix.NewStore(store, types.EthereumBlacklistKey)

	iterator := blacklistAddressStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		blacklistAddress := common.BytesToAddress(iterator.Key())
		blacklistedAddresses = append(blacklistedAddresses, blacklistAddress.String())
	}

	return blacklistedAddresses
}

// DeleteEthereumBlacklistAddress deletes the address from blacklist.
func (k *Keeper) DeleteEthereumBlacklistAddress(ctx sdk.Context, addr common.Address) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.getStore(ctx).Delete(types.GetEthereumBlacklistStoreKey(addr))
}

// Returns true if the provided address is invalid to send to Ethereum this could be
// for one of several reasons. (1) it is invalid in general like the Zero address, (2)
// it is invalid for a subset of ERC20 addresses or (3) it is on the governance deposit/withdraw
// blacklist. (2) is not yet implemented
// Blocking some addresses is technically motivated, if any ERC20 transfers in a batch fail the entire batch
// becomes impossible to execute.
func (k Keeper) InvalidSendToEthAddress(ctx sdk.Context, addr common.Address) bool {
	return k.IsOnBlacklist(ctx, addr) || addr == types.ZeroAddress()
}
