package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type AccountsMsgServer struct {
	*Keeper
	svcTags metrics.Tags
}

// AccountsMsgServerImpl returns an implementation of the bank MsgServer interface for the provided Keeper for account functions.
func AccountsMsgServerImpl(keeper *Keeper) AccountsMsgServer {
	return AccountsMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "acc_msg_h",
		},
	}
}

func (k AccountsMsgServer) Deposit(
	c context.Context,
	msg *v2.MsgDeposit,
) (*v2.MsgDepositResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgDeposit")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("Deposit",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	if err := k.executeDeposit(ctx, msg); err != nil {
		return nil, err
	}

	return &v2.MsgDepositResponse{}, nil
}

func (k AccountsMsgServer) Withdraw(
	c context.Context,
	msg *v2.MsgWithdraw,
) (*v2.MsgWithdrawResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgWithdraw")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("Withdraw",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	if err := k.ExecuteWithdraw(ctx, msg); err != nil {
		return nil, err
	}

	return &v2.MsgWithdrawResponse{}, nil
}

func (k AccountsMsgServer) SubaccountTransfer(
	goCtx context.Context,
	msg *v2.MsgSubaccountTransfer,
) (*v2.MsgSubaccountTransferResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	var (
		denom           = msg.Amount.Denom
		amount          = msg.Amount.Amount.ToLegacyDec()
		sender          = sdk.MustAccAddressFromBech32(msg.Sender)
		srcSubaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
		dstSubaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.DestinationSubaccountId)
	)

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgSubaccountTransfer")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("SubaccountTransfer",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}
	if err := k.Keeper.DecrementDeposit(ctx, srcSubaccountID, denom, amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if err := k.Keeper.IncrementDepositForNonDefaultSubaccount(ctx, dstSubaccountID, denom, amount); err != nil {
		return nil, err
	}

	k.EmitEvent(ctx, &v2.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	return &v2.MsgSubaccountTransferResponse{}, nil
}

func (k AccountsMsgServer) ExternalTransfer(
	goCtx context.Context,
	msg *v2.MsgExternalTransfer,
) (*v2.MsgExternalTransferResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	var (
		denom           = msg.Amount.Denom
		amount          = msg.Amount.Amount.ToLegacyDec()
		sender          = sdk.MustAccAddressFromBech32(msg.Sender)
		srcSubaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
		dstSubaccountID = common.HexToHash(msg.DestinationSubaccountId)
	)

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		gasConsumedBefore := ctx.GasMeter().GasConsumed()
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgExternalTransfer")
		totalGas := ctx.GasMeter().GasConsumed()

		// todo: remove after QA
		defer func() {
			k.Logger(ctx).Info("ExternalTransfer",
				"gas_ante", gasConsumedBefore,
				"gas_msg", totalGas-gasConsumedBefore,
				"gas_total", totalGas,
				"sender", msg.Sender,
			)
		}()

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	if err := k.Keeper.DecrementDeposit(ctx, srcSubaccountID, denom, amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	recipientAddr := types.SubaccountIDToSdkAddress(dstSubaccountID)
	// create new account for recipient if it doesn't exist already
	if !k.AccountKeeper.HasAccount(ctx, recipientAddr) {
		defer telemetry.IncrCounter(1, "new", "account")
		k.AccountKeeper.SetAccount(ctx, k.AccountKeeper.NewAccountWithAddress(ctx, recipientAddr))
	}

	if types.IsDefaultSubaccountID(dstSubaccountID) {
		k.IncrementDepositOrSendToBank(ctx, dstSubaccountID, denom, amount)
	} else {
		if err := k.IncrementDepositForNonDefaultSubaccount(ctx, dstSubaccountID, denom, amount); err != nil {
			return nil, err
		}
	}

	k.EmitEvent(ctx, &v2.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	return &v2.MsgExternalTransferResponse{}, nil
}

func (k AccountsMsgServer) RewardsOptOut(
	goCtx context.Context,
	msg *v2.MsgRewardsOptOut,
) (*v2.MsgRewardsOptOutResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	account, _ := sdk.AccAddressFromBech32(msg.Sender)
	if isAlreadyOptedOut := k.GetIsOptedOutOfRewards(ctx, account); isAlreadyOptedOut {
		return nil, types.ErrAlreadyOptedOutOfRewards
	}

	k.SetIsOptedOutOfRewards(ctx, account, true)

	return &v2.MsgRewardsOptOutResponse{}, nil
}

func (k AccountsMsgServer) AuthorizeStakeGrants(
	goCtx context.Context,
	msg *v2.MsgAuthorizeStakeGrants,
) (*v2.MsgAuthorizeStakeGrantsResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	granter := sdk.MustAccAddressFromBech32(msg.Sender)
	granterStake := k.CalculateStakedAmountWithoutCache(ctx, granter, types.MaxGranterDelegations)

	// ensure that the granter has enough stake to cover the grants
	if err := k.ensureValidGrantAuthorization(ctx, granter, msg.Grants, granterStake); err != nil {
		return nil, err
	}

	// update the last delegation check time
	k.setLastValidGrantDelegationCheckTime(ctx, msg.Sender, ctx.BlockTime().Unix())

	// process the grants
	for idx := range msg.Grants {
		grant := msg.Grants[idx]
		grantee := sdk.MustAccAddressFromBech32(grant.Grantee)
		k.authorizeStakeGrant(ctx, granter, grantee, grant.Amount)
	}

	k.EmitEvent(ctx, &v2.EventGrantAuthorizations{
		Granter: granter.String(),
		Grants:  msg.Grants,
	})

	return &v2.MsgAuthorizeStakeGrantsResponse{}, nil
}

func (k AccountsMsgServer) authorizeStakeGrant(
	ctx sdk.Context,
	granter sdk.AccAddress,
	grantee sdk.AccAddress,
	amount math.Int,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	existingGrantAmount := k.GetGrantAuthorization(ctx, granter, grantee)
	existingTotalGrantAmount := k.GetTotalGrantAmount(ctx, granter)

	// update total grant amount accordingly
	totalGrantAmount := existingTotalGrantAmount.Sub(existingGrantAmount).Add(amount)

	k.setTotalGrantAmount(ctx, granter, totalGrantAmount)
	k.setGrantAuthorization(ctx, granter, grantee, amount)

	activeGrant := k.GetActiveGrant(ctx, grantee)

	// TODO: consider not activating the grant authorization if no active grant for the grantee exists, as the grantee
	// may not necessarily desire a grant
	hasActiveGrant := activeGrant != nil

	// update the grantee's active stake grant if the granter matches
	hasActiveGrantFromGranter := hasActiveGrant && activeGrant.Granter == granter.String()

	if hasActiveGrantFromGranter || !hasActiveGrant {
		k.setActiveGrant(ctx, grantee, v2.NewActiveGrant(granter, amount))
	}
}

func (k AccountsMsgServer) ActivateStakeGrant(
	goCtx context.Context,
	msg *v2.MsgActivateStakeGrant,
) (*v2.MsgActivateStakeGrantResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	grantee := sdk.MustAccAddressFromBech32(msg.Sender)
	granter := sdk.MustAccAddressFromBech32(msg.Granter)

	if !k.ExistsGrantAuthorization(ctx, granter, grantee) {
		return nil, sdkerrors.Wrapf(types.ErrInvalidStakeGrant, "grant from %s for %s does not exist", granter.String(), grantee.String())
	}

	granterStake := k.CalculateStakedAmountWithoutCache(ctx, granter, types.MaxGranterDelegations)
	totalGrantAmount := k.GetTotalGrantAmount(ctx, granter)

	if totalGrantAmount.GT(granterStake) {
		return nil, sdkerrors.Wrapf(
			types.ErrInvalidStakeGrant,
			"grant from %s to %s is invalid since granter staked amount %v is smaller than granter total stake delegated amount %v",
			granter.String(),
			grantee.String(),
			granterStake,
			totalGrantAmount,
		)
	}

	grantAuthorizationAmount := k.GetGrantAuthorization(ctx, granter, grantee)
	k.setActiveGrant(ctx, grantee, v2.NewActiveGrant(granter, grantAuthorizationAmount))

	return &v2.MsgActivateStakeGrantResponse{}, nil
}
