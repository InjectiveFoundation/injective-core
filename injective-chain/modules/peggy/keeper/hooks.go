package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Wrapper struct
type Hooks struct {
	k *Keeper

	svcTags metrics.Tags
}

func NewHooks(keeper *Keeper) Hooks {
	return Hooks{
		k: keeper,
		svcTags: metrics.Tags{
			"svc": "peggy_hooks",
		},
	}
}

var _ stakingtypes.StakingHooks = Hooks{}

// Create new peggy hooks
func (k *Keeper) Hooks() Hooks { return NewHooks(k) }

func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	metrics.ReportFuncCall(h.svcTags)
	doneFn := metrics.ReportFuncTiming(h.svcTags)
	defer doneFn()

	// When Validator starts Unbonding, Persist the block height in the store
	// Later in endblocker, check if there is atleast one validator who started unbonding and create a valset request.
	// The reason for creating valset requests in endblock is to create only one valset request per block if multiple validators starts unbonding at same block.

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	h.k.SetLastUnbondingBlockHeight(sdkCtx, uint64(sdkCtx.BlockHeight()))

	return nil
}

func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, validator sdk.ValAddress) error {
	metrics.ReportFuncCall(h.svcTags)
	defer metrics.ReportFuncTiming(h.svcTags)()

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	ev := h.k.GetLastEventByValidator(sdkCtx, validator)

	isFirstTimeValidator := ev.EthereumEventHeight == 0 && ev.EthereumEventNonce == 0
	if !isFirstTimeValidator {
		// no need to do anything, not a first time validator
		return nil
	}

	lowestObservedNonce := h.k.GetLastObservedEventNonce(sdkCtx)

	// when the chain starts from genesis state, as there are no events broadcasted, lowest_observed_nonce will be zero.
	// Bridge relayer has to scan the events from the height at which bridge contract is deployed on ethereum.
	// So, if lowest_observed_nonce is zero, we don't need to do anything.
	if lowestObservedNonce == 0 {
		return nil
	}

	// otherwise, set the last event to the current last observed event nonce and ethereum block height so the validator
	// can begin attesting starting from the next event after the last observed event nonce.
	h.k.setLastEventByValidator(
		sdkCtx,
		validator,
		lowestObservedNonce,
		h.k.GetLastObservedEthereumBlockHeight(sdkCtx).EthereumBlockHeight,
	)

	return nil
}

func (h Hooks) AfterValidatorCreated(ctx context.Context, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeValidatorModified(ctx context.Context, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorRemoved(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	return nil
}

func (h Hooks) AfterUnbondingInitiated(ctx context.Context, id uint64) error {
	return nil
}
