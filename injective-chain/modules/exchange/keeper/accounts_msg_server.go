package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type AccountsMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// AccountsMsgServerImpl returns an implementation of the bank MsgServer interface for the provided Keeper for account functions.
func AccountsMsgServerImpl(keeper Keeper) AccountsMsgServer {
	return AccountsMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "acc_msg_h",
		},
	}
}

func (k AccountsMsgServer) BatchUpdateOrders(
	goCtx context.Context,
	msg *types.MsgBatchUpdateOrders,
) (*types.MsgBatchUpdateOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	return k.ExecuteBatchUpdateOrders(
		ctx,
		sender,
		msg.SubaccountId,
		msg.SpotMarketIdsToCancelAll,
		msg.DerivativeMarketIdsToCancelAll,
		msg.BinaryOptionsMarketIdsToCancelAll,
		msg.SpotOrdersToCancel,
		msg.DerivativeOrdersToCancel,
		msg.BinaryOptionsOrdersToCancel,
		msg.SpotOrdersToCreate,
		msg.DerivativeOrdersToCreate,
		msg.BinaryOptionsOrdersToCreate,
	)
}

func (k AccountsMsgServer) Deposit(
	goCtx context.Context,
	msg *types.MsgDeposit,
) (*types.MsgDepositResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.executeDeposit(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgDepositResponse{}, nil
}

func (k AccountsMsgServer) Withdraw(
	goCtx context.Context,
	msg *types.MsgWithdraw,
) (*types.MsgWithdrawResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.ExecuteWithdraw(ctx, msg); err != nil {
		return nil, err
	}
	return &types.MsgWithdrawResponse{}, nil
}

func (k AccountsMsgServer) SubaccountTransfer(
	goCtx context.Context,
	msg *types.MsgSubaccountTransfer,
) (*types.MsgSubaccountTransferResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	srcSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
	dstSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.DestinationSubaccountId)

	denom := msg.Amount.Denom
	amount := msg.Amount.Amount.ToLegacyDec()

	if err := k.Keeper.DecrementDeposit(ctx, srcSubaccountID, denom, amount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if err := k.Keeper.IncrementDepositForNonDefaultSubaccount(ctx, dstSubaccountID, denom, amount); err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	return &types.MsgSubaccountTransferResponse{}, nil
}

func (k AccountsMsgServer) ExternalTransfer(
	goCtx context.Context,
	msg *types.MsgExternalTransfer,
) (*types.MsgExternalTransferResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	srcSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SourceSubaccountId)
	dstSubaccountID := common.HexToHash(msg.DestinationSubaccountId)

	denom := msg.Amount.Denom
	amount := msg.Amount.Amount.ToLegacyDec()

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
		err := k.IncrementDepositForNonDefaultSubaccount(ctx, dstSubaccountID, denom, amount)
		if err != nil {
			return nil, err
		}
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubaccountBalanceTransfer{
		SrcSubaccountId: srcSubaccountID.Hex(),
		DstSubaccountId: dstSubaccountID.Hex(),
		Amount:          msg.Amount,
	})

	return &types.MsgExternalTransferResponse{}, nil
}

func (k AccountsMsgServer) RewardsOptOut(
	goCtx context.Context,
	msg *types.MsgRewardsOptOut,
) (*types.MsgRewardsOptOutResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	account, _ := sdk.AccAddressFromBech32(msg.Sender)
	isAlreadyOptedOut := k.GetIsOptedOutOfRewards(ctx, account)

	if isAlreadyOptedOut {
		return nil, types.ErrAlreadyOptedOutOfRewards
	}

	k.SetIsOptedOutOfRewards(ctx, account, true)

	return &types.MsgRewardsOptOutResponse{}, nil
}

func (k AccountsMsgServer) AuthorizeStakeGrants(
	goCtx context.Context,
	msg *types.MsgAuthorizeStakeGrants,
) (*types.MsgAuthorizeStakeGrantsResponse, error) {
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

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventGrantAuthorizations{
		Granter: granter.String(),
		Grants:  msg.Grants,
	})
	return &types.MsgAuthorizeStakeGrantsResponse{}, nil
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
		k.setActiveGrant(ctx, grantee, types.NewActiveGrant(granter, amount))
	}
}

func (k AccountsMsgServer) ActivateStakeGrant(
	goCtx context.Context,
	msg *types.MsgActivateStakeGrant,
) (*types.MsgActivateStakeGrantResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	grantee := sdk.MustAccAddressFromBech32(msg.Sender)
	granter := sdk.MustAccAddressFromBech32(msg.Granter)

	if !k.ExistsGrantAuthorization(ctx, granter, grantee) {
		return nil, errors.Wrapf(types.ErrInvalidStakeGrant, "grant from %s for %s does not exist", granter.String(), grantee.String())
	}

	granterStake := k.CalculateStakedAmountWithoutCache(ctx, granter, types.MaxGranterDelegations)
	totalGrantAmount := k.GetTotalGrantAmount(ctx, granter)

	if totalGrantAmount.GT(granterStake) {
		return nil, errors.Wrapf(types.ErrInvalidStakeGrant, "grant from %s to %s is invalid since granter staked amount %v is smaller than granter total stake delegated amount %v", granter.String(), grantee.String(), granterStake, totalGrantAmount)
	}

	grantAuthorizationAmount := k.GetGrantAuthorization(ctx, granter, grantee)
	k.setActiveGrant(ctx, grantee, types.NewActiveGrant(granter, grantAuthorizationAmount))

	return &types.MsgActivateStakeGrantResponse{}, nil
}

func (k AccountsMsgServer) BatchExchangeModification(
	goCtx context.Context,
	msg *types.MsgBatchExchangeModification,
) (*types.MsgBatchExchangeModificationResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	isGovernanceAllowed := msg.Sender == k.authority
	if !isGovernanceAllowed {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleBatchExchangeModificationProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &types.MsgBatchExchangeModificationResponse{}, nil
}
