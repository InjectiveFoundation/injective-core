package app

import (
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/codec/types"
)

type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState() GenesisState {
	encCfg := types.MakeEncodingConfig()
	return ModuleBasics.DefaultGenesis(encCfg.Codec)
}

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *InjectiveApp) ExportAppStateAndValidators(
	forZeroHeight bool,
	jailAllowedAddrs []string,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// as if they could withdraw from the start of the next block
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// We export at last height + 1, because that's the height at which
	// Tendermint will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0
		if err := app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	genState, err := app.mm.ExportGenesisForModules(ctx, app.codec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          height,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, err
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favor of export at a block height
func (app *InjectiveApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) error {
	applyAllowedAddrs := false

	// check if there is a allowed address list
	if len(jailAllowedAddrs) > 0 {
		applyAllowedAddrs = true
	}

	allowedAddrsMap := make(map[string]bool)

	for _, addr := range jailAllowedAddrs {
		_, err := sdk.ValAddressFromBech32(addr)
		if err != nil {
			return err
		}
		allowedAddrsMap[addr] = true
	}

	/* Just to be safe, assert the invariants on current state. */
	app.CrisisKeeper.AssertInvariants(ctx)

	/* Handle fee distribution state. */

	// withdraw all validator commission
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		validator, _ := sdk.ValAddressFromBech32(val.GetOperator())
		_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, validator)
		return false
	})

	if err != nil {
		return err
	}

	// withdraw all delegator rewards
	dels, _ := app.StakingKeeper.GetAllDelegations(ctx)
	for _, delegation := range dels {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			return err
		}

		delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
		if err != nil {
			return err
		}
		_, _ = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr)
	}

	// clear validator slash events
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)

	// clear validator historical rewards
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// reinitialize all validators
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		// donate any unwithdrawn outstanding reward fraction tokens to the community pool
		valAddr, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())

		if err == nil {
			return false
		}

		scraps, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valAddr)

		if err != nil {
			return false
		}

		feePool, err := app.DistrKeeper.FeePool.Get(ctx)

		if err != nil {
			return false
		}

		feePool.CommunityPool = feePool.CommunityPool.Add(scraps...)
		err = app.DistrKeeper.FeePool.Set(ctx, feePool)

		if err != nil {
			return false
		}

		_ = app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valAddr) //nolint:errcheck // err is always nil
		return false
	})

	if err != nil {
		return err
	}

	// reinitialize all delegations
	for _, del := range dels {
		valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
		if err != nil {
			return err
		}
		delAddr, err := sdk.AccAddressFromBech32(del.DelegatorAddress)
		if err != nil {
			return err
		}
		_ = app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr) //nolint:errcheck // err is always nil
		_ = app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr) //nolint:errcheck // err is always nil
	}

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// iterate through redelegations, reset creation height
	err = app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		_ = app.StakingKeeper.SetRedelegation(ctx, red)
		return false
	})
	if err != nil {
		return err
	}

	// iterate through unbonding delegations, reset creation height
	err = app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		_ = app.StakingKeeper.SetUnbondingDelegation(ctx, ubd)
		return false
	})

	if err != nil {
		return err
	}

	// Iterate through validators by power descending, reset bond heights, and
	// update bond intra-tx counters.
	store := ctx.KVStore(app.keys[stakingtypes.StoreKey])
	iter := storetypes.KVStoreReversePrefixIterator(store, stakingtypes.ValidatorsKey)
	counter := int16(0)

	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(iter.Key()[1:])
		validator, err := app.StakingKeeper.GetValidator(ctx, addr)
		if err != nil {
			return fmt.Errorf("expected validator %s not found", addr)
		}

		validator.UnbondingHeight = 0
		if applyAllowedAddrs && !allowedAddrsMap[addr.String()] {
			validator.Jailed = true
		}

		_ = app.StakingKeeper.SetValidator(ctx, validator)
		counter++
	}

	iter.Close()

	if _, err := app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx); err != nil {
		return err
	}

	/* Handle slashing state. */

	// reset start height on signing infos
	err = app.SlashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			_ = app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info)
			return false
		},
	)
	return err
}
