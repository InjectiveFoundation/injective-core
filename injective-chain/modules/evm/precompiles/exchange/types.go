package exchange

import (
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

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
Inputs
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

func CastApproveParams(methodInputs abi.Arguments, values []any) (
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

func CastRevokeParams(args []any) (common.Address, []exchangetypesv2.MsgType, error) {
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

func CastAllowanceParams(args []any) (*AllowanceParams, error) {
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

func CastCreateDerivativeOrderParams(
	methodInputs abi.Arguments,
	values []any,
) (sender sdk.Address, order *exchangetypesv2.DerivativeOrder, err error) {
	type SolCreateDerivativeOrderParams struct {
		Sender common.Address
		Order  exchangeabi.IExchangeModuleDerivativeOrder
	}

	var solArgs SolCreateDerivativeOrderParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	orderType, err := parseOrderType(solArgs.Order.OrderType)
	if err != nil {
		return sdk.AccAddress{}, nil, err
	}

	triggerPrice := sdkmath.LegacyNewDecFromBigInt(solArgs.Order.TriggerPrice)

	order = &exchangetypesv2.DerivativeOrder{
		MarketId: solArgs.Order.MarketID,
		OrderInfo: exchangetypesv2.OrderInfo{
			SubaccountId: solArgs.Order.SubaccountID,
			FeeRecipient: solArgs.Order.FeeRecipient,
			Price:        sdkmath.LegacyNewDecFromBigInt(solArgs.Order.Price),
			Quantity:     sdkmath.LegacyNewDecFromBigInt(solArgs.Order.Quantity),
			Cid:          solArgs.Order.Cid,
		},
		OrderType:    orderType,
		Margin:       sdkmath.LegacyNewDecFromBigInt(solArgs.Order.Margin),
		TriggerPrice: &triggerPrice,
	}

	return sender, order, nil
}

func CastCreateSpotOrderParams(methodInputs abi.Arguments, values []any) (sender sdk.Address, order *exchangetypesv2.SpotOrder, err error) {
	type SolCreateSpotOrderParams struct {
		Sender common.Address
		Order  exchangeabi.IExchangeModuleSpotOrder
	}

	var solArgs SolCreateSpotOrderParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	orderType, err := parseOrderType(solArgs.Order.OrderType)
	if err != nil {
		return sdk.AccAddress{}, nil, err
	}

	triggerPrice := sdkmath.LegacyNewDecFromBigInt(solArgs.Order.TriggerPrice)

	order = &exchangetypesv2.SpotOrder{
		MarketId: solArgs.Order.MarketID,
		OrderInfo: exchangetypesv2.OrderInfo{
			SubaccountId: solArgs.Order.SubaccountID,
			FeeRecipient: solArgs.Order.FeeRecipient,
			Price:        sdkmath.LegacyNewDecFromBigInt(solArgs.Order.Price),
			Quantity:     sdkmath.LegacyNewDecFromBigInt(solArgs.Order.Quantity),
			Cid:          solArgs.Order.Cid,
		},
		OrderType:    orderType,
		TriggerPrice: &triggerPrice,
	}

	return sender, order, nil
}

func CastCreateDerivativeOrdersParams(
	methodInputs abi.Arguments,
	values []any,
) (sender sdk.Address, orders []exchangetypesv2.DerivativeOrder, err error) {
	type SolCreateDerivativeOrdersParams struct {
		Sender common.Address
		Orders []exchangeabi.IExchangeModuleDerivativeOrder
	}

	var solArgs SolCreateDerivativeOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	derivativeOrders := []exchangetypesv2.DerivativeOrder{}

	for _, order := range solArgs.Orders {
		orderType, err := parseOrderType(order.OrderType)
		if err != nil {
			return sdk.AccAddress{}, nil, err
		}
		triggerPrice := sdkmath.LegacyNewDecFromBigInt(order.TriggerPrice)
		derivativeOrders = append(
			derivativeOrders,
			exchangetypesv2.DerivativeOrder{
				MarketId: order.MarketID,
				OrderInfo: exchangetypesv2.OrderInfo{
					SubaccountId: order.SubaccountID,
					FeeRecipient: order.FeeRecipient,
					Price:        sdkmath.LegacyNewDecFromBigInt(order.Price),
					Quantity:     sdkmath.LegacyNewDecFromBigInt(order.Quantity),
					Cid:          order.Cid,
				},
				OrderType:    orderType,
				Margin:       sdkmath.LegacyNewDecFromBigInt(order.Margin),
				TriggerPrice: &triggerPrice,
			},
		)
	}

	return sender, derivativeOrders, nil
}

func CastCreateSpotOrdersParams(
	methodInputs abi.Arguments,
	values []any,
) (sender sdk.Address, orders []exchangetypesv2.SpotOrder, err error) {
	type SolCreateSpotOrdersParams struct {
		Sender common.Address
		Orders []exchangeabi.IExchangeModuleSpotOrder
	}

	var solArgs SolCreateSpotOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	spotOrders := []exchangetypesv2.SpotOrder{}

	for _, order := range solArgs.Orders {
		orderType, err := parseOrderType(order.OrderType)
		if err != nil {
			return sdk.AccAddress{}, nil, err
		}
		triggerPrice := sdkmath.LegacyNewDecFromBigInt(order.TriggerPrice)
		spotOrders = append(
			spotOrders,
			exchangetypesv2.SpotOrder{
				MarketId: order.MarketID,
				OrderInfo: exchangetypesv2.OrderInfo{
					SubaccountId: order.SubaccountID,
					FeeRecipient: order.FeeRecipient,
					Price:        sdkmath.LegacyNewDecFromBigInt(order.Price),
					Quantity:     sdkmath.LegacyNewDecFromBigInt(order.Quantity),
					Cid:          order.Cid,
				},
				OrderType:    orderType,
				TriggerPrice: &triggerPrice,
			},
		)
	}

	return sender, spotOrders, nil
}

func CastBatchCancelOrdersParams(
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

func CastBatchUpdateOrdersParams(
	methodInputs abi.Arguments,
	values []any,
) (sender sdk.AccAddress, msg *exchangetypesv2.MsgBatchUpdateOrders, err error) {
	type SolBatchUpdateOrdersParams struct {
		Sender  common.Address
		Request exchangeabi.IExchangeModuleBatchUpdateOrdersRequest
	}

	var solArgs SolBatchUpdateOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return sdk.AccAddress{}, nil, err
	}

	sender = sdk.AccAddress(solArgs.Sender.Bytes())

	spotOrdersToCancelPointers := castOrderData(solArgs.Request.SpotOrdersToCancel)

	spotOrdersToCreatePointers := castSpotOrders(solArgs.Request.SpotOrdersToCreate)

	derivativeOrdersToCancelPointers := castOrderData(solArgs.Request.DerivativeOrdersToCancel)

	derivativeOrdersToCreatePointers := castDerivativeOrders(solArgs.Request.DerivativeOrdersToCreate)

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

	return sender, msg, nil
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

func castSpotOrders(orders []exchangeabi.IExchangeModuleSpotOrder) []*exchangetypesv2.SpotOrder {
	spotOrders := []*exchangetypesv2.SpotOrder{}
	for _, order := range orders {
		orderType, err := parseOrderType(order.OrderType)
		if err != nil {
			return nil
		}
		triggerPrice := sdkmath.LegacyNewDecFromBigInt(order.TriggerPrice)
		spotOrders = append(
			spotOrders,
			&exchangetypesv2.SpotOrder{
				MarketId: order.MarketID,
				OrderInfo: exchangetypesv2.OrderInfo{
					SubaccountId: order.SubaccountID,
					FeeRecipient: order.FeeRecipient,
					Price:        sdkmath.LegacyNewDecFromBigInt(order.Price),
					Quantity:     sdkmath.LegacyNewDecFromBigInt(order.Quantity),
					Cid:          order.Cid,
				},
				OrderType:    orderType,
				TriggerPrice: &triggerPrice,
			},
		)
	}
	return spotOrders
}

func castDerivativeOrders(orders []exchangeabi.IExchangeModuleDerivativeOrder) []*exchangetypesv2.DerivativeOrder {
	derivativeOrders := []*exchangetypesv2.DerivativeOrder{}
	for _, order := range orders {
		orderType, err := parseOrderType(order.OrderType)
		if err != nil {
			return nil
		}
		triggerPrice := sdkmath.LegacyNewDecFromBigInt(order.TriggerPrice)
		derivativeOrders = append(
			derivativeOrders,
			&exchangetypesv2.DerivativeOrder{
				MarketId: order.MarketID,
				OrderInfo: exchangetypesv2.OrderInfo{
					SubaccountId: order.SubaccountID,
					FeeRecipient: order.FeeRecipient,
					Price:        sdkmath.LegacyNewDecFromBigInt(order.Price),
					Quantity:     sdkmath.LegacyNewDecFromBigInt(order.Quantity),
					Cid:          order.Cid,
				},
				OrderType:    orderType,
				Margin:       sdkmath.LegacyNewDecFromBigInt(order.Margin),
				TriggerPrice: &triggerPrice,
			},
		)
	}
	return derivativeOrders
}

func CastQueryDerivativeOrdersRequest(
	methodInputs abi.Arguments,
	values []any,
) (query *exchangetypesv2.QueryDerivativeOrdersByHashesRequest, err error) {
	type SolQueryDerivativeOrdersParams struct {
		Request exchangeabi.IExchangeModuleDerivativeOrdersRequest
	}

	var solArgs SolQueryDerivativeOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return nil, err
	}

	query = &exchangetypesv2.QueryDerivativeOrdersByHashesRequest{
		MarketId:     solArgs.Request.MarketID,
		SubaccountId: solArgs.Request.SubaccountID,
		OrderHashes:  solArgs.Request.OrderHashes,
	}

	return query, nil
}

func CastQuerySpotOrdersRequest(
	methodInputs abi.Arguments,
	values []any,
) (query *exchangetypesv2.QuerySpotOrdersByHashesRequest, err error) {
	type SolQuerySpotOrdersParams struct {
		Request exchangeabi.IExchangeModuleDerivativeOrdersRequest
	}

	var solArgs SolQuerySpotOrdersParams
	if err := methodInputs.Copy(&solArgs, values); err != nil {
		return nil, err
	}

	query = &exchangetypesv2.QuerySpotOrdersByHashesRequest{
		MarketId:     solArgs.Request.MarketID,
		SubaccountId: solArgs.Request.SubaccountID,
		OrderHashes:  solArgs.Request.OrderHashes,
	}

	return query, nil
}

/*******************************************************************************
Outputs
*******************************************************************************/

func convertCreateDerivativeMarketOrderResponse(
	in exchangetypesv2.MsgCreateDerivativeMarketOrderResponse,
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
		res.Quantity = precompiletypes.ConvertLegacyDecToBigInt(in.Results.Quantity)
		res.Price = precompiletypes.ConvertLegacyDecToBigInt(in.Results.Price)
		res.Fee = precompiletypes.ConvertLegacyDecToBigInt(in.Results.Fee)
		res.Payout = precompiletypes.ConvertLegacyDecToBigInt(in.Results.Payout)
		res.DeltaIsLong = in.Results.PositionDelta.IsLong
		res.DeltaExecutionPrice = precompiletypes.ConvertLegacyDecToBigInt(in.Results.PositionDelta.ExecutionPrice)
		res.DeltaExecutionQuantity = precompiletypes.ConvertLegacyDecToBigInt(in.Results.PositionDelta.ExecutionQuantity)
		res.DeltaExecutionMargin = precompiletypes.ConvertLegacyDecToBigInt(in.Results.PositionDelta.ExecutionMargin)
	}
	return res
}

func convertTrimmedDerivativeOrders(
	orders []*exchangetypesv2.TrimmedDerivativeLimitOrder,
) []exchangeabi.IExchangeModuleTrimmedDerivativeLimitOrder {
	solOrders := []exchangeabi.IExchangeModuleTrimmedDerivativeLimitOrder{}

	for _, order := range orders {
		solOrders = append(solOrders, exchangeabi.IExchangeModuleTrimmedDerivativeLimitOrder{
			Price:     precompiletypes.ConvertLegacyDecToBigInt(order.Price),
			Quantity:  precompiletypes.ConvertLegacyDecToBigInt(order.Quantity),
			Margin:    precompiletypes.ConvertLegacyDecToBigInt(order.Margin),
			Fillable:  precompiletypes.ConvertLegacyDecToBigInt(order.Fillable),
			IsBuy:     order.IsBuy,
			OrderHash: order.OrderHash,
			Cid:       order.Cid,
		})
	}

	return solOrders
}

func convertTrimmedSpotOrders(
	orders []*exchangetypesv2.TrimmedSpotLimitOrder,
) []exchangeabi.IExchangeModuleTrimmedSpotLimitOrder {
	solOrders := []exchangeabi.IExchangeModuleTrimmedSpotLimitOrder{}

	for _, order := range orders {
		solOrders = append(solOrders, exchangeabi.IExchangeModuleTrimmedSpotLimitOrder{
			Price:     precompiletypes.ConvertLegacyDecToBigInt(order.Price),
			Quantity:  precompiletypes.ConvertLegacyDecToBigInt(order.Quantity),
			Fillable:  precompiletypes.ConvertLegacyDecToBigInt(order.Fillable),
			IsBuy:     order.IsBuy,
			OrderHash: order.OrderHash,
			Cid:       order.Cid,
		})
	}

	return solOrders
}

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

func convertSubaccountPositionsResponse(
	resp *exchangetypesv2.QuerySubaccountPositionsResponse,
) []exchangeabi.IExchangeModuleDerivativePosition {
	solResults := []exchangeabi.IExchangeModuleDerivativePosition{}

	for _, pos := range resp.State {
		solPos := exchangeabi.IExchangeModuleDerivativePosition{
			SubaccountID: pos.SubaccountId,
			MarketID:     pos.MarketId,
		}
		if pos.Position != nil {
			solPos.IsLong = pos.Position.IsLong
			solPos.Quantity = precompiletypes.ConvertLegacyDecToBigInt(pos.Position.Quantity)
			solPos.EntryPrice = precompiletypes.ConvertLegacyDecToBigInt(pos.Position.EntryPrice)
			solPos.Margin = precompiletypes.ConvertLegacyDecToBigInt(pos.Position.Margin)
			solPos.CumulativeFundingEntry = precompiletypes.ConvertLegacyDecToBigInt(pos.Position.CumulativeFundingEntry)
		}
		solResults = append(solResults, solPos)
	}

	return solResults
}

func convertCreateSpotMarketOrderResponse(
	in exchangetypesv2.MsgCreateSpotMarketOrderResponse,
) exchangeabi.IExchangeModuleCreateSpotMarketOrderResponse {
	res := exchangeabi.IExchangeModuleCreateSpotMarketOrderResponse{
		OrderHash: in.OrderHash,
		Cid:       in.Cid,
		Quantity:  big.NewInt(0),
		Price:     big.NewInt(0),
		Fee:       big.NewInt(0),
	}

	if in.Results != nil {
		res.Quantity = precompiletypes.ConvertLegacyDecToBigInt(in.Results.Quantity)
		res.Price = precompiletypes.ConvertLegacyDecToBigInt(in.Results.Price)
		res.Fee = precompiletypes.ConvertLegacyDecToBigInt(in.Results.Fee)

	}
	return res
}
