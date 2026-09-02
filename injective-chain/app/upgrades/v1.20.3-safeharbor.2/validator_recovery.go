//revive:disable-next-line:package-directory-mismatch // Semver upgrade directory names cannot be valid Go package identifiers.
package v1dot20dot3safeharbor2

import (
	"bytes"
	"strconv"
	"time"

	"cosmossdk.io/log"
	evidencetypes "cosmossdk.io/x/evidence/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
)

const (
	// IncidentDoubleSignHeight is the height at which the exempt equivocations occurred.
	IncidentDoubleSignHeight int64 = 181027006
	// IncidentValidatorUnbondHeight is the height at which the incident jails began unbonding.
	IncidentValidatorUnbondHeight int64 = 181027007

	validatorRecoveryEventType = "safeharbor_validator_recovery"
)

var incidentValidatorUnbondTime = time.Date(2026, time.September, 21, 19, 52, 14, 378595183, time.UTC)

// IncidentValidatorUnbondTime returns the incident validator-unbonding queue time.
func IncidentValidatorUnbondTime() time.Time {
	return incidentValidatorUnbondTime
}

// IncidentValidatorRecovery is the audited mainnet state needed to reverse one
// validator's incident-attributed double-sign punishment. Voting power, rounds
// and evidence heights are retained in the manifest for operational review.
type IncidentValidatorRecovery struct {
	Name                    string
	OperatorAddress         string
	ConsensusAddress        string
	VotingPower             int64
	DoubleSignHeight        int64
	DoubleSignRounds        []int32
	EvidenceHeights         []int64
	IncidentUnbondingID     uint64
	PreviousUnbondingHeight int64
	PreviousUnbondingTime   time.Time
	PreviousUnbondingIDs    []uint64
	PreviousJailedUntil     time.Time
}

// The manifest combines the validator and signing-info state at height
// 181027005 with the duplicate-vote evidence committed at 181027007 and
// 181027009. The voting power is the power carried by that evidence.
var incidentValidatorRecoveries = []IncidentValidatorRecovery{
	{
		Name:                    "Binance Staking",
		OperatorAddress:         "injvaloper1lsuqpgm8kgwpq96ewyew26xnfwyn3lh3ncrkrk",
		ConsensusAddress:        "injvalcons1893nm5rlsl3pdcx26q0qhcskughcn2el3w335z",
		VotingPower:             3670720,
		DoubleSignHeight:        IncidentDoubleSignHeight,
		DoubleSignRounds:        []int32{1, 18, 34, 49},
		EvidenceHeights:         []int64{181027007},
		IncidentUnbondingID:     499354,
		PreviousUnbondingHeight: 102055130,
		PreviousUnbondingTime:   time.Date(2025, time.February, 3, 10, 8, 26, 978179650, time.UTC),
		PreviousUnbondingIDs:    []uint64{1154, 1183, 23553, 403262},
		PreviousJailedUntil:     time.Date(2025, time.January, 13, 10, 18, 26, 978179650, time.UTC),
	},
	{
		Name:                    "Everstake",
		OperatorAddress:         "injvaloper134dct56cq5v7uerxcy2cn4m06mqf4dxrlgpp24",
		ConsensusAddress:        "injvalcons175ctmkculqfkwzn0du09w3r56ny38p04w8fen0",
		VotingPower:             3494964,
		DoubleSignHeight:        IncidentDoubleSignHeight,
		DoubleSignRounds:        []int32{6, 25, 40, 60, 74, 89, 108, 125, 142, 157, 173, 192, 209},
		EvidenceHeights:         []int64{181027007, 181027009},
		IncidentUnbondingID:     499353,
		PreviousUnbondingHeight: 125375849,
		PreviousUnbondingTime:   time.Date(2025, time.August, 6, 16, 56, 28, 551511197, time.UTC),
		PreviousUnbondingIDs:    []uint64{446582},
		PreviousJailedUntil:     time.Unix(0, 0).UTC(),
	},
	{
		Name:                    "B-Harvest",
		OperatorAddress:         "injvaloper1zpy3qf7us3m0pxpqkp72gzjv55t70huy33t47x",
		ConsensusAddress:        "injvalcons1ajt4ahdfyd65c6axq3xuvpqmxht6myk0xrqard",
		VotingPower:             676858,
		DoubleSignHeight:        IncidentDoubleSignHeight,
		DoubleSignRounds:        []int32{55},
		EvidenceHeights:         []int64{181027007},
		IncidentUnbondingID:     499352,
		PreviousUnbondingHeight: 125325847,
		PreviousUnbondingTime:   time.Date(2025, time.August, 6, 7, 15, 58, 942647875, time.UTC),
		PreviousUnbondingIDs:    []uint64{446373},
		PreviousJailedUntil:     time.Unix(0, 0).UTC(),
	},
}

// IncidentValidatorRecoveries returns a deep copy of the audited recovery
// manifest so callers cannot mutate the consensus configuration.
func IncidentValidatorRecoveries() []IncidentValidatorRecovery {
	recoveries := make([]IncidentValidatorRecovery, len(incidentValidatorRecoveries))
	for idx, recovery := range incidentValidatorRecoveries {
		recoveries[idx] = recovery
		recoveries[idx].DoubleSignRounds = append([]int32(nil), recovery.DoubleSignRounds...)
		recoveries[idx].EvidenceHeights = append([]int64(nil), recovery.EvidenceHeights...)
		recoveries[idx].PreviousUnbondingIDs = append([]uint64(nil), recovery.PreviousUnbondingIDs...)
	}
	return recoveries
}

type plannedValidatorRecovery struct {
	recovery       IncidentValidatorRecovery
	operator       sdk.ValAddress
	consensus      sdk.ConsAddress
	validator      stakingtypes.Validator
	signingInfo    slashingtypes.ValidatorSigningInfo
	alreadyApplied bool
}

// RecoverIncidentValidators reverses the verified double-sign punishment and
// cancels the incident-attributed validator unbonding. The upgrade runner
// executes this function in a cached context, so any validation or execution
// error discards the entire recovery batch. The normal staking end blocker
// performs the canonical Unbonding-to-Bonded transition in the same block.
func RecoverIncidentValidators(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	return recoverIncidentValidators(ctx, app, logger, incidentValidatorRecoveries)
}

// RecoverIncidentValidatorsFromManifest runs the recovery engine with an
// explicit manifest. The registered upgrade always calls
// RecoverIncidentValidators and therefore always uses the fixed audited
// mainnet manifest; this entry point supports deterministic migration tests.
func RecoverIncidentValidatorsFromManifest(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	logger log.Logger,
	recoveries []IncidentValidatorRecovery,
) error {
	return recoverIncidentValidators(ctx, app, logger, recoveries)
}

func recoverIncidentValidators(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	logger log.Logger,
	recoveries []IncidentValidatorRecovery,
) error {
	if app.ChainID() != upgrades.MainnetChainID || ctx.ChainID() != upgrades.MainnetChainID {
		return errors.Errorf(
			"incident validator recovery is mainnet-only: app_chain_id=%s context_chain_id=%s",
			app.ChainID(),
			ctx.ChainID(),
		)
	}
	if err := validateIncidentValidatorManifest(recoveries); err != nil {
		return err
	}

	plannedValidators, err := planValidatorRecoveries(ctx, app, recoveries)
	if err != nil {
		return err
	}

	if err := executeValidatorRecoveries(ctx, app, plannedValidators); err != nil {
		return err
	}

	recoveredValidators := 0
	for _, planned := range plannedValidators {
		if !planned.alreadyApplied {
			recoveredValidators++
		}
	}
	logger.Info(
		"Recovered incident validators and scheduled rebonding",
		"validators", recoveredValidators,
		"already_recovered_validators", len(plannedValidators)-recoveredValidators,
	)
	return nil
}

// RecoverSimulatedIncidentValidator exercises the recovery mechanics on the
// isolated interchain-test chain. Mainnet always uses the fixed audited
// manifest through RecoverIncidentValidators.
func RecoverSimulatedIncidentValidator(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	if app.ChainID() != InterchainTestChainID || ctx.ChainID() != InterchainTestChainID {
		return errors.Errorf(
			"simulated validator recovery is restricted to %s: app_chain_id=%s context_chain_id=%s",
			InterchainTestChainID,
			app.ChainID(),
			ctx.ChainID(),
		)
	}

	stakingKeeper := app.GetStakingKeeper()
	slashingKeeper := app.GetSlashingKeeper()
	validators, err := stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return errors.Wrap(err, "load validators for simulated recovery")
	}

	plannedValidators := make([]plannedValidatorRecovery, 0, 1)
	for _, validator := range validators {
		consensus, err := validator.GetConsAddr()
		if err != nil {
			return errors.Wrapf(err, "derive consensus address for %s", validator.OperatorAddress)
		}
		signingInfo, err := slashingKeeper.GetValidatorSigningInfo(ctx, consensus)
		if err != nil {
			return errors.Wrapf(err, "load signing info for %s", validator.OperatorAddress)
		}
		if !signingInfo.Tombstoned {
			continue
		}
		if !validator.Jailed || validator.Status != stakingtypes.Unbonding || len(validator.UnbondingIds) == 0 {
			return errors.Errorf(
				"simulated tombstoned validator has unexpected state: operator=%s jailed=%t status=%s unbonding_ids=%v",
				validator.OperatorAddress,
				validator.Jailed,
				validator.Status.String(),
				validator.UnbondingIds,
			)
		}

		incidentUnbondingID := validator.UnbondingIds[len(validator.UnbondingIds)-1]
		unbondingType, err := stakingKeeper.GetUnbondingType(ctx, incidentUnbondingID)
		if err != nil {
			return errors.Wrapf(err, "load simulated validator unbonding type %d", incidentUnbondingID)
		}
		if unbondingType != stakingtypes.UnbondingType_ValidatorUnbonding {
			return errors.Errorf("unexpected simulated validator unbonding type %d", unbondingType)
		}
		indexedValidator, err := stakingKeeper.GetValidatorByUnbondingID(ctx, incidentUnbondingID)
		if err != nil {
			return errors.Wrapf(err, "load simulated validator unbonding index %d", incidentUnbondingID)
		}
		if indexedValidator.OperatorAddress != validator.OperatorAddress {
			return errors.Errorf("simulated validator unbonding index %d points to %s", incidentUnbondingID, indexedValidator.OperatorAddress)
		}
		queueOccurrences, err := validatorQueueOccurrences(
			ctx,
			stakingKeeper,
			validator.UnbondingTime,
			validator.UnbondingHeight,
			validator.OperatorAddress,
		)
		if err != nil {
			return errors.Wrapf(err, "inspect simulated validator queue for %s", validator.OperatorAddress)
		}
		if queueOccurrences != 1 {
			return errors.Errorf("expected one simulated validator queue entry for %s, got %d", validator.OperatorAddress, queueOccurrences)
		}

		consensusAddress, err := stakingKeeper.ConsensusAddressCodec().BytesToString(consensus)
		if err != nil {
			return errors.Wrapf(err, "encode consensus address for %s", validator.OperatorAddress)
		}
		operator, err := stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.OperatorAddress)
		if err != nil {
			return errors.Wrapf(err, "decode operator address for %s", validator.OperatorAddress)
		}
		recovery := IncidentValidatorRecovery{
			Name:                    validator.Description.Moniker,
			OperatorAddress:         validator.OperatorAddress,
			ConsensusAddress:        consensusAddress,
			DoubleSignHeight:        validator.UnbondingHeight - 1,
			IncidentUnbondingID:     incidentUnbondingID,
			PreviousUnbondingHeight: 0,
			PreviousUnbondingTime:   time.Time{},
			PreviousUnbondingIDs:    append([]uint64(nil), validator.UnbondingIds[:len(validator.UnbondingIds)-1]...),
			PreviousJailedUntil:     time.Unix(0, 0).UTC(),
		}
		plannedValidators = append(plannedValidators, plannedValidatorRecovery{
			recovery:    recovery,
			operator:    operator,
			consensus:   consensus,
			validator:   validator,
			signingInfo: signingInfo,
		})
	}
	if len(plannedValidators) != 1 {
		return errors.Errorf("expected exactly one simulated tombstoned validator, got %d", len(plannedValidators))
	}

	if err := executeValidatorRecoveries(ctx, app, plannedValidators); err != nil {
		return err
	}

	logger.Info(
		"Recovered simulated incident validator and scheduled rebonding",
		"validator", plannedValidators[0].recovery.OperatorAddress,
	)
	return nil
}

func validateIncidentValidatorManifest(recoveries []IncidentValidatorRecovery) error {
	if len(recoveries) == 0 {
		return errors.New("incident validator recovery manifest is empty")
	}

	operators := make(map[string]struct{}, len(recoveries))
	consensusAddresses := make(map[string]struct{}, len(recoveries))
	unbondingIDs := make(map[uint64]struct{}, len(recoveries))
	for _, recovery := range recoveries {
		if recovery.Name == "" || recovery.VotingPower <= 0 {
			return errors.Errorf("invalid incident validator metadata for %s", recovery.OperatorAddress)
		}
		if recovery.DoubleSignHeight != IncidentDoubleSignHeight {
			return errors.Errorf("unexpected double-sign height %d for %s", recovery.DoubleSignHeight, recovery.Name)
		}
		if len(recovery.DoubleSignRounds) == 0 || len(recovery.EvidenceHeights) == 0 {
			return errors.Errorf("missing double-sign audit data for %s", recovery.Name)
		}
		if _, duplicate := operators[recovery.OperatorAddress]; duplicate {
			return errors.Errorf("duplicate incident validator operator %s", recovery.OperatorAddress)
		}
		operators[recovery.OperatorAddress] = struct{}{}
		if _, duplicate := consensusAddresses[recovery.ConsensusAddress]; duplicate {
			return errors.Errorf("duplicate incident validator consensus address %s", recovery.ConsensusAddress)
		}
		consensusAddresses[recovery.ConsensusAddress] = struct{}{}
		if _, duplicate := unbondingIDs[recovery.IncidentUnbondingID]; duplicate {
			return errors.Errorf("duplicate incident validator unbonding ID %d", recovery.IncidentUnbondingID)
		}
		unbondingIDs[recovery.IncidentUnbondingID] = struct{}{}
		for _, previousID := range recovery.PreviousUnbondingIDs {
			if previousID == recovery.IncidentUnbondingID {
				return errors.Errorf("incident unbonding ID %d is present in previous IDs for %s", previousID, recovery.Name)
			}
		}
	}
	return nil
}

func planValidatorRecoveries(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	recoveries []IncidentValidatorRecovery,
) ([]plannedValidatorRecovery, error) {
	stakingKeeper := app.GetStakingKeeper()
	slashingKeeper := app.GetSlashingKeeper()
	stakingStore := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	planned := make([]plannedValidatorRecovery, 0, len(recoveries))

	for _, recovery := range recoveries {
		operator, err := stakingKeeper.ValidatorAddressCodec().StringToBytes(recovery.OperatorAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "decode operator address for %s", recovery.Name)
		}
		consensus, err := stakingKeeper.ConsensusAddressCodec().StringToBytes(recovery.ConsensusAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "decode consensus address for %s", recovery.Name)
		}

		validator, err := stakingKeeper.GetValidator(ctx, operator)
		if err != nil {
			return nil, errors.Wrapf(err, "load validator %s", recovery.Name)
		}
		actualConsensus, err := validator.GetConsAddr()
		if err != nil {
			return nil, errors.Wrapf(err, "derive consensus address for %s", recovery.Name)
		}
		if !bytes.Equal(actualConsensus, consensus) {
			return nil, errors.Errorf("consensus address mismatch for %s", recovery.Name)
		}

		signingInfo, err := slashingKeeper.GetValidatorSigningInfo(ctx, consensus)
		if err != nil {
			return nil, errors.Wrapf(err, "load signing info for %s", recovery.Name)
		}

		incidentIndexExists := stakingStore.Has(stakingtypes.GetUnbondingIndexKey(recovery.IncidentUnbondingID))
		incidentTypeExists := stakingStore.Has(stakingtypes.GetUnbondingTypeKey(recovery.IncidentUnbondingID))
		incidentQueueCount, err := validatorQueueOccurrences(
			ctx,
			stakingKeeper,
			incidentValidatorUnbondTime,
			IncidentValidatorUnbondHeight,
			recovery.OperatorAddress,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "inspect validator queue for %s", recovery.Name)
		}

		isIncidentState := validator.Jailed &&
			validator.Status == stakingtypes.Unbonding &&
			validator.UnbondingHeight == IncidentValidatorUnbondHeight &&
			validator.UnbondingTime.Equal(incidentValidatorUnbondTime) &&
			validator.UnbondingOnHoldRefCount == 0 &&
			equalUint64s(validator.UnbondingIds, appendCopy(recovery.PreviousUnbondingIDs, recovery.IncidentUnbondingID)) &&
			signingInfo.Tombstoned &&
			signingInfo.JailedUntil.Equal(evidencetypes.DoubleSignJailEndTime) &&
			incidentIndexExists && incidentTypeExists && incidentQueueCount == 1

		isRecoveredState := !validator.Jailed &&
			(validator.Status == stakingtypes.Unbonding || validator.Status == stakingtypes.Bonded) &&
			validator.UnbondingHeight == recovery.PreviousUnbondingHeight &&
			validator.UnbondingTime.Equal(recovery.PreviousUnbondingTime) &&
			validator.UnbondingOnHoldRefCount == 0 &&
			equalUint64s(validator.UnbondingIds, recovery.PreviousUnbondingIDs) &&
			!signingInfo.Tombstoned &&
			signingInfo.JailedUntil.Equal(recovery.PreviousJailedUntil) &&
			!incidentIndexExists && !incidentTypeExists && incidentQueueCount == 0

		if !isIncidentState && !isRecoveredState {
			return nil, errors.Errorf(
				"validator %s is neither in the expected incident state nor the recovered state: jailed=%t status=%s unbonding_height=%d unbonding_time=%s unbonding_ids=%v tombstoned=%t jailed_until=%s incident_index=%t incident_type=%t incident_queue_occurrences=%d",
				recovery.Name,
				validator.Jailed,
				validator.Status.String(),
				validator.UnbondingHeight,
				validator.UnbondingTime.UTC().Format(time.RFC3339Nano),
				validator.UnbondingIds,
				signingInfo.Tombstoned,
				signingInfo.JailedUntil.UTC().Format(time.RFC3339Nano),
				incidentIndexExists,
				incidentTypeExists,
				incidentQueueCount,
			)
		}

		if isIncidentState {
			unbondingType, err := stakingKeeper.GetUnbondingType(ctx, recovery.IncidentUnbondingID)
			if err != nil {
				return nil, errors.Wrapf(err, "load incident unbonding type for %s", recovery.Name)
			}
			if unbondingType != stakingtypes.UnbondingType_ValidatorUnbonding {
				return nil, errors.Errorf("unexpected incident unbonding type %d for %s", unbondingType, recovery.Name)
			}
			indexedValidator, err := stakingKeeper.GetValidatorByUnbondingID(ctx, recovery.IncidentUnbondingID)
			if err != nil {
				return nil, errors.Wrapf(err, "load incident validator index for %s", recovery.Name)
			}
			if indexedValidator.OperatorAddress != recovery.OperatorAddress {
				return nil, errors.Errorf("incident unbonding ID %d points to %s instead of %s", recovery.IncidentUnbondingID, indexedValidator.OperatorAddress, recovery.OperatorAddress)
			}
		}

		planned = append(planned, plannedValidatorRecovery{
			recovery:       recovery,
			operator:       operator,
			consensus:      consensus,
			validator:      validator,
			signingInfo:    signingInfo,
			alreadyApplied: isRecoveredState,
		})
	}
	return planned, nil
}

func validatorQueueOccurrences(
	ctx sdk.Context,
	stakingKeeper *stakingkeeper.Keeper,
	unbondingTime time.Time,
	unbondingHeight int64,
	operatorAddress string,
) (int, error) {
	addresses, err := stakingKeeper.GetUnbondingValidators(ctx, unbondingTime, unbondingHeight)
	if err != nil {
		return 0, err
	}
	wanted, err := stakingKeeper.ValidatorAddressCodec().StringToBytes(operatorAddress)
	if err != nil {
		return 0, err
	}

	occurrences := 0
	for _, address := range addresses {
		stored, err := stakingKeeper.ValidatorAddressCodec().StringToBytes(address)
		if err != nil {
			return 0, err
		}
		if bytes.Equal(stored, wanted) {
			occurrences++
		}
	}
	return occurrences, nil
}

func executeValidatorRecoveries(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	planned []plannedValidatorRecovery,
) error {
	stakingKeeper := app.GetStakingKeeper()
	slashingKeeper := app.GetSlashingKeeper()
	stakingStore := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))

	for _, recovery := range planned {
		if recovery.alreadyApplied {
			continue
		}
		if err := stakingKeeper.DeleteValidatorQueue(ctx, recovery.validator); err != nil {
			return errors.Wrapf(err, "remove incident validator queue entry for %s", recovery.recovery.Name)
		}
		if err := stakingKeeper.DeleteUnbondingIndex(ctx, recovery.recovery.IncidentUnbondingID); err != nil {
			return errors.Wrapf(err, "delete incident unbonding index for %s", recovery.recovery.Name)
		}
		stakingStore.Delete(stakingtypes.GetUnbondingTypeKey(recovery.recovery.IncidentUnbondingID))

		validator := recovery.validator
		validator.UnbondingHeight = recovery.recovery.PreviousUnbondingHeight
		validator.UnbondingTime = recovery.recovery.PreviousUnbondingTime
		validator.UnbondingIds = append([]uint64(nil), recovery.recovery.PreviousUnbondingIDs...)
		if err := stakingKeeper.SetValidator(ctx, validator); err != nil {
			return errors.Wrapf(err, "restore validator metadata for %s", recovery.recovery.Name)
		}

		signingInfo := recovery.signingInfo
		signingInfo.Tombstoned = false
		signingInfo.JailedUntil = recovery.recovery.PreviousJailedUntil
		if err := slashingKeeper.SetValidatorSigningInfo(ctx, recovery.consensus, signingInfo); err != nil {
			return errors.Wrapf(err, "restore signing info for %s", recovery.recovery.Name)
		}
		if err := slashingKeeper.Unjail(ctx, recovery.operator); err != nil {
			return errors.Wrapf(err, "unjail validator %s", recovery.recovery.Name)
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			validatorRecoveryEventType,
			sdk.NewAttribute("name", recovery.recovery.Name),
			sdk.NewAttribute("operator_address", recovery.recovery.OperatorAddress),
			sdk.NewAttribute("consensus_address", recovery.recovery.ConsensusAddress),
			sdk.NewAttribute("incident_height", strconv.FormatInt(recovery.recovery.DoubleSignHeight, 10)),
			sdk.NewAttribute("incident_unbonding_id", strconv.FormatUint(recovery.recovery.IncidentUnbondingID, 10)),
		))
	}
	return nil
}

func appendCopy(values []uint64, value uint64) []uint64 {
	result := append([]uint64(nil), values...)
	return append(result, value)
}

func equalUint64s(left, right []uint64) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}
