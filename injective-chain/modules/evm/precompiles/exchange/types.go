package exchange

import (
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	exchangeabi "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/bindings/cosmos/precompile/exchange"
	precompiletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/types"
	exchangetypesv2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

var (
	errInvalidNumberOfArgs = "invalid number of arguments: expected %d, got %d"
	errInvalidGranteeArg   = "invalid grantee argument: %v"
	errInvalidGranterArg   = "invalid granter argument: %v"
	errInvalidMethodsArgs  = "invalid methods arguments"
	errEmptyMethodsArgs    = "methods argument cannot be empty"
)

/*
******************************************************************************
Authz
******************************************************************************
*/

type Authorization struct {
	MsgType         exchangetypesv2.MsgType
	SpendLimit      sdk.Coins
	DurationSeconds int64
}

type ApproveParams struct {
	Grantee        sdk.AccAddress
	Authorizations []Authorization
}

func castApproveParams(methodInputs abi.Arguments, values []any) (
	params *ApproveParams,
	err error,
) {
	type SolApprovalParams struct {
		Grantee        common.Address
		Authorizations []exchangeabi.IExchangeModuleAuthorization
	}

	var solArgs SolApprovalParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return nil, err
	}

	res := &ApproveParams{}

	res.Grantee = sdk.AccAddress(solArgs.Grantee.Bytes())

	authorizations := []Authorization{}
	for _, auth := range solArgs.Authorizations {
		authorization := Authorization{
			MsgType:         exchangetypesv2.MsgType(auth.Method),
			DurationSeconds: auth.Duration.Int64(),
		}
		spendLimit := sdk.Coins{}
		for _, coin := range auth.SpendLimit {
			spendLimit = append(spendLimit, sdk.Coin{Denom: coin.Denom, Amount: sdkmath.NewIntFromBigInt(coin.Amount)})
		}
		authorization.SpendLimit = spendLimit
		authorizations = append(authorizations, authorization)
	}
	res.Authorizations = authorizations

	return res, nil
}

func castRevokeParams(args []any) (common.Address, []exchangetypesv2.MsgType, error) {
	if len(args) != 2 {
		return common.Address{}, nil, fmt.Errorf(errInvalidNumberOfArgs, 2, len(args))
	}

	grantee, ok := args[0].(common.Address)
	if !ok || grantee == (common.Address{}) {
		return common.Address{}, nil, fmt.Errorf(errInvalidGranterArg, args[0])
	}

	msgTypeUints, ok := args[1].([]uint8)
	if !ok {
		return common.Address{}, nil, errors.New(errInvalidMethodsArgs)
	}
	msgTypes := []exchangetypesv2.MsgType{}
	for _, msgTypeUint := range msgTypeUints {
		msgTypes = append(msgTypes, exchangetypesv2.MsgType(msgTypeUint))
	}
	if len(msgTypes) == 0 {
		return common.Address{}, nil, errors.New(errEmptyMethodsArgs)
	}

	return grantee, msgTypes, nil
}

type AllowanceParams struct {
	Grantee common.Address
	Granter common.Address
	MsgType exchangetypesv2.MsgType
}

func castAllowanceParams(args []any) (*AllowanceParams, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf(errInvalidNumberOfArgs, 3, len(args))
	}

	grantee, ok := args[0].(common.Address)
	if !ok || grantee == (common.Address{}) {
		return nil, fmt.Errorf(errInvalidGranteeArg, args[0])
	}

	granter, ok := args[1].(common.Address)
	if !ok || granter == (common.Address{}) {
		return nil, fmt.Errorf(errInvalidGranterArg, args[1])
	}

	msgTypeUint, ok := args[2].(uint8)
	if !ok {
		return nil, errors.New(errInvalidMethodsArgs)
	}
	msgType := exchangetypesv2.MsgType(msgTypeUint)

	return &AllowanceParams{grantee, granter, msgType}, nil
}

/*
********************************************************************************
Derivative Orders
********************************************************************************
*/

func (ec *ExchangeContract) castCreateDerivativeOrderParams(
	methodInputs abi.Arguments,
	values []any,
	evm *vm.EVM,
) (
	sdk.Address,
	*exchangetypesv2.DerivativeOrder,
	sdk.Coins,
	*exchangetypesv2.DerivativeMarket,
	error,
) {
	type SolCreateDerivativeOrderParams struct {
		Sender common.Address
		Order  exchangeabi.IExchangeModuleDerivativeOrder
	}

	var solArgs SolCreateDerivativeOrderParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, nil, nil, err
	}

	sender := sdk.AccAddress(solArgs.Sender.Bytes())

	order, hold, market, err := ec.castDerivativeOrder(solArgs.Order, evm)
	if err != nil {
		return sdk.AccAddress{}, nil, nil, nil, err
	}

	return sender, order, hold, market, nil
}

func countCreateDerivativeOrdersParams(
	methodInputs abi.Arguments,
	values []any,
) (int, error) {
	type SolCreateDerivativeOrdersParams struct {
		Sender common.Address
		Orders []exchangeabi.IExchangeModuleDerivativeOrder
	}

	var solArgs SolCreateDerivativeOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return 0, err
	}

	return len(solArgs.Orders), nil
}

func (ec *ExchangeContract) castCreateDerivativeOrdersParams(
	methodInputs abi.Arguments,
	values []any,
	evm *vm.EVM,
) (
	sender sdk.Address,
	orders []exchangetypesv2.DerivativeOrder,
	hold sdk.Coins,
	err error,
) {
	type SolCreateDerivativeOrdersParams struct {
		Sender common.Address
		Orders []exchangeabi.IExchangeModuleDerivativeOrder
	}

	var solArgs SolCreateDerivativeOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	derivativeOrdersV2, cumulativeHold, err := ec.castDerivativeOrders(solArgs.Orders, evm)
	if err != nil {
		return sdk.AccAddress{}, nil, nil, err
	}

	return sender, derivativeOrdersV2, cumulativeHold, nil
}

func (ec *ExchangeContract) castDerivativeOrder(
	solOrder exchangeabi.IExchangeModuleDerivativeOrder,
	evm *vm.EVM,
) (
	*exchangetypesv2.DerivativeOrder,
	sdk.Coins,
	*exchangetypesv2.DerivativeMarket,
	error,
) {
	market, err := ec.getDerivativeMarket(solOrder.MarketID, evm)
	if err != nil {
		return nil, nil, nil, err
	}

	humanReadableQuantity := market.QuantityFromChainFormat(sdkmath.LegacyNewDecFromBigInt(solOrder.Quantity))
	humanReadablePrice := market.PriceFromChainFormat(sdkmath.LegacyNewDecFromBigInt(solOrder.Price))
	humanReadableTriggerPrice := market.PriceFromChainFormat(sdkmath.LegacyNewDecFromBigInt(solOrder.TriggerPrice))
	humanReadableMargin := market.NotionalFromChainFormat(sdkmath.LegacyNewDecFromBigInt(solOrder.Margin))

	orderType, err := parseOrderType(solOrder.OrderType)
	if err != nil {
		return nil, nil, nil, err
	}

	orderV2 := &exchangetypesv2.DerivativeOrder{
		MarketId: solOrder.MarketID,
		OrderInfo: exchangetypesv2.OrderInfo{
			SubaccountId: solOrder.SubaccountID,
			FeeRecipient: solOrder.FeeRecipient,
			Price:        humanReadablePrice,
			Quantity:     humanReadableQuantity,
			Cid:          solOrder.Cid,
		},
		OrderType:    orderType,
		Margin:       humanReadableMargin,
		TriggerPrice: &humanReadableTriggerPrice,
	}

	hold := sdk.Coins{
		sdk.NewCoin(
			market.QuoteDenom,
			sdkmath.NewIntFromBigInt(solOrder.Margin),
		),
	}

	return orderV2, hold, market, nil
}

func (ec *ExchangeContract) castDerivativeOrders(
	solOrders []exchangeabi.IExchangeModuleDerivativeOrder,
	evm *vm.EVM,
) (
	[]exchangetypesv2.DerivativeOrder,
	sdk.Coins,
	error,
) {
	ordersV2 := []exchangetypesv2.DerivativeOrder{}
	cumulativeHold := sdk.Coins{}
	for _, solOrder := range solOrders {
		orderV2, hold, _, err := ec.castDerivativeOrder(solOrder, evm)
		if err != nil {
			return nil, nil, err
		}
		ordersV2 = append(ordersV2, *orderV2)
		cumulativeHold = cumulativeHold.Add(hold...)
	}
	return ordersV2, cumulativeHold, nil
}

func (ec *ExchangeContract) castQueryDerivativeOrdersRequest(
	methodInputs abi.Arguments,
	values []any,
	evm *vm.EVM,
) (
	query *exchangetypesv2.QueryDerivativeOrdersByHashesRequest,
	market *exchangetypesv2.DerivativeMarket,
	err error,
) {
	type SolQueryDerivativeOrdersParams struct {
		Request exchangeabi.IExchangeModuleDerivativeOrdersRequest
	}

	var solArgs SolQueryDerivativeOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return nil, nil, err
	}

	market, err = ec.getDerivativeMarket(solArgs.Request.MarketID, evm)
	if err != nil {
		return nil, nil, err
	}

	query = &exchangetypesv2.QueryDerivativeOrdersByHashesRequest{
		MarketId:     solArgs.Request.MarketID,
		SubaccountId: solArgs.Request.SubaccountID,
		OrderHashes:  solArgs.Request.OrderHashes,
	}

	return query, market, nil
}

func convertCreateDerivativeMarketOrderResponse(
	in exchangetypesv2.MsgCreateDerivativeMarketOrderResponse,
	market *exchangetypesv2.DerivativeMarket,
) exchangeabi.IExchangeModuleCreateDerivativeMarketOrderResponse {
	res := exchangeabi.IExchangeModuleCreateDerivativeMarketOrderResponse{
		OrderHash:              in.OrderHash,
		Cid:                    in.Cid,
		Quantity:               big.NewInt(0),
		Price:                  big.NewInt(0),
		Fee:                    big.NewInt(0),
		Payout:                 big.NewInt(0),
		DeltaIsLong:            false,
		DeltaExecutionQuantity: big.NewInt(0),
		DeltaExecutionMargin:   big.NewInt(0),
		DeltaExecutionPrice:    big.NewInt(0),
	}

	if in.Results != nil {
		res.Quantity = precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(in.Results.Quantity))
		res.Price = precompiletypes.ConvertLegacyDecToBigInt(market.PriceToChainFormat(in.Results.Price))
		res.Fee = precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(in.Results.Fee))
		res.Payout = precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(in.Results.Payout))
		res.DeltaIsLong = in.Results.PositionDelta.IsLong
		res.DeltaExecutionPrice = precompiletypes.ConvertLegacyDecToBigInt(market.PriceToChainFormat(in.Results.PositionDelta.ExecutionPrice))
		res.DeltaExecutionQuantity = precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(in.Results.PositionDelta.ExecutionQuantity))
		res.DeltaExecutionMargin = precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(in.Results.PositionDelta.ExecutionMargin))
	}
	return res
}

func convertTrimmedDerivativeOrders(
	orders []*exchangetypesv2.TrimmedDerivativeLimitOrder,
	market *exchangetypesv2.DerivativeMarket,
) []exchangeabi.IExchangeModuleTrimmedDerivativeLimitOrder {
	solOrders := []exchangeabi.IExchangeModuleTrimmedDerivativeLimitOrder{}

	for _, order := range orders {
		solOrders = append(solOrders, exchangeabi.IExchangeModuleTrimmedDerivativeLimitOrder{
			Price:     precompiletypes.ConvertLegacyDecToBigInt(market.PriceToChainFormat(order.Price)),
			Quantity:  precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(order.Quantity)),
			Margin:    precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(order.Margin)),
			Fillable:  precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(order.Fillable)),
			IsBuy:     order.IsBuy,
			OrderHash: order.OrderHash,
			Cid:       order.Cid,
		})
	}

	return solOrders
}

/******************************************************************************/
/* Spot Orders
*******************************************************************************/

func (ec *ExchangeContract) castCreateSpotOrderParams(
	methodInputs abi.Arguments,
	values []any,
	evm *vm.EVM,
) (
	sdk.Address,
	*exchangetypesv2.SpotOrder,
	sdk.Coins,
	*exchangetypesv2.SpotMarket,
	error,
) {
	type SolCreateSpotOrderParams struct {
		Sender common.Address
		Order  exchangeabi.IExchangeModuleSpotOrder
	}

	var solArgs SolCreateSpotOrderParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, nil, nil, err
	}

	sender := sdk.AccAddress(solArgs.Sender.Bytes())

	order, hold, market, err := ec.castSpotOrder(solArgs.Order, evm)
	if err != nil {
		return sdk.AccAddress{}, nil, nil, nil, err
	}

	return sender, order, hold, market, nil
}

func countCreateSpotOrdersParams(
	methodInputs abi.Arguments,
	values []any,
) (int, error) {
	type SolCreateSpotOrdersParams struct {
		Sender common.Address
		Orders []exchangeabi.IExchangeModuleSpotOrder
	}

	var solArgs SolCreateSpotOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return 0, err
	}

	return len(solArgs.Orders), nil
}

func (ec *ExchangeContract) castCreateSpotOrdersParams(
	methodInputs abi.Arguments,
	values []any,
	evm *vm.EVM,
) (
	sender sdk.Address,
	orders []exchangetypesv2.SpotOrder,
	hold sdk.Coins,
	err error,
) {
	type SolCreateSpotOrdersParams struct {
		Sender common.Address
		Orders []exchangeabi.IExchangeModuleSpotOrder
	}

	var solArgs SolCreateSpotOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	spotOrdersV2, cumulativeHold, err := ec.castSpotOrders(solArgs.Orders, evm)
	if err != nil {
		return sdk.AccAddress{}, nil, nil, err
	}

	return sender, spotOrdersV2, cumulativeHold, nil
}

func (ec *ExchangeContract) castSpotOrder(
	solOrder exchangeabi.IExchangeModuleSpotOrder,
	evm *vm.EVM,
) (
	*exchangetypesv2.SpotOrder,
	sdk.Coins,
	*exchangetypesv2.SpotMarket,
	error,
) {
	market, err := ec.getSpotMarket(solOrder.MarketID, evm)
	if err != nil {
		return nil, nil, nil, err
	}

	humanReadableQuantity := market.QuantityFromChainFormat(sdkmath.LegacyNewDecFromBigInt(solOrder.Quantity))
	humanReadablePrice := market.NotionalFromChainFormat(sdkmath.LegacyNewDecFromBigInt(solOrder.Price))
	humanReadableTriggerPrice := market.NotionalFromChainFormat(sdkmath.LegacyNewDecFromBigInt(solOrder.TriggerPrice))

	orderType, err := parseOrderType(solOrder.OrderType)
	if err != nil {
		return nil, nil, nil, err
	}

	orderV2 := &exchangetypesv2.SpotOrder{
		MarketId: solOrder.MarketID,
		OrderInfo: exchangetypesv2.OrderInfo{
			SubaccountId: solOrder.SubaccountID,
			FeeRecipient: solOrder.FeeRecipient,
			Price:        humanReadablePrice,
			Quantity:     humanReadableQuantity,
			Cid:          solOrder.Cid,
		},
		OrderType:    orderType,
		TriggerPrice: &humanReadableTriggerPrice,
	}

	humanReadableHoldAmount, denom := orderV2.GetBalanceHoldAndMarginDenom(market)
	var chainFormattedHoldAmount math.LegacyDec
	if orderV2.IsBuy() {
		chainFormattedHoldAmount = market.NotionalToChainFormat(humanReadableHoldAmount)
	} else {
		chainFormattedHoldAmount = market.QuantityToChainFormat(humanReadableHoldAmount)
	}
	hold := sdk.Coins{
		sdk.NewCoin(
			denom,
			chainFormattedHoldAmount.TruncateInt(),
		),
	}

	return orderV2, hold, market, nil
}

func (ec *ExchangeContract) castSpotOrders(
	solOrders []exchangeabi.IExchangeModuleSpotOrder,
	evm *vm.EVM,
) (
	[]exchangetypesv2.SpotOrder,
	sdk.Coins,
	error,
) {
	ordersV2 := []exchangetypesv2.SpotOrder{}
	cumulativeHold := sdk.Coins{}
	for _, solOrder := range solOrders {
		orderV2, hold, _, err := ec.castSpotOrder(solOrder, evm)
		if err != nil {
			return nil, nil, err
		}
		ordersV2 = append(ordersV2, *orderV2)
		cumulativeHold = cumulativeHold.Add(hold...)
	}
	return ordersV2, cumulativeHold, nil
}

func (ec *ExchangeContract) castQuerySpotOrdersRequest(
	methodInputs abi.Arguments,
	values []any,
	evm *vm.EVM,
) (
	query *exchangetypesv2.QuerySpotOrdersByHashesRequest,
	market *exchangetypesv2.SpotMarket,
	err error,
) {
	type SolQuerySpotOrdersParams struct {
		Request exchangeabi.IExchangeModuleDerivativeOrdersRequest
	}

	var solArgs SolQuerySpotOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return nil, nil, err
	}

	market, err = ec.getSpotMarket(solArgs.Request.MarketID, evm)
	if err != nil {
		return nil, nil, err
	}

	query = &exchangetypesv2.QuerySpotOrdersByHashesRequest{
		MarketId:     solArgs.Request.MarketID,
		SubaccountId: solArgs.Request.SubaccountID,
		OrderHashes:  solArgs.Request.OrderHashes,
	}

	return query, market, nil
}

func (ec *ExchangeContract) convertCreateSpotMarketOrderResponse(
	in exchangetypesv2.MsgCreateSpotMarketOrderResponse,
	market *exchangetypesv2.SpotMarket,
) exchangeabi.IExchangeModuleCreateSpotMarketOrderResponse {
	res := exchangeabi.IExchangeModuleCreateSpotMarketOrderResponse{
		OrderHash: in.OrderHash,
		Cid:       in.Cid,
		Quantity:  big.NewInt(0),
		Price:     big.NewInt(0),
		Fee:       big.NewInt(0),
	}

	if in.Results != nil {
		res.Quantity = precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(in.Results.Quantity))
		res.Price = precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(in.Results.Price))
		res.Fee = precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(in.Results.Fee))
	}
	return res
}

func (ec *ExchangeContract) convertTrimmedSpotOrders(
	orders []*exchangetypesv2.TrimmedSpotLimitOrder,
	market *exchangetypesv2.SpotMarket,
) []exchangeabi.IExchangeModuleTrimmedSpotLimitOrder {
	solOrders := []exchangeabi.IExchangeModuleTrimmedSpotLimitOrder{}

	for _, order := range orders {
		solOrders = append(solOrders, exchangeabi.IExchangeModuleTrimmedSpotLimitOrder{
			Price:     precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(order.Price)),
			Quantity:  precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(order.Quantity)),
			Fillable:  precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(order.Fillable)),
			IsBuy:     order.IsBuy,
			OrderHash: order.OrderHash,
			Cid:       order.Cid,
		})
	}

	return solOrders
}

/******************************************************************************/

func castBatchCancelOrdersParams(
	methodInputs abi.Arguments,
	values []any,
) (sender sdk.Address, orderDatas []exchangetypesv2.OrderData, err error) {
	type SolBatchCancelParams struct {
		Sender common.Address
		Data   []exchangeabi.IExchangeModuleOrderData
	}

	var solArgs SolBatchCancelParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	data := []exchangetypesv2.OrderData{}
	for _, item := range solArgs.Data {
		data = append(
			data,
			exchangetypesv2.OrderData{
				MarketId:     item.MarketID,
				SubaccountId: item.SubaccountID,
				OrderHash:    item.OrderHash,
				OrderMask:    item.OrderMask,
				Cid:          item.Cid,
			},
		)
	}

	return sender, data, nil
}

type BatchUpdateCount struct {
	DerivativeOrdersToCancel       int
	DerivativeOrdersToCreate       int
	DerivativeMarketIdsToCancelAll int
	SpotOrdersToCancel             int
	SpotOrdersToCreate             int
	SpotMarketIdsToCancelAll       int
}

func countBatchUpdateOrdersParams(
	methodInputs abi.Arguments,
	values []any,
) (*BatchUpdateCount, error) {
	type SolBatchUpdateOrdersParams struct {
		Sender  common.Address
		Request exchangeabi.IExchangeModuleBatchUpdateOrdersRequest
	}

	var solArgs SolBatchUpdateOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return nil, err
	}

	res := &BatchUpdateCount{
		DerivativeOrdersToCancel:       len(solArgs.Request.DerivativeOrdersToCancel),
		DerivativeOrdersToCreate:       len(solArgs.Request.DerivativeOrdersToCreate),
		DerivativeMarketIdsToCancelAll: len(solArgs.Request.DerivativeMarketIDsToCancelAll),
		SpotOrdersToCancel:             len(solArgs.Request.SpotOrdersToCancel),
		SpotOrdersToCreate:             len(solArgs.Request.SpotOrdersToCreate),
		SpotMarketIdsToCancelAll:       len(solArgs.Request.SpotMarketIDsToCancelAll),
	}

	return res, nil
}

func (ec *ExchangeContract) castBatchUpdateOrdersParams(
	methodInputs abi.Arguments,
	values []any,
	evm *vm.EVM,
) (
	sender sdk.AccAddress,
	msg *exchangetypesv2.MsgBatchUpdateOrders,
	hold sdk.Coins,
	err error,
) {
	type SolBatchUpdateOrdersParams struct {
		Sender  common.Address
		Request exchangeabi.IExchangeModuleBatchUpdateOrdersRequest
	}

	var solArgs SolBatchUpdateOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	spotOrdersToCancelPointers := castOrderData(solArgs.Request.SpotOrdersToCancel)

	spotOrdersToCreate, spotOrdersHold, err := ec.castSpotOrders(solArgs.Request.SpotOrdersToCreate, evm)
	if err != nil {
		return sdk.AccAddress{}, nil, nil, err
	}
	spotOrdersToCreatePointers := make([]*exchangetypesv2.SpotOrder, len(spotOrdersToCreate))
	for i, v := range spotOrdersToCreate {
		spotOrdersToCreatePointers[i] = &v
	}

	derivativeOrdersToCancelPointers := castOrderData(solArgs.Request.DerivativeOrdersToCancel)

	derivativeOrdersToCreate, derivativeOrdersHold, err := ec.castDerivativeOrders(solArgs.Request.DerivativeOrdersToCreate, evm)
	if err != nil {
		return sdk.AccAddress{}, nil, nil, err
	}
	derivativeOrdersToCreatePointers := make([]*exchangetypesv2.DerivativeOrder, len(derivativeOrdersToCreate))
	for i, v := range derivativeOrdersToCreate {
		derivativeOrdersToCreatePointers[i] = &v
	}

	totalHold := spotOrdersHold.Add(derivativeOrdersHold...)

	msg = &exchangetypesv2.MsgBatchUpdateOrders{
		Sender:                         sender.String(),
		SubaccountId:                   solArgs.Request.SubaccountID,
		SpotMarketIdsToCancelAll:       solArgs.Request.SpotMarketIDsToCancelAll,
		SpotOrdersToCancel:             spotOrdersToCancelPointers,
		SpotOrdersToCreate:             spotOrdersToCreatePointers,
		DerivativeMarketIdsToCancelAll: solArgs.Request.DerivativeMarketIDsToCancelAll,
		DerivativeOrdersToCancel:       derivativeOrdersToCancelPointers,
		DerivativeOrdersToCreate:       derivativeOrdersToCreatePointers,
	}

	return sender, msg, totalHold, nil
}

func parseOrderType(value string) (exchangetypesv2.OrderType, error) {
	var orderType exchangetypesv2.OrderType
	var err error
	switch value {
	case "buy":
		orderType = exchangetypesv2.OrderType_BUY
	case "buyPostOnly":
		orderType = exchangetypesv2.OrderType_BUY_PO
	case "sell":
		orderType = exchangetypesv2.OrderType_SELL
	case "sellPostOnly":
		orderType = exchangetypesv2.OrderType_SELL_PO
	default:
		err = errors.New("order type must be \"buy\", \"buyPostOnly\", \"sellPostOnly\" or \"sell\"")
	}
	return orderType, err
}

func castOrderData(orderData []exchangeabi.IExchangeModuleOrderData) []*exchangetypesv2.OrderData {
	res := []*exchangetypesv2.OrderData{}
	for _, item := range orderData {
		res = append(res, &exchangetypesv2.OrderData{
			MarketId:     item.MarketID,
			SubaccountId: item.SubaccountID,
			OrderHash:    item.OrderHash,
			OrderMask:    item.OrderMask,
			Cid:          item.Cid,
		})
	}
	return res
}

/*******************************************************************************
* Positions
*******************************************************************************/

func (ec *ExchangeContract) castIncreasePositionParams(
	args []any,
	evm *vm.EVM,
) (
	*exchangetypesv2.MsgIncreasePositionMargin,
	sdk.Coins,
	error,
) {
	if len(args) != 5 {
		return nil, nil, fmt.Errorf(errInvalidNumberOfArgs, 5, len(args))
	}

	sender, err := precompiletypes.CastAddress(args[0])
	if err != nil {
		return nil, nil, err
	}
	sourceSubaccountID, err := precompiletypes.CastString(args[1])
	if err != nil {
		return nil, nil, err
	}
	destinationSubaccountID, err := precompiletypes.CastString(args[2])
	if err != nil {
		return nil, nil, err
	}
	marketID, err := precompiletypes.CastString(args[3])
	if err != nil {
		return nil, nil, err
	}
	chainFormatAmount, err := precompiletypes.CastBigInt(args[4])
	if err != nil {
		return nil, nil, err
	}

	market, err := ec.getDerivativeMarket(marketID, evm)
	if err != nil {
		return nil, nil, err
	}

	humanReadableAmount := market.NotionalFromChainFormat(sdkmath.LegacyNewDecFromBigInt(chainFormatAmount))

	msg := &exchangetypesv2.MsgIncreasePositionMargin{
		Sender:                  sender.String(),
		SourceSubaccountId:      sourceSubaccountID,
		DestinationSubaccountId: destinationSubaccountID,
		MarketId:                marketID,
		Amount:                  humanReadableAmount,
	}

	hold := sdk.Coins{
		sdk.NewCoin(
			market.QuoteDenom,
			sdkmath.NewIntFromBigInt(chainFormatAmount),
		),
	}

	return msg, hold, nil
}

func (ec *ExchangeContract) castDecreasePositionParams(
	args []any,
	evm *vm.EVM,
) (
	*exchangetypesv2.MsgDecreasePositionMargin,
	sdk.Coins,
	error,
) {
	if len(args) != 5 {
		return nil, nil, fmt.Errorf(errInvalidNumberOfArgs, 5, len(args))
	}

	sender, err := precompiletypes.CastAddress(args[0])
	if err != nil {
		return nil, nil, err
	}
	sourceSubaccountID, err := precompiletypes.CastString(args[1])
	if err != nil {
		return nil, nil, err
	}
	destinationSubaccountID, err := precompiletypes.CastString(args[2])
	if err != nil {
		return nil, nil, err
	}
	marketID, err := precompiletypes.CastString(args[3])
	if err != nil {
		return nil, nil, err
	}
	chainFormatAmount, err := precompiletypes.CastBigInt(args[4])
	if err != nil {
		return nil, nil, err
	}

	market, err := ec.getDerivativeMarket(marketID, evm)
	if err != nil {
		return nil, nil, err
	}

	humanReadableAmount := market.NotionalFromChainFormat(sdkmath.LegacyNewDecFromBigInt(chainFormatAmount))

	msg := &exchangetypesv2.MsgDecreasePositionMargin{
		Sender:                  sender.String(),
		SourceSubaccountId:      sourceSubaccountID,
		DestinationSubaccountId: destinationSubaccountID,
		MarketId:                marketID,
		Amount:                  humanReadableAmount,
	}

	hold := sdk.Coins{
		sdk.NewCoin(
			market.QuoteDenom,
			sdkmath.NewIntFromBigInt(chainFormatAmount),
		),
	}

	return msg, hold, nil
}

func (ec *ExchangeContract) convertSubaccountPositionsResponse(
	resp *exchangetypesv2.QuerySubaccountPositionsResponse,
	evm *vm.EVM,
) ([]exchangeabi.IExchangeModuleDerivativePosition, error) {
	solResults := []exchangeabi.IExchangeModuleDerivativePosition{}

	for _, pos := range resp.State {
		market, err := ec.getDerivativeMarket(pos.MarketId, evm)
		if err != nil {
			return nil, err
		}
		solPos := exchangeabi.IExchangeModuleDerivativePosition{
			SubaccountID: pos.SubaccountId,
			MarketID:     pos.MarketId,
		}
		if pos.Position != nil {
			solPos.IsLong = pos.Position.IsLong
			solPos.Quantity = precompiletypes.ConvertLegacyDecToBigInt(market.QuantityToChainFormat(pos.Position.Quantity))
			solPos.EntryPrice = precompiletypes.ConvertLegacyDecToBigInt(market.PriceToChainFormat(pos.Position.EntryPrice))
			solPos.Margin = precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(pos.Position.Margin))
			solPos.CumulativeFundingEntry = precompiletypes.ConvertLegacyDecToBigInt(market.NotionalToChainFormat(pos.Position.CumulativeFundingEntry))
		}
		solResults = append(solResults, solPos)
	}

	return solResults, nil
}

/*******************************************************************************
* Account
*******************************************************************************/

func convertAndSortSubaccountDeposits(
	deposits map[string]*exchangetypesv2.Deposit,
) []exchangeabi.IExchangeModuleSubaccountDepositData {
	solDeposits := []exchangeabi.IExchangeModuleSubaccountDepositData{}

	for denom, deposit := range deposits {
		solDeposits = append(solDeposits, exchangeabi.IExchangeModuleSubaccountDepositData{
			Denom:            denom,
			AvailableBalance: precompiletypes.ConvertLegacyDecToBigInt(deposit.AvailableBalance),
			TotalBalance:     precompiletypes.ConvertLegacyDecToBigInt(deposit.TotalBalance),
		})
	}

	// It is necessary to sort the output of the map iteration because iterating
	// through a map is not deterministic. Even if this is a query, the result
	// needs to be deterministic because it could be called and used by a
	// smart-contract.
	// SortStable only guarantees the original order of equal elements. But the
	// original order (solDeposits, output of map iteration) is not deterministic,
	// so we add a secondary index (denom) to break ties deterministically.
	slices.SortStableFunc(
		solDeposits,
		func(left, right exchangeabi.IExchangeModuleSubaccountDepositData) int {
			if c := left.TotalBalance.Cmp(right.TotalBalance); c != 0 {
				return c
			}
			// Tie-break by denom to make ordering deterministic across runs.
			if left.Denom < right.Denom {
				return -1
			}
			if left.Denom > right.Denom {
				return 1
			}
			// There is maximum one occurrence of each denom in the deposits map
			// so we never get here.
			return 0
		},
	)

	return solDeposits
}
