package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type BinaryOptionsMsgServer struct {
	*Keeper
}

const mainnetChainID = "injective-1"

func isBinaryOptionsDisabled(ctx sdk.Context) bool {
	return ctx.ChainID() == mainnetChainID
}

func binaryOptionsDisabledError() error {
	return errors.Wrap(types.ErrFeatureDisabled, "binary options are disabled")
}

// NewBinaryOptionsMsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper for binary options market functions.
func NewBinaryOptionsMsgServerImpl(keeper *Keeper) BinaryOptionsMsgServer {
	return BinaryOptionsMsgServer{
		Keeper: keeper,
	}
}

func (k BinaryOptionsMsgServer) InstantBinaryOptionsMarketLaunch(
	c context.Context, msg *v2.MsgInstantBinaryOptionsMarketLaunch,
) (*v2.MsgInstantBinaryOptionsMarketLaunchResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "InstantBinaryOptionsMarketLaunch")()

	if isBinaryOptionsDisabled(ctx) {
		return nil, binaryOptionsDisabledError()
	}

	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	fee := k.GetCachedParams(ctx).BinaryOptionsMarketInstantListingFee
	if err := k.DistributionKeeper.FundCommunityPool(ctx, sdk.Coins{fee}, senderAddr); err != nil {

		k.Logger(ctx).Error("failed launching binary options market", err)
		return nil, err
	}

	if err := k.checkDenomMinNotional(ctx, senderAddr, msg.QuoteDenom, msg.MinNotional); err != nil {
		return nil, err
	}

	// check if the market launch proposal already exists
	marketID := types.NewBinaryOptionsMarketID(msg.Ticker, msg.QuoteDenom, msg.OracleSymbol, msg.OracleProvider, msg.OracleType)
	if k.checkIfMarketLaunchProposalExist(
		ctx, marketID, types.ProposalTypeBinaryOptionsMarketLaunch, v2.ProposalTypeBinaryOptionsMarketLaunch,
	) {

		ctx.Logger().Info("the binary options market launch proposal already exists", "marketID", marketID.Hex())
		return nil, errors.Wrapf(
			types.ErrMarketLaunchProposalAlreadyExists,
			"the binary options market launch proposal already exists: marketID=%s", marketID.Hex(),
		)
	}

	_, err := k.BinaryOptionsMarketLaunch(
		ctx,
		msg.Ticker,
		msg.OracleSymbol,
		msg.OracleProvider,
		msg.OracleType,
		msg.OracleScaleFactor,
		msg.MakerFeeRate,
		msg.TakerFeeRate,
		msg.ExpirationTimestamp,
		msg.SettlementTimestamp,
		msg.Admin,
		msg.QuoteDenom,
		msg.MinPriceTickSize,
		msg.MinQuantityTickSize,
		msg.MinNotional,
		msg.OpenNotionalCap,
	)

	if err != nil {

		k.Logger(ctx).Error("failed launching binary options market", err)
		return nil, err
	}

	return &v2.MsgInstantBinaryOptionsMarketLaunchResponse{}, nil
}

func (k BinaryOptionsMsgServer) CreateBinaryOptionsLimitOrder(
	c context.Context, msg *v2.MsgCreateBinaryOptionsLimitOrder,
) (*v2.MsgCreateBinaryOptionsLimitOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "CreateBinaryOptionsLimitOrder")()

	if isBinaryOptionsDisabled(ctx) {
		return nil, binaryOptionsDisabledError()
	}

	if k.IsFixedGasEnabled() {
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateBinaryOptionsLimitOrder")
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	account, _ := sdk.AccAddressFromBech32(msg.Sender)

	market := k.GetBinaryOptionsMarket(ctx, msg.Order.MarketID(), true)
	if market == nil {
		k.Logger(ctx).Error("active binary options market doesn't exist", "marketId", msg.Order.MarketId)

		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "marketID %s", msg.Order.MarketId)
	}

	requiredMargin := msg.Order.GetRequiredBinaryOptionsMargin(market.OracleScaleFactor)
	if msg.Order.Margin.GT(requiredMargin) {
		// decrease order margin to the required amount if greater, since there's no need to overpay
		msg.Order.Margin = requiredMargin
	}

	orderHash, err := k.CreateDerivativeLimitOrder(ctx, account, &msg.Order, market, math.LegacyDec{})

	if err != nil {

		return nil, err
	}

	return &v2.MsgCreateBinaryOptionsLimitOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.OrderInfo.Cid,
	}, nil
}

func (k BinaryOptionsMsgServer) CreateBinaryOptionsMarketOrder(
	c context.Context, msg *v2.MsgCreateBinaryOptionsMarketOrder,
) (*v2.MsgCreateBinaryOptionsMarketOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "CreateBinaryOptionsMarketOrder")()

	if isBinaryOptionsDisabled(ctx) {
		return nil, binaryOptionsDisabledError()
	}

	if k.IsFixedGasEnabled() {
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCreateBinaryOptionsMarketOrder")
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	account, _ := sdk.AccAddressFromBech32(msg.Sender)

	market := k.GetBinaryOptionsMarket(ctx, msg.Order.MarketID(), true)
	if market == nil {
		k.Logger(ctx).Error("active binary options market doesn't exist", "marketId", msg.Order.MarketId)

		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "marketID %s", msg.Order.MarketId)
	}

	orderHash, results, err := k.CreateBinaryOptionsMarketOrderWithResultsForAtomicExecution(
		ctx,
		account,
		&msg.Order,
		market,
		math.LegacyDec{},
	)
	if err != nil {

		return nil, err
	}

	resp := &v2.MsgCreateBinaryOptionsMarketOrderResponse{
		OrderHash: orderHash.Hex(),
		Cid:       msg.Order.Cid(),
	}

	if results != nil {
		resp.Results = results
	}

	return resp, nil
}

func (k BinaryOptionsMsgServer) CancelBinaryOptionsOrder(
	c context.Context, msg *v2.MsgCancelBinaryOptionsOrder,
) (*v2.MsgCancelBinaryOptionsOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "CancelBinaryOptionsOrder")()

	if k.IsFixedGasEnabled() {
		ctx.GasMeter().ConsumeGas(DetermineGas(msg), "MsgCancelBinaryOptionsOrder")
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	var (
		sender       = sdk.MustAccAddressFromBech32(msg.Sender)
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, msg.SubaccountId)
		marketID     = common.HexToHash(msg.MarketId)
		identifier   = types.GetOrderIdentifier(msg.OrderHash, msg.Cid)
	)

	market := k.GetBinaryOptionsMarketByID(ctx, marketID)
	err := k.CancelDerivativeOrder(ctx, subaccountID, identifier, market, marketID, msg.OrderMask)

	if err != nil {
		k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, msg.OrderHash, msg.Cid, err))
		return nil, err
	}

	return &v2.MsgCancelBinaryOptionsOrderResponse{}, nil
}

func (k BinaryOptionsMsgServer) AdminUpdateBinaryOptionsMarket(
	c context.Context, msg *v2.MsgAdminUpdateBinaryOptionsMarket,
) (*v2.MsgAdminUpdateBinaryOptionsMarketResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "AdminUpdateBinaryOptionsMarket")()

	if isBinaryOptionsDisabled(ctx) {
		return nil, binaryOptionsDisabledError()
	}

	marketID := common.HexToHash(msg.MarketId)
	market := k.GetBinaryOptionsMarketByID(ctx, marketID)

	if market == nil {
		k.Logger(ctx).Error("binary options market doesn't exist", "marketID", msg.MarketId)

		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "marketID %s", msg.MarketId)
	}

	if market.Admin != msg.Sender {
		k.Logger(ctx).Error("message sender is not an admin of binary options market", "sender", msg.Sender, "admin", market.Admin)

		return nil, errors.Wrapf(types.ErrSenderIsNotAnAdmin, "sender %s, admin %s", msg.Sender, market.Admin)
	}

	if market.Status == v2.MarketStatus_Demolished {

		return nil, errors.Wrapf(types.ErrInvalidMarketStatus, "can't update market that was demolished already")
	}

	expTimestamp, settlementTimestamp := market.ExpirationTimestamp, market.SettlementTimestamp

	if msg.ExpirationTimestamp > 0 {
		if msg.ExpirationTimestamp <= ctx.BlockTime().Unix() {

			return nil, errors.Wrapf(types.ErrInvalidExpiry, "expiration timestamp %d is in the past", msg.ExpirationTimestamp)
		}
		if market.Status != v2.MarketStatus_Active {

			return nil, errors.Wrap(types.ErrInvalidExpiry, "cannot change expiration time of an expired market")
		}
		expTimestamp = msg.ExpirationTimestamp
	}

	if msg.SettlementTimestamp > 0 {
		if msg.SettlementTimestamp <= ctx.BlockTime().Unix() {

			return nil, errors.Wrapf(types.ErrInvalidSettlement, "SettlementTimestamp %d should be in future", msg.SettlementTimestamp)
		}
		if msg.SettlementTimestamp <= expTimestamp {

			return nil, errors.Wrap(types.ErrInvalidSettlement, "settlement time must be after expiration time")
		}
		settlementTimestamp = msg.SettlementTimestamp
	}

	if expTimestamp >= settlementTimestamp {
		return nil, errors.Wrap(types.ErrInvalidExpiry, "expiration timestamp should be prior to settlement timestamp")
	}

	// we convert it to UpdateProposal type to not duplicate the code
	newParams := v2.BinaryOptionsMarketParamUpdateProposal{
		MarketId:            msg.MarketId,
		Status:              msg.Status,
		ExpirationTimestamp: msg.ExpirationTimestamp,
		SettlementTimestamp: msg.SettlementTimestamp,
		SettlementPrice:     msg.SettlementPrice,
	}
	// schedule market param change in transient store
	if err := k.ScheduleBinaryOptionsMarketParamUpdate(ctx, &newParams); err != nil {
		return nil, err
	}

	return &v2.MsgAdminUpdateBinaryOptionsMarketResponse{}, nil
}

func (k BinaryOptionsMsgServer) BatchCancelBinaryOptionsOrders(
	c context.Context, msg *v2.MsgBatchCancelBinaryOptionsOrders,
) (*v2.MsgBatchCancelBinaryOptionsOrdersResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "BatchCancelBinaryOptionsOrders")()

	successes := make([]bool, len(msg.Data))
	for idx := range msg.Data {
		if _, err := k.CancelBinaryOptionsOrder(ctx, &v2.MsgCancelBinaryOptionsOrder{
			Sender:       msg.Sender,
			MarketId:     msg.Data[idx].MarketId,
			SubaccountId: msg.Data[idx].SubaccountId,
			OrderHash:    msg.Data[idx].OrderHash,
			Cid:          msg.Data[idx].Cid,
		}); err != nil {

		} else {
			successes[idx] = true
		}
	}

	return &v2.MsgBatchCancelBinaryOptionsOrdersResponse{Success: successes}, nil
}

func (k DerivativesMsgServer) LaunchBinaryOptionsMarket(
	c context.Context, msg *v2.MsgBinaryOptionsMarketLaunch,
) (*v2.MsgBinaryOptionsMarketLaunchResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "LaunchBinaryOptionsMarket")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleBinaryOptionsMarketLaunchProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgBinaryOptionsMarketLaunchResponse{}, nil
}

func (k DerivativesMsgServer) BinaryOptionsMarketParamUpdate(
	c context.Context, msg *v2.MsgBinaryOptionsMarketParamUpdate,
) (*v2.MsgBinaryOptionsMarketParamUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "BinaryOptionsMarketParamUpdate")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleBinaryOptionsMarketParamUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgBinaryOptionsMarketParamUpdateResponse{}, nil
}
