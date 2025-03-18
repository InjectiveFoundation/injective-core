package keeper

import (
	"context"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
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

func (k AccountsMsgServer) BatchUpdateOrders(
	goCtx context.Context,
	msg *types.MsgBatchUpdateOrders,
) (*types.MsgBatchUpdateOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		return k.FixedGasBatchUpdateOrders(ctx, msg)
	}

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

	return &types.MsgDepositResponse{}, nil
}

func (k AccountsMsgServer) Withdraw(
	goCtx context.Context,
	msg *types.MsgWithdraw,
) (*types.MsgWithdrawResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
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
	return &types.MsgWithdrawResponse{}, nil
}

func (k AccountsMsgServer) SubaccountTransfer(
	goCtx context.Context,
	msg *types.MsgSubaccountTransfer,
) (*types.MsgSubaccountTransferResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

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
		return nil, errorsmod.Wrapf(types.ErrInvalidStakeGrant, "grant from %s for %s does not exist", granter.String(), grantee.String())
	}

	granterStake := k.CalculateStakedAmountWithoutCache(ctx, granter, types.MaxGranterDelegations)
	totalGrantAmount := k.GetTotalGrantAmount(ctx, granter)

	if totalGrantAmount.GT(granterStake) {
		//nolint:revive //this is fine
		return nil, errorsmod.Wrapf(types.ErrInvalidStakeGrant, "grant from %s to %s is invalid since granter staked amount %v is smaller than granter total stake delegated amount %v", granter.String(), grantee.String(), granterStake, totalGrantAmount)
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

//nolint:revive // this is fine
func (k *Keeper) FixedGasBatchUpdateOrders(
	c context.Context,
	msg *types.MsgBatchUpdateOrders,
) (*types.MsgBatchUpdateOrdersResponse, error) {
	//	no clever method shadowing here

	cc, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(cc)
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	var (
		subaccountId         = msg.SubaccountId
		spotMarkets          = make(map[common.Hash]*types.SpotMarket)
		derivativeMarkets    = make(map[common.Hash]*types.DerivativeMarket)
		binaryOptionsMarkets = make(map[common.Hash]*types.BinaryOptionsMarket)

		spotCancelSuccesses          = make([]bool, len(msg.SpotOrdersToCancel))
		derivativeCancelSuccesses    = make([]bool, len(msg.DerivativeOrdersToCancel))
		binaryOptionsCancelSuccesses = make([]bool, len(msg.BinaryOptionsOrdersToCancel))
		spotOrderHashes              = make([]string, len(msg.SpotOrdersToCreate))
		derivativeOrderHashes        = make([]string, len(msg.DerivativeOrdersToCreate))
		binaryOptionsOrderHashes     = make([]string, len(msg.BinaryOptionsOrdersToCreate))

		createdSpotOrdersCids          = make([]string, 0)
		failedSpotOrdersCids           = make([]string, 0)
		createdDerivativeOrdersCids    = make([]string, 0)
		failedDerivativeOrdersCids     = make([]string, 0)
		createdBinaryOptionsOrdersCids = make([]string, 0)
		failedBinaryOptionsOrdersCids  = make([]string, 0)
	)

	// reference the gas meter early to consume gas later on in loop iterations
	gasMeter := ctx.GasMeter()
	gasConsumedBefore := gasMeter.GasConsumed()

	defer func() {
		totalGas := gasMeter.GasConsumed()
		k.Logger(ctx).Info("MsgBatchUpdateOrders",
			"gas_ante", gasConsumedBefore,
			"gas_msg", totalGas-gasConsumedBefore,
			"gas_total", totalGas,
			"sender", msg.Sender,
		)
	}()

	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	/**	1. Cancel all **/
	// NOTE: provided subaccountID indicates cancelling all orders in a market for given market IDs
	if isCancelAll := subaccountId != ""; isCancelAll {
		//  Derive the subaccountID.
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, subaccountId)

		/**	1. a) Cancel all spot limit orders in markets **/
		for _, spotMarketIdToCancelAll := range msg.SpotMarketIdsToCancelAll {
			marketID := common.HexToHash(spotMarketIdToCancelAll)
			market := k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				continue
			}
			spotMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug("failed to cancel all spot limit orders", "marketID", marketID.Hex())
				continue
			}

			// k.CancelAllSpotLimitOrders(ctx, market, subaccountID, marketID)
			// get all orders to cancel
			var (
				restingBuyOrders = k.GetAllSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				restingSellOrders = k.GetAllSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					subaccountID,
				)
				transientBuyOrders = k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				transientSellOrders = k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					subaccountID,
				)
			)

			// consume gas
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(restingBuyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(restingSellOrders)), "")
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(transientBuyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas*uint64(len(transientSellOrders)), "")

			// cancel orders
			for idx := range restingBuyOrders {
				k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, true, restingBuyOrders[idx])
			}

			for idx := range restingSellOrders {
				k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, false, restingSellOrders[idx])
			}

			for idx := range transientBuyOrders {
				k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, transientBuyOrders[idx])
			}

			for idx := range transientSellOrders {
				k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, transientSellOrders[idx])
			}
		}

		/**	1. b) Cancel all derivative limit orders in markets **/
		for _, derivativeMarketIdToCancelAll := range msg.DerivativeMarketIdsToCancelAll {
			marketID := common.HexToHash(derivativeMarketIdToCancelAll)
			market := k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel all derivative limit orders for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			derivativeMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug(
					"failed to cancel all derivative limit orders for market whose status doesnt support cancellations",
					"marketID",
					marketID.Hex(),
				)
				continue
			}

			var (
				restingBuyOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				restingSellOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false, subaccountID,
				)
				buyOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					true,
				)
				sellOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					false,
				)
				higherMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					true,
					market.GetMarketType(),
					subaccountID,
				)
				lowerMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					true,
					market.GetMarketType(), subaccountID)
				higherLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					false,
					market.GetMarketType(), subaccountID)
				lowerLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					false,
					market.GetMarketType(),
					subaccountID,
				)
			)

			// consume gas
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(restingBuyOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(restingSellOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(buyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(sellOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(higherMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(lowerMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(higherLimitOrders)), "")
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas*uint64(len(lowerLimitOrders)), "")

			for _, hash := range restingBuyOrderHashes {
				isBuy := true

				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range restingSellOrderHashes {
				isBuy := false
				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, buyOrder := range buyOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
					orderHash := common.BytesToHash(buyOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for buyOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					_ = ctx.EventManager().EmitTypedEvent(types.NewEventOrderCancelFail(
						marketID,
						subaccountID,
						orderHash.Hex(),
						buyOrder.Cid(),
						err,
					))
				}
			}

			for _, sellOrder := range sellOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
					orderHash := common.BytesToHash(sellOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for sellOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					_ = ctx.EventManager().EmitTypedEvent(types.NewEventOrderCancelFail(
						marketID,
						subaccountID,
						orderHash.Hex(),
						sellOrder.Cid(),
						err,
					))
				}
			}

			for _, hash := range higherMarketOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerMarketOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range higherLimitOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerLimitOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}
		}

		/**	1. c) Cancel all bo limit orders in markets **/
		for _, binaryOptionsMarketIdToCancelAll := range msg.BinaryOptionsMarketIdsToCancelAll {
			marketID := common.HexToHash(binaryOptionsMarketIdToCancelAll)
			market := k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel all binary options limit orders for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			binaryOptionsMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug(
					"failed to cancel all binary options limit orders for market whose status doesnt support cancellations",
					"marketID",
					marketID.Hex(),
				)
				continue
			}

			var (
				restingBuyOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					subaccountID,
				)
				restingSellOrderHashes = k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					subaccountID,
				)
				buyOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					true,
				)
				sellOrders = k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
					ctx,
					marketID,
					&subaccountID,
					false,
				)
				higherMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					true,
					market.GetMarketType(),
					subaccountID,
				)
				lowerMarketOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					true,
					market.GetMarketType(),
					subaccountID,
				)
				higherLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					true,
					false,
					market.GetMarketType(),
					subaccountID,
				)
				lowerLimitOrders = k.GetAllConditionalOrderHashesBySubaccountAndMarket(
					ctx,
					marketID,
					false,
					false,
					market.GetMarketType(),
					subaccountID,
				)
			)

			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(restingBuyOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(restingSellOrderHashes)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(buyOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(sellOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(higherMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(lowerMarketOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(higherLimitOrders)), "")
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas*uint64(len(lowerLimitOrders)), "")

			for _, hash := range restingBuyOrderHashes {
				isBuy := true
				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range restingSellOrderHashes {
				isBuy := false
				_ = k.CancelRestingDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isBuy,
					hash,
					true,
					true,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, buyOrder := range buyOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
					orderHash := common.BytesToHash(buyOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for buyOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					_ = ctx.EventManager().EmitTypedEvent(
						types.NewEventOrderCancelFail(
							marketID,
							subaccountID,
							orderHash.Hex(),
							buyOrder.Cid(),
							err,
						))
				}
			}

			for _, sellOrder := range sellOrders {
				if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
					orderHash := common.BytesToHash(sellOrder.OrderHash)
					//nolint:revive // this is fine
					k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for sellOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", orderHash.Hex(), err)
					_ = ctx.EventManager().EmitTypedEvent(types.NewEventOrderCancelFail(
						marketID,
						subaccountID,
						orderHash.Hex(),
						sellOrder.Cid(),
						err,
					))
				}
			}

			for _, hash := range higherMarketOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerMarketOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeMarketOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range higherLimitOrders {
				isTriggerPriceHigher := true
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}

			for _, hash := range lowerLimitOrders {
				isTriggerPriceHigher := false
				_ = k.CancelConditionalDerivativeLimitOrder(
					ctx,
					market,
					subaccountID,
					&isTriggerPriceHigher,
					hash,
				) //nolint:errcheck // cannot possibly fail
			}
		}
	}

	/**	2. Cancel all spot limit orders **/
	for idx, spotOrderToCancel := range msg.SpotOrdersToCancel {
		marketID := common.HexToHash(spotOrderToCancel.MarketId)

		var market *types.SpotMarket
		if m, ok := spotMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel spot limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			spotMarkets[marketID] = market
		}

		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, spotOrderToCancel.SubaccountId)

		err := k.cancelSpotLimitOrder(ctx, subaccountID, spotOrderToCancel.GetIdentifier(), market, marketID)

		if err == nil {
			gasMeter.ConsumeGas(MsgCancelSpotOrderGas, "")
			spotCancelSuccesses[idx] = true
		} else {
			ev := types.NewEventOrderCancelFail(
				marketID,
				subaccountID,
				spotOrderToCancel.GetOrderHash(),
				spotOrderToCancel.GetCid(),
				err,
			)
			_ = ctx.EventManager().EmitTypedEvent(ev)
		}
	}

	/**	3. Cancel all derivative limit orders **/
	for idx, derivativeOrderToCancel := range msg.DerivativeOrdersToCancel {
		marketID := common.HexToHash(derivativeOrderToCancel.MarketId)

		var market *types.DerivativeMarket
		if m, ok := derivativeMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel derivative limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			derivativeMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, derivativeOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(
			ctx,
			subaccountID,
			derivativeOrderToCancel.GetIdentifier(),
			market,
			marketID,
			derivativeOrderToCancel.OrderMask,
		)

		if err == nil {
			derivativeCancelSuccesses[idx] = true
			gasMeter.ConsumeGas(MsgCancelDerivativeOrderGas, "")
		} else {
			ev := types.NewEventOrderCancelFail(
				marketID,
				subaccountID,
				derivativeOrderToCancel.GetOrderHash(),
				derivativeOrderToCancel.GetCid(),
				err,
			)
			_ = ctx.EventManager().EmitTypedEvent(ev)
		}
	}

	/**	4. Cancel all bo limit orders **/
	for idx, binaryOptionsOrderToCancel := range msg.BinaryOptionsOrdersToCancel {
		marketID := common.HexToHash(binaryOptionsOrderToCancel.MarketId)

		var market *types.BinaryOptionsMarket
		if m, ok := binaryOptionsMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to cancel binary options limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			binaryOptionsMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, binaryOptionsOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(
			ctx,
			subaccountID,
			binaryOptionsOrderToCancel.GetIdentifier(),
			market,
			marketID,
			binaryOptionsOrderToCancel.OrderMask,
		)

		if err == nil {
			gasMeter.ConsumeGas(MsgCancelBinaryOptionsOrderGas, "")
			binaryOptionsCancelSuccesses[idx] = true
		} else {
			ev := types.NewEventOrderCancelFail(
				marketID,
				subaccountID,
				binaryOptionsOrderToCancel.GetOrderHash(),
				binaryOptionsOrderToCancel.GetCid(),
				err,
			)
			_ = ctx.EventManager().EmitTypedEvent(ev)
		}
	}

	orderFailEvent := types.EventOrderFail{
		Account: sender.Bytes(),
		Hashes:  make([][]byte, 0),
		Flags:   make([]uint32, 0),
		Cids:    make([]string, 0),
	}

	/**	5. Create spot limit orders **/
	for idx, spotOrder := range msg.SpotOrdersToCreate {
		marketID := common.HexToHash(spotOrder.MarketId)
		var market *types.SpotMarket
		if m, ok := spotMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to create spot limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			spotMarkets[marketID] = market
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug(
				"failed to create spot limit order for non-active market",
				"marketID",
				marketID.Hex(),
			)
			continue
		}

		var gasToConsume uint64
		if spotOrder.OrderType == types.OrderType_BUY_PO || spotOrder.OrderType == types.OrderType_SELL_PO {
			gasToConsume = MsgCreateSpotLimitPostOnlyOrderGas
		} else {
			gasToConsume = MsgCreateSpotLimitOrderGas
		}

		if orderHash, err := k.createSpotLimitOrder(ctx, sender, spotOrder, market); err != nil {
			sdkerror := &errorsmod.Error{}
			if errors.As(err, &sdkerror) {
				spotOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, spotOrder.Cid(), sdkerror.ABCICode())
				failedSpotOrdersCids = append(failedSpotOrdersCids, spotOrder.Cid())
			}
		} else {
			gasMeter.ConsumeGas(gasToConsume, "")
			spotOrderHashes[idx] = orderHash.Hex()
			createdSpotOrdersCids = append(createdSpotOrdersCids, spotOrder.Cid())
		}
	}

	markPrices := make(map[common.Hash]math.LegacyDec)

	/**	6. Create derivative limit orders **/
	for idx, derivativeOrder := range msg.DerivativeOrdersToCreate {
		marketID := derivativeOrder.MarketID()

		var market *types.DerivativeMarket
		var markPrice math.LegacyDec
		if m, ok := derivativeMarkets[marketID]; ok {
			market = m
		} else {
			market, markPrice = k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to create derivative limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			derivativeMarkets[marketID] = market
			markPrices[marketID] = markPrice
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug(
				"failed to create derivative limit orders for non-active market",
				"marketID",
				marketID.Hex(),
			)
			continue
		}

		if _, ok := markPrices[marketID]; !ok {
			price, err := k.GetDerivativeMarketPrice(
				ctx,
				market.OracleBase,
				market.OracleQuote,
				market.OracleScaleFactor,
				market.OracleType,
			)
			if err != nil {
				k.Logger(ctx).Debug(
					"failed to create derivative limit order for market with no mark price",
					"marketID",
					marketID.Hex(),
				)
				metrics.ReportFuncError(k.svcTags)
				continue
			}
			markPrices[marketID] = *price
		}
		markPrice = markPrices[marketID]

		var gasToConsume uint64
		if derivativeOrder.OrderType == types.OrderType_BUY_PO || derivativeOrder.OrderType == types.OrderType_SELL_PO {
			gasToConsume = MsgCreateDerivativeLimitPostOnlyOrderGas
		} else {
			gasToConsume = MsgCreateDerivativeLimitOrderGas
		}

		if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, derivativeOrder, market, markPrice); err != nil {
			sdkerror := &errorsmod.Error{}
			if errors.As(err, &sdkerror) {
				derivativeOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, derivativeOrder.Cid(), sdkerror.ABCICode())
				failedDerivativeOrdersCids = append(failedDerivativeOrdersCids, derivativeOrder.Cid())
			}
		} else {
			gasMeter.ConsumeGas(gasToConsume, "")
			derivativeOrderHashes[idx] = orderHash.Hex()
			createdDerivativeOrdersCids = append(createdDerivativeOrdersCids, derivativeOrder.Cid())
		}
	}

	/**	7. Create bo limit orders **/
	for idx, order := range msg.BinaryOptionsOrdersToCreate {
		marketID := order.MarketID()

		var market *types.BinaryOptionsMarket
		if m, ok := binaryOptionsMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetBinaryOptionsMarket(ctx, marketID, true)
			if market == nil {
				k.Logger(ctx).Debug(
					"failed to create binary options limit order for non-existent market",
					"marketID",
					marketID.Hex(),
				)
				continue
			}
			binaryOptionsMarkets[marketID] = market
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug(
				"failed to create binary options limit orders for non-active market",
				"marketID",
				marketID.Hex(),
			)
			continue
		}

		var gasToConsume uint64
		switch order.OrderType {
		case types.OrderType_BUY_PO, types.OrderType_SELL_PO:
			gasToConsume = MsgCreateBinaryOptionsLimitPostOnlyOrderGas
		default:
			gasToConsume = MsgCreateBinaryOptionsLimitOrderGas
		}

		if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, order, market, math.LegacyDec{}); err != nil {
			sdkerror := &errorsmod.Error{}
			if errors.As(err, &sdkerror) {
				binaryOptionsOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, order.Cid(), sdkerror.ABCICode())
				failedBinaryOptionsOrdersCids = append(failedBinaryOptionsOrdersCids, order.Cid())
			}
		} else {
			gasMeter.ConsumeGas(gasToConsume, "")
			binaryOptionsOrderHashes[idx] = orderHash.Hex()
			createdBinaryOptionsOrdersCids = append(createdBinaryOptionsOrdersCids, order.Cid())
		}
	}

	if !orderFailEvent.IsEmpty() {
		// nolint:errcheck // ignored on purpose
		ctx.EventManager().EmitTypedEvent(&orderFailEvent)
	}

	return &types.MsgBatchUpdateOrdersResponse{
		SpotCancelSuccess:              spotCancelSuccesses,
		DerivativeCancelSuccess:        derivativeCancelSuccesses,
		SpotOrderHashes:                spotOrderHashes,
		DerivativeOrderHashes:          derivativeOrderHashes,
		BinaryOptionsCancelSuccess:     binaryOptionsCancelSuccesses,
		BinaryOptionsOrderHashes:       binaryOptionsOrderHashes,
		CreatedSpotOrdersCids:          createdSpotOrdersCids,
		FailedSpotOrdersCids:           failedSpotOrdersCids,
		CreatedDerivativeOrdersCids:    createdDerivativeOrdersCids,
		FailedDerivativeOrdersCids:     failedDerivativeOrdersCids,
		CreatedBinaryOptionsOrdersCids: createdBinaryOptionsOrdersCids,
		FailedBinaryOptionsOrdersCids:  failedBinaryOptionsOrdersCids,
	}, nil
}
