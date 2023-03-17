package keeper

import (
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

func (h Hooks) AfterValidatorBeginUnbonding(ctx sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) {
	metrics.ReportFuncCall(h.svcTags)
	doneFn := metrics.ReportFuncTiming(h.svcTags)
	defer doneFn()

	// When Validator starts Unbonding, Persist the block height in the store
	// Later in endblocker, check if there is atleast one validator who started unbonding and create a valset request.
	// The reason for creating valset requests in endblock is to create only one valset request per block if multiple validators starts unbonding at same block.

	h.k.SetLastUnbondingBlockHeight(ctx, uint64(ctx.BlockHeight()))

}

func (h Hooks) BeforeDelegationCreated(_ sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
}
func (h Hooks) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) {}
func (h Hooks) BeforeValidatorModified(_ sdk.Context, _ sdk.ValAddress)       {}
func (h Hooks) AfterValidatorBonded(ctx sdk.Context, _ sdk.ConsAddress, validator sdk.ValAddress) {
	metrics.ReportFuncCall(h.svcTags)
	defer metrics.ReportFuncTiming(h.svcTags)()

	ev := h.k.GetLastEventByValidator(ctx, validator)

	isFirstTimeValidator := ev.EthereumEventHeight == 0 && ev.EthereumEventNonce == 0
	if !isFirstTimeValidator {
		// no need to do anything, not a first time validator
		return
	}

	lowestObservedNonce := h.k.GetLastObservedEventNonce(ctx)

	// when the chain starts from genesis state, as there are no events broadcasted, lowest_observed_nonce will be zero.
	// Bridge relayer has to scan the events from the height at which bridge contract is deployed on ethereum.
	// So, if lowest_observed_nonce is zero, we don't need to do anything.
	if lowestObservedNonce == 0 {
		return
	}
	// otherwise, set the last event to the current last observed event nonce and ethereum block height so the validator
	// can begin attesting starting from the next event after the last observed event nonce.
	h.k.setLastEventByValidator(
		ctx,
		validator,
		lowestObservedNonce,
		h.k.GetLastObservedEthereumBlockHeight(ctx).EthereumBlockHeight,
	)
}

func (h Hooks) BeforeDelegationRemoved(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress)        {}
func (h Hooks) AfterValidatorRemoved(ctx sdk.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) {}
func (h Hooks) BeforeValidatorSlashed(ctx sdk.Context, valAddr sdk.ValAddress, fraction sdk.Dec) {}
func (h Hooks) BeforeDelegationSharesModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
}
func (h Hooks) AfterDelegationModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
}
