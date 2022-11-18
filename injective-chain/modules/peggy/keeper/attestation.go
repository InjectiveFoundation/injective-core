package keeper

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/metrics"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

func (k *Keeper) Attest(ctx sdk.Context, claim types.EthereumClaim, anyClaim *codectypes.Any) (*types.Attestation, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	valAddr, found := k.GetOrchestratorValidator(ctx, claim.GetClaimer())
	if !found {
		metrics.ReportFuncError(k.svcTags)
		panic("Could not find ValAddr for delegate key, should be checked by now")
	}

	// Check that the nonce of this event is exactly one higher than the last nonce stored by this validator.
	// We check the event nonce in processAttestation as well,
	// but checking it here gives individual eth signers a chance to retry,
	// and prevents validators from submitting two claims with the same nonce
	lastEvent := k.GetLastEventByValidator(ctx, valAddr)
	if claim.GetEventNonce() != lastEvent.EthereumEventNonce+1 {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrNonContiguousEventNonce
	}

	// Tries to get an attestation with the same eventNonce and claim as the claim that was submitted.
	att := k.GetAttestation(ctx, claim.GetEventNonce(), claim.ClaimHash())
	isNewAttestation := false

	// If it does not exist, create a new one.
	if att == nil {
		att = &types.Attestation{
			Observed: false,
			Height:   uint64(ctx.BlockHeight()),
			Claim:    anyClaim,
		}
		isNewAttestation = true
	}

	// Add the validator's vote to this attestation
	att.Votes = append(att.Votes, valAddr.String())

	k.SetAttestation(ctx, claim.GetEventNonce(), claim.ClaimHash(), att)
	k.setLastEventByValidator(ctx, valAddr, claim.GetEventNonce(), claim.GetBlockHeight())
	attestationId := types.GetAttestationKey(claim.GetEventNonce(), claim.ClaimHash())

	if isNewAttestation {
		emitNewClaimEvent(ctx, claim, attestationId)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventAttestationVote{
		EventNonce:    claim.GetEventNonce(),
		AttestationId: attestationId,
		Voter:         valAddr.String(),
	})

	return att, nil
}

func emitNewClaimEvent(ctx sdk.Context, claim types.EthereumClaim, attestationId []byte) {
	switch claim := claim.(type) {
	case *types.MsgDepositClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventDepositClaim{
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			EthereumSender:      claim.GetEthereumSender(),
			CosmosReceiver:      claim.GetCosmosReceiver(),
			TokenContract:       claim.GetTokenContract(),
			Amount:              claim.Amount,
			AttestationId:       attestationId,
			OrchestratorAddress: claim.GetOrchestrator(),
			Data:                claim.Data,
		})
	case *types.MsgWithdrawClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventWithdrawClaim{
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			BatchNonce:          claim.GetBatchNonce(),
			TokenContract:       claim.GetTokenContract(),
			OrchestratorAddress: claim.GetOrchestrator(),
			AttestationId:       attestationId,
		})
	case *types.MsgERC20DeployedClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventERC20DeployedClaim{
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			CosmosDenom:         claim.GetCosmosDenom(),
			TokenContract:       claim.GetTokenContract(),
			Name:                claim.GetName(),
			Symbol:              claim.GetSymbol(),
			Decimals:            claim.GetDecimals(),
			OrchestratorAddress: claim.GetOrchestrator(),
			AttestationId:       attestationId,
		})
	case *types.MsgValsetUpdatedClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventValsetUpdateClaim{
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			ValsetNonce:         claim.GetValsetNonce(),
			ValsetMembers:       claim.GetMembers(),
			RewardAmount:        claim.RewardAmount,
			RewardToken:         claim.GetRewardToken(),
			OrchestratorAddress: claim.GetOrchestrator(),
			AttestationId:       attestationId,
		})
	}
}

// TryAttestation checks if an attestation has enough votes to be applied to the consensus state
// and has not already been marked Observed, then calls processAttestation to actually apply it to the state,
// and then marks it Observed and emits an event.
func (k *Keeper) TryAttestation(ctx sdk.Context, att *types.Attestation) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	claim, err := k.UnpackAttestationClaim(att)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic("could not cast to claim")
	}
	// If the attestation has not yet been Observed, sum up the votes and see if it is ready to apply to the state.
	// This conditional stops the attestation from accidentally being applied twice.
	if !att.Observed {
		// Sum the current powers of all validators who have voted and see if it passes the current threshold
		totalPower := k.StakingKeeper.GetLastTotalPower(ctx)
		requiredPower := types.AttestationVotesPowerThreshold.Mul(totalPower).Quo(sdk.NewInt(100))
		attestationPower := sdk.ZeroInt()
		for _, validator := range att.Votes {
			val, err := sdk.ValAddressFromBech32(validator)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				panic(err)
			}
			validatorPower := k.StakingKeeper.GetLastValidatorPower(ctx, val)
			// Add it to the attestation power's sum
			attestationPower = attestationPower.Add(sdk.NewInt(validatorPower))
			// If the power of all the validators that have voted on the attestation is higher or equal to the threshold,
			// process the attestation, set Observed to true, and break
			if attestationPower.GTE(requiredPower) {
				lastEventNonce := k.GetLastObservedEventNonce(ctx)
				// this check is performed at the next level up so this should never panic
				// outside of programmer error.
				if claim.GetEventNonce() != lastEventNonce+1 {
					metrics.ReportFuncError(k.svcTags)
					panic("attempting to apply events to state out of order")
				}
				k.setLastObservedEventNonce(ctx, claim.GetEventNonce())
				k.SetLastObservedEthereumBlockHeight(ctx, claim.GetBlockHeight())

				att.Observed = true
				k.SetAttestation(ctx, claim.GetEventNonce(), claim.ClaimHash(), att)

				k.processAttestation(ctx, claim)
				k.emitObservedEvent(ctx, att, claim)

				// handle the case where user sends arbitrary data in the MsgDepositClaim
				k.ProcessClaimData(ctx, claim)
				break
			}
		}
	} else {
		// We panic here because this should never happen
		metrics.ReportFuncError(k.svcTags)
		panic("attempting to process observed attestation")
	}
}

// processAttestation actually applies the attestation to the consensus state
func (k *Keeper) processAttestation(ctx sdk.Context, claim types.EthereumClaim) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// then execute in a new Tx so that we can store state on failure
	xCtx, commit := ctx.CacheContext()
	if err := k.AttestationHandler.Handle(xCtx, claim); err != nil { // execute with a transient storage
		// If the attestation fails, something has gone wrong and we can't recover it. Log and move on
		// The attestation will still be marked "Observed", and validators can still be slashed for not
		// having voted for it.
		k.logger.WithError(err).WithFields(log.Fields{
			"claim_type": claim.GetType(),
			"id":         types.GetAttestationKey(claim.GetEventNonce(), claim.ClaimHash()),
			"nonce":      fmt.Sprint(claim.GetEventNonce()),
		}).Warningln("attestation failed")
	} else {
		commit() // persist transient storage
	}
}

// emitObservedEvent emits an event with information about an attestation that has been applied to
// consensus state.
func (k *Keeper) emitObservedEvent(ctx sdk.Context, _ *types.Attestation, claim types.EthereumClaim) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventAttestationObserved{
		AttestationType: claim.GetType(),
		BridgeContract:  k.GetBridgeContractAddress(ctx).Hex(),
		BridgeChainId:   k.GetBridgeChainID(ctx),
		AttestationId:   types.GetAttestationKey(claim.GetEventNonce(), claim.ClaimHash()),
		Nonce:           claim.GetEventNonce(),
	})
}

func (k *Keeper) ProcessClaimData(ctx sdk.Context, claim types.EthereumClaim) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	switch claim := claim.(type) {
	case *types.MsgDepositClaim:
		// Handle arbitrary data in deposit claim
		if claim.Data != "" {

			// Check if the claim data is a valid sdk.Msg. If not, ignore the data
			ethereumSenderInjAccAddr := sdk.AccAddress(common.FromHex(claim.EthereumSender))
			claimDataMsg, err := k.ValidateClaimData(ctx, claim.Data, ethereumSenderInjAccAddr)
			if err != nil {
				log.Infoln("claim data is not a valid sdk.Msg", err)
				return
			}

			// then execute sdk.msg in a new cache cTx so that we can avoid state changes on failure
			xCtx, commit := ctx.CacheContext()
			wrappedCacheCtx := sdk.WrapSDKContext(xCtx)

			// Process the claim data msg
			switch msg := claimDataMsg.(type) {
			case *exchangetypes.MsgDeposit:
				// Enforce that the deposit denom is same as deposit claim amount denom
				// Enforce the deposit amount is not greater than the deposit claim amount
				_, denom := k.ERC20ToDenomLookup(xCtx, common.HexToAddress(claim.TokenContract))
				if msg.Amount.Denom != denom {
					k.logger.WithFields(log.Fields{
						"deposit_denom": msg.Amount.Denom,
						"claim_denom":   denom,
					}).Warningln("deposit denom should be same as deposit claim amount denom")
					return
				}

				claimAmount := sdk.NewCoin(denom, claim.Amount)
				if claimAmount.IsLT(msg.Amount) {
					k.logger.WithFields(log.Fields{
						"deposit_amount": msg.Amount,
						"claim_amount":   claimAmount,
					}).Warningln("deposit amount exceeds deposit claim amount")
					return
				}

				// Execute the message
				_, err := k.exchangeMsgServer.Deposit(wrappedCacheCtx, msg)
				if err == nil {
					commit() // persist transient storage
				}
			}
		}
	}
}

// SetAttestation sets the attestation in the store
func (k *Keeper) SetAttestation(ctx sdk.Context, eventNonce uint64, claimHash []byte, att *types.Attestation) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	aKey := types.GetAttestationKey(eventNonce, claimHash)
	store.Set(aKey, k.cdc.MustMarshal(att))
}

// GetAttestation return an attestation given a nonce
func (k *Keeper) GetAttestation(ctx sdk.Context, eventNonce uint64, claimHash []byte) *types.Attestation {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	aKey := types.GetAttestationKey(eventNonce, claimHash)
	bz := store.Get(aKey)
	if len(bz) == 0 {
		return nil
	}

	var att types.Attestation
	k.cdc.MustUnmarshal(bz, &att)

	return &att
}

// DeleteAttestation deletes an attestation given an event nonce and claim
func (k *Keeper) DeleteAttestation(ctx sdk.Context, att *types.Attestation) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	claim, err := k.UnpackAttestationClaim(att)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic("Bad Attestation in DeleteAttestation")
	}

	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetAttestationKeyWithHash(claim.GetEventNonce(), claim.ClaimHash()))
}

// GetAttestationMapping returns a mapping of eventnonce -> attestations at that nonce
func (k *Keeper) GetAttestationMapping(ctx sdk.Context) (out map[uint64][]*types.Attestation) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	out = make(map[uint64][]*types.Attestation)
	k.IterateAttestations(ctx, func(_ []byte, attestation *types.Attestation) (stop bool) {
		claim, err := k.UnpackAttestationClaim(attestation)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			panic("couldn't UnpackAttestationClaim")
		}

		eventNonce := claim.GetEventNonce()
		out[eventNonce] = append(out[eventNonce], attestation)

		return false
	})

	return
}

// IterateAttestations iterates through all attestations
func (k *Keeper) IterateAttestations(ctx sdk.Context, cb func(k []byte, v *types.Attestation) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	prefix := types.OracleAttestationKey

	iter := store.Iterator(PrefixRange(prefix))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		attestation := types.Attestation{}

		k.cdc.MustUnmarshal(iter.Value(), &attestation)

		// cb returns true to stop early
		if cb(iter.Key(), &attestation) {
			return
		}
	}
}

// GetLastObservedValset retrieves the last observed validator set from the store
// WARNING: This value is not an up to date validator set on Ethereum, it is a validator set
// that AT ONE POINT was the one in the Gravity bridge on Ethereum. If you assume that it's up
// to date you may break the bridge
func (k *Keeper) GetLastObservedValset(ctx sdk.Context) *types.Valset {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.LastObservedValsetKey)

	if len(bytes) == 0 {
		return nil
	}

	valset := types.Valset{}
	k.cdc.MustUnmarshal(bytes, &valset)

	return &valset
}

// SetLastObservedValset updates the last observed validator set in the store
func (k *Keeper) SetLastObservedValset(ctx sdk.Context, valset types.Valset) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LastObservedValsetKey, k.cdc.MustMarshal(&valset))
}

// GetLastObservedEventNonce returns the latest observed event nonce
func (k *Keeper) GetLastObservedEventNonce(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.LastObservedEventNonceKey)

	if len(bytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(bytes)
}

// GetLastObservedEthereumBlockHeight height gets the block height to of the last observed attestation from
// the store
func (k *Keeper) GetLastObservedEthereumBlockHeight(ctx sdk.Context) types.LastObservedEthereumBlockHeight {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.LastObservedEthereumBlockHeightKey)

	if len(bytes) == 0 {
		return types.LastObservedEthereumBlockHeight{
			CosmosBlockHeight:   0,
			EthereumBlockHeight: 0,
		}
	}

	height := types.LastObservedEthereumBlockHeight{}
	k.cdc.MustUnmarshal(bytes, &height)

	return height
}

// SetLastObservedEthereumBlockHeight sets the block height in the store.
func (k *Keeper) SetLastObservedEthereumBlockHeight(ctx sdk.Context, ethereumHeight uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	height := types.LastObservedEthereumBlockHeight{
		EthereumBlockHeight: ethereumHeight,
		CosmosBlockHeight:   uint64(ctx.BlockHeight()),
	}

	store.Set(types.LastObservedEthereumBlockHeightKey, k.cdc.MustMarshal(&height))
}

// setLastObservedEventNonce sets the latest observed event nonce
func (k *Keeper) setLastObservedEventNonce(ctx sdk.Context, nonce uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LastObservedEventNonceKey, types.UInt64Bytes(nonce))
}

func (k *Keeper) setLastEventByValidator(ctx sdk.Context, validator sdk.ValAddress, nonce, blockHeight uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	lastClaimEvent := types.LastClaimEvent{
		EthereumEventNonce:  nonce,
		EthereumEventHeight: blockHeight,
	}

	store.Set(types.GetLastEventByValidatorKey(validator), k.cdc.MustMarshal(&lastClaimEvent))

}

// GetLastEventByValidator returns the latest event for a given validator
func (k *Keeper) GetLastEventByValidator(ctx sdk.Context, validator sdk.ValAddress) (lastEvent types.LastClaimEvent) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.GetLastEventByValidatorKey(validator))

	if len(bytes) == 0 {
		// in the case that we have no existing value this is the first
		// time a validator is submitting a claim. Since we don't want to force
		// them to replay the entire history of all events ever we can't start
		// at zero
		//
		// We could start at the LastObservedEventNonce but if we do that this
		// validator will be slashed, because they are responsible for making a claim
		// on any attestation that has not yet passed the slashing window.
		//
		// Therefore we need to return to them the lowest attestation that is still within
		// the slashing window. Since we delete attestations after the slashing window that's
		// just the lowest observed event in the store. If no claims have been submitted in for
		// params.SignedClaimsWindow we may have no attestations in our nonce. At which point
		// the last observed which is a persistent and never cleaned counter will suffice.
		lowestObservedNonce := k.GetLastObservedEventNonce(ctx)
		lowestObservedHeight := k.GetLastObservedEthereumBlockHeight(ctx)
		peggyParams := k.GetParams(ctx)
		attmap := k.GetAttestationMapping(ctx)

		// when the chain starts from genesis state, as there are no events broadcasted, lowest_observed_nonce will be zero.
		// Bridge relayer has to scan the events from the height at which bridge contract is deployed on ethereum.
		if lowestObservedNonce == 0 {
			lastEvent = types.LastClaimEvent{
				EthereumEventNonce:  lowestObservedNonce,
				EthereumEventHeight: peggyParams.BridgeContractStartHeight,
			}
			return
		}

		// no new claims in params.SignedClaimsWindow, we can return the current value
		// because the validator can't be slashed for an event that has already passed.
		// so they only have to worry about the *next* event to occur
		if len(attmap) == 0 {
			lastEvent = types.LastClaimEvent{
				EthereumEventNonce:  lowestObservedNonce,
				EthereumEventHeight: lowestObservedHeight.EthereumBlockHeight,
			}
			return
		}

		for nonce, atts := range attmap {
			for att := range atts {
				if atts[att].Observed && nonce < lowestObservedNonce {
					claim, err := k.UnpackAttestationClaim(atts[att])
					if err != nil {
						metrics.ReportFuncError(k.svcTags)
						panic("could not cast to claim")
					}
					lastEvent = types.LastClaimEvent{
						EthereumEventNonce:  nonce,
						EthereumEventHeight: claim.GetBlockHeight(),
					}
				}
			}
		}

		return
	} else {
		// Unmarshall last observed event by validator
		k.cdc.MustUnmarshal(bytes, &lastEvent)
		return
	}
}
