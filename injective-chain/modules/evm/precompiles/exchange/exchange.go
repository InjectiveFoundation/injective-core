package exchange

import (
	"errors"
	"math/big"
	"time"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/bindings/cosmos/precompile/exchange"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/types"

	storetypes "cosmossdk.io/store/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypesv1 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangetypesv2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

const (
	// Auth Methods
	ApproveMethodName  = "approve"
	RevokeMethodName   = "revoke"
	AllowanceQueryName = "allowance"

	// Account Transactions
	DepositMethodName                = "deposit"
	WithdrawMethodName               = "withdraw"
	SubaccountTransferMethodName     = "subaccountTransfer"
	ExternalTransferMethodName       = "externalTransfer"
	IncreasePositionMarginMethodName = "increasePositionMargin"
	DecreasePositionMarginMethodName = "decreasePositionMargin"
	BatchUpdateOrdersMethodName      = "batchUpdateOrders"

	// Account Queries
	SubaccountDepositQueryMethodName   = "subaccountDeposit"
	SubaccountDepositsQueryMethodName  = "subaccountDeposits"
	SubaccountPositionsQueryMethodName = "subaccountPositions"

	// Derivative Transactions
	CreateDerivativeLimitOrderMethodName       = "createDerivativeLimitOrder"
	BatchCreateDerivativeLimitOrdersMethodName = "batchCreateDerivativeLimitOrders"
	CreateDerivativeMarketOrderMethodName      = "createDerivativeMarketOrder"
	CancelDerivativeOrderMethodName            = "cancelDerivativeOrder"
	BatchCancelDerivativeOrdersMethodName      = "batchCancelDerivativeOrders"

	// Derivative Queries
	DerivativeOrdersByHashesQueryMethodName = "derivativeOrdersByHashes"

	// Spot Transactions
	CreateSpotLimitOrderMethodName       = "createSpotLimitOrder"
	BatchCreateSpotLimitOrdersMethodName = "batchCreateSpotLimitOrders"
	CreateSpotMarketOrderMethodName      = "createSpotMarketOrder"
	CancelSpotOrderMethodName            = "cancelSpotOrder"
	BatchCancelSpotOrdersMethodName      = "batchCancelSpotOrders"

	// Spot Queries
	SpotOrdersByHashesQueryMethodName = "spotOrdersByHashes"
)

var (
	exchangeABI                 abi.ABI
	exchangeContractAddress     = common.BytesToAddress([]byte{101})
	exchangeGasRequiredByMethod = map[[4]byte]uint64{}
)

var (
	ErrPrecompilePanic = errors.New("precompile panic")
)

func init() {
	if err := exchangeABI.UnmarshalJSON([]byte(exchange.ExchangeModuleMetaData.ABI)); err != nil {
		panic(err)
	}
	for methodName := range exchangeABI.Methods {
		var methodID [4]byte
		copy(methodID[:], exchangeABI.Methods[methodName].ID[:4])
		switch methodName {
		case ApproveMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case RevokeMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case DepositMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case WithdrawMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case SubaccountTransferMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case ExternalTransferMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case CreateDerivativeLimitOrderMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case BatchCreateDerivativeLimitOrdersMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case CreateDerivativeMarketOrderMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case CancelDerivativeOrderMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case BatchCancelDerivativeOrdersMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case IncreasePositionMarginMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case DecreasePositionMarginMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case BatchUpdateOrdersMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case CreateSpotLimitOrderMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case BatchCreateSpotLimitOrdersMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case CreateSpotMarketOrderMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case CancelSpotOrderMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		case BatchCancelSpotOrdersMethodName:
			exchangeGasRequiredByMethod[methodID] = 200_000
		default:
			exchangeGasRequiredByMethod[methodID] = 0
		}
	}
}

type ExchangeContract struct {
	exchangeKeeper      *exchangekeeper.Keeper
	exchangeQueryServer exchangetypesv2.QueryServer
	authzKeeper         *authzkeeper.Keeper
	exchangeMsgServer   exchangetypesv2.MsgServer
	kvGasConfig         storetypes.GasConfig
}

func NewExchangeContract(
	exchangeKeeper *exchangekeeper.Keeper,
	authzKeeper *authzkeeper.Keeper,
	kvGasConfig storetypes.GasConfig,
) vm.PrecompiledContract {
	return &ExchangeContract{
		exchangeKeeper:      exchangeKeeper,
		exchangeQueryServer: exchangekeeper.NewQueryServer(exchangeKeeper),
		authzKeeper:         authzKeeper,
		exchangeMsgServer:   exchangekeeper.NewMsgServerImpl(exchangeKeeper),
		kvGasConfig:         kvGasConfig,
	}
}

func (ec *ExchangeContract) ABI() abi.ABI {
	return exchangeABI
}

func (ec *ExchangeContract) Address() common.Address {
	return exchangeContractAddress
}

func (ec *ExchangeContract) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	// base cost to prevent large input size
	baseCost := uint64(len(input)) * ec.kvGasConfig.WriteCostPerByte
	var methodID [4]byte
	copy(methodID[:], input[:4])
	requiredGas, ok := exchangeGasRequiredByMethod[methodID]
	if ok {
		return requiredGas + baseCost
	}
	return baseCost
}

func (ec *ExchangeContract) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) (output []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrPrecompilePanic
			output = nil
		}
	}()

	// parse input
	methodID := contract.Input[:4]
	method, err := exchangeABI.MethodById(methodID)
	if err != nil {
		return nil, err
	}

	args, err := method.Inputs.Unpack(contract.Input[4:])
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}

	caller := sdk.AccAddress(contract.Caller().Bytes())

	switch method.Name {
	case ApproveMethodName:
		return ec.approve(evm, caller, method, args, readonly)
	case RevokeMethodName:
		return ec.revoke(evm, caller, method, args, readonly)
	case AllowanceQueryName:
		return ec.queryAllowance(evm, evm.Origin, method, args, readonly)
	case DepositMethodName:
		return ec.deposit(evm, caller, method, args, readonly)
	case WithdrawMethodName:
		return ec.withdraw(evm, caller, method, args, readonly)
	case SubaccountTransferMethodName:
		return ec.subaccountTransfer(evm, caller, method, args, readonly)
	case ExternalTransferMethodName:
		return ec.externalTransfer(evm, caller, method, args, readonly)
	case CreateDerivativeLimitOrderMethodName:
		return ec.createDerivativeLimitOrder(evm, caller, method, args, readonly)
	case BatchCreateDerivativeLimitOrdersMethodName:
		return ec.batchCreateDerivativeLimitOrder(evm, caller, method, args, readonly)
	case CreateDerivativeMarketOrderMethodName:
		return ec.createDerivativeMarketOrder(evm, caller, method, args, readonly)
	case CancelDerivativeOrderMethodName:
		return ec.cancelDerivativeOrder(evm, caller, method, args, readonly)
	case BatchCancelDerivativeOrdersMethodName:
		return ec.batchCancelDerivativeOrders(evm, caller, method, args, readonly)
	case IncreasePositionMarginMethodName:
		return ec.increasePositionMargin(evm, caller, method, args, readonly)
	case DecreasePositionMarginMethodName:
		return ec.decreasePositionMargin(evm, caller, method, args, readonly)
	case SubaccountDepositQueryMethodName:
		return ec.querySubaccountDeposit(evm, caller, method, args, readonly)
	case SubaccountDepositsQueryMethodName:
		return ec.querySubaccountDeposits(evm, caller, method, args, readonly)
	case DerivativeOrdersByHashesQueryMethodName:
		return ec.queryDerivativeOrdersByHashes(evm, caller, method, args, readonly)
	case SubaccountPositionsQueryMethodName:
		return ec.querySubaccountPositions(evm, caller, method, args, readonly)
	case BatchUpdateOrdersMethodName:
		return ec.batchUpdateOrders(evm, caller, method, args, readonly)
	case CreateSpotLimitOrderMethodName:
		return ec.createSpotLimitOrder(evm, caller, method, args, readonly)
	case BatchCreateSpotLimitOrdersMethodName:
		return ec.batchCreateSpotLimitOrders(evm, caller, method, args, readonly)
	case CreateSpotMarketOrderMethodName:
		return ec.createSpotMarketOrder(evm, caller, method, args, readonly)
	case CancelSpotOrderMethodName:
		return ec.cancelSpotOrder(evm, caller, method, args, readonly)
	case BatchCancelSpotOrdersMethodName:
		return ec.batchCancelSpotOrders(evm, caller, method, args, readonly)
	case SpotOrdersByHashesQueryMethodName:
		return ec.querySpotOrdersByHashes(evm, caller, method, args, readonly)

	default:
		return nil, errors.New("unknown method")
	}
}

/*******************************************************************************
AUTHZ TRANSACTIONS
*******************************************************************************/

func (ec *ExchangeContract) approve(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	params, err := CastApproveParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB.(precompiles.ExtStateDB)

	for _, auth := range params.Authorizations {
		err = stateDB.ExecuteNativeAction(
			common.Address{},
			nil,
			func(ctx sdk.Context) (err error) {
				blockTime := ctx.BlockTime()
				expiration := blockTime.Add(time.Duration(auth.DurationSeconds) * time.Second)

				grant, err := authz.NewGrant(
					blockTime,
					exchangetypesv2.NewGenericExchangeAuthorization(auth.MsgType.URL(), auth.SpendLimit),
					&expiration,
				)
				if err != nil {
					return err
				}

				_, err = ec.authzKeeper.Grant(
					ctx,
					&authz.MsgGrant{
						Granter: caller.String(),
						Grantee: sdk.AccAddress(params.Grantee.Bytes()).String(),
						Grant:   grant,
					},
				)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) revoke(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	grantee, msgTypes, err := CastRevokeParams(args)
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB.(precompiles.ExtStateDB)

	for _, msgType := range msgTypes {
		err = stateDB.ExecuteNativeAction(
			common.Address{},
			nil,
			func(ctx sdk.Context) (err error) {
				_, err = ec.authzKeeper.Revoke(
					ctx,
					&authz.MsgRevoke{
						Granter:    caller.String(),
						Grantee:    sdk.AccAddress(grantee.Bytes()).String(),
						MsgTypeUrl: msgType.URL(),
					},
				)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return method.Outputs.Pack(true)
}

/*******************************************************************************
AUTHZ QUERIES
*******************************************************************************/

func (ec *ExchangeContract) queryAllowance(
	evm *vm.EVM,
	_ common.Address,
	method *abi.Method,
	args []interface{},
	_ bool,
) ([]byte, error) {
	params, err := CastAllowanceParams(args)
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB.(precompiles.ExtStateDB)

	var auth authz.Authorization
	var expiration *time.Time
	err = stateDB.ExecuteNativeAction(
		common.Address{},
		nil,
		func(ctx sdk.Context) (err error) {
			auth, expiration = ec.authzKeeper.GetAuthorization(
				ctx,
				sdk.AccAddress(params.Grantee.Bytes()),
				sdk.AccAddress(params.Granter.Bytes()),
				params.MsgType.URL(),
			)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	res := false
	blockTime := stateDB.Context().BlockTime()
	if auth != nil && (expiration == nil || blockTime.Before(*expiration)) {
		res = true
	}

	return method.Outputs.Pack(res)
}

/*******************************************************************************
ACCOUNT TRANSACTIONS
*******************************************************************************/

func (ec *ExchangeContract) deposit(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	subaccountID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	denom, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[3])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgDeposit{
		Sender:       sender.String(),
		SubaccountId: subaccountID,
		Amount: sdk.NewCoin(
			denom,
			sdkmath.NewIntFromBigInt(amount),
		),
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, sdk.Coins{msg.Amount})
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgDepositResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) withdraw(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	subaccountID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	denom, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[3])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgWithdraw{
		Sender:       sender.String(),
		SubaccountId: subaccountID,
		Amount: sdk.NewCoin(
			denom,
			sdkmath.NewIntFromBigInt(amount),
		),
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, sdk.Coins{msg.Amount})
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgWithdrawResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) subaccountTransfer(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	sourceSubaccountID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	destinationSubaccountID, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	denom, err := types.CastString(args[3])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[4])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgSubaccountTransfer{
		Sender:                  sender.String(),
		SourceSubaccountId:      sourceSubaccountID,
		DestinationSubaccountId: destinationSubaccountID,
		Amount: sdk.NewCoin(
			denom,
			sdkmath.NewIntFromBigInt(amount),
		),
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, sdk.Coins{msg.Amount})
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgSubaccountTransferResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) externalTransfer(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	sourceSubaccountID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	destinationSubaccountID, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	denom, err := types.CastString(args[3])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[4])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgExternalTransfer{
		Sender:                  sender.String(),
		SourceSubaccountId:      sourceSubaccountID,
		DestinationSubaccountId: destinationSubaccountID,
		Amount: sdk.NewCoin(
			denom,
			sdkmath.NewIntFromBigInt(amount),
		),
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, sdk.Coins{msg.Amount})
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgExternalTransferResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) increasePositionMargin(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	sourceSubaccountID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	destinationSubaccountID, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	marketID, err := types.CastString(args[3])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[4])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgIncreasePositionMargin{
		Sender:                  sender.String(),
		SourceSubaccountId:      sourceSubaccountID,
		DestinationSubaccountId: destinationSubaccountID,
		MarketId:                marketID,
		Amount:                  sdkmath.LegacyNewDecFromBigInt(amount),
	}

	hold, err := ec.getDerivativeOrderHold(marketID, amount, evm)
	if err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgIncreasePositionMarginResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) decreasePositionMargin(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	sourceSubaccountID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	destinationSubaccountID, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	marketID, err := types.CastString(args[3])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[4])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgDecreasePositionMargin{
		Sender:                  sender.String(),
		SourceSubaccountId:      sourceSubaccountID,
		DestinationSubaccountId: destinationSubaccountID,
		MarketId:                marketID,
		Amount:                  sdkmath.LegacyNewDecFromBigInt(amount),
	}

	hold, err := ec.getDerivativeOrderHold(marketID, amount, evm)
	if err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgDecreasePositionMarginResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) batchUpdateOrders(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	_, msg, err := CastBatchUpdateOrdersParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	spendCoins := sdk.Coins{}
	for _, order := range msg.DerivativeOrdersToCreate {
		hold, err := ec.getDerivativeOrderHold(order.MarketId, order.Margin.BigInt(), evm)
		if err != nil {
			return nil, err
		}
		spendCoins.Add(hold...)
	}
	for _, order := range msg.SpotOrdersToCreate {
		hold, err := ec.getSpotOrderHold(order.MarketId, order, evm)
		if err != nil {
			return nil, err
		}
		spendCoins.Add(hold...)
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, spendCoins)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgBatchUpdateOrdersResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(resp)
}

/*******************************************************************************
ACCOUNT QUERIES
*******************************************************************************/

func (ec *ExchangeContract) querySubaccountDeposit(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	_ bool,
) ([]byte, error) {
	subaccountID, err := types.CastString(args[0])
	if err != nil {
		return nil, err
	}
	denom, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}

	req := &exchangetypesv2.QuerySubaccountDepositRequest{
		SubaccountId: subaccountID,
		Denom:        denom,
	}

	var resp *exchangetypesv2.QuerySubaccountDepositResponse
	err = ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			resp, err = ec.exchangeQueryServer.SubaccountDeposit(ctx, req)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	availableBalance := big.NewInt(0)
	totalBalance := big.NewInt(0)

	if resp != nil && resp.Deposits != nil {
		availableBalance = types.ConvertLegacyDecToBigInt(resp.Deposits.AvailableBalance)
		totalBalance = types.ConvertLegacyDecToBigInt(resp.Deposits.TotalBalance)
	}

	return method.Outputs.Pack(availableBalance, totalBalance)
}

func (ec *ExchangeContract) querySubaccountDeposits(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	_ bool,
) ([]byte, error) {
	subaccountID, err := types.CastString(args[0])
	if err != nil {
		return nil, err
	}
	trader, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	subaccountNonce, err := types.CastUint32(args[2])
	if err != nil {
		return nil, err
	}

	req := &exchangetypesv2.QuerySubaccountDepositsRequest{
		SubaccountId: subaccountID,
	}
	if trader != "" {
		req.Subaccount = &exchangetypesv2.Subaccount{
			Trader:          trader,
			SubaccountNonce: subaccountNonce,
		}
	}

	var resp *exchangetypesv2.QuerySubaccountDepositsResponse
	err = ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			resp, err = ec.exchangeQueryServer.SubaccountDeposits(ctx, req)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	solDeposits := convertAndSortSubaccountDeposits(resp.Deposits)

	return method.Outputs.Pack(solDeposits)
}

func (ec *ExchangeContract) querySubaccountPositions(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	_ bool,
) ([]byte, error) {
	subaccountID, err := types.CastString(args[0])
	if err != nil {
		return nil, err
	}

	req := &exchangetypesv2.QuerySubaccountPositionsRequest{
		SubaccountId: subaccountID,
	}

	var resp *exchangetypesv2.QuerySubaccountPositionsResponse
	err = ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			resp, err = ec.exchangeQueryServer.SubaccountPositions(ctx, req)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	solResults := convertSubaccountPositionsResponse(resp)

	return method.Outputs.Pack(solResults)
}

/*******************************************************************************
DERIVATIVE TRANSACTIONS
*******************************************************************************/

func (ec *ExchangeContract) createDerivativeLimitOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, err := CastCreateDerivativeOrderParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateDerivativeLimitOrder{
		Sender: sender.String(),
		Order:  *order,
	}

	hold, err := ec.getDerivativeOrderHold(
		order.MarketId,
		types.ConvertLegacyDecToBigInt(order.Margin),
		evm,
	)
	if err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgCreateDerivativeLimitOrderResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(resp)
}

func (ec *ExchangeContract) batchCreateDerivativeLimitOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, orders, err := CastCreateDerivativeOrdersParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgBatchCreateDerivativeLimitOrders{
		Sender: sender.String(),
		Orders: orders,
	}

	spendCoins := sdk.Coins{}
	for _, order := range orders {
		hold, err := ec.getDerivativeOrderHold(
			order.MarketId,
			types.ConvertLegacyDecToBigInt(order.Margin),
			evm,
		)
		if err != nil {
			return nil, err
		}
		spendCoins.Add(hold...)
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, spendCoins)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgBatchCreateDerivativeLimitOrdersResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(resp)
}

func (ec *ExchangeContract) createDerivativeMarketOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, err := CastCreateDerivativeOrderParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateDerivativeMarketOrder{
		Sender: sender.String(),
		Order:  *order,
	}

	hold, err := ec.getDerivativeOrderHold(
		order.MarketId,
		types.ConvertLegacyDecToBigInt(order.Margin),
		evm,
	)
	if err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgCreateDerivativeMarketOrderResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	solResp := convertCreateDerivativeMarketOrderResponse(resp)

	return method.Outputs.Pack(solResp)
}

func (ec *ExchangeContract) cancelDerivativeOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	marketID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	subaccountID, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	orderHash, err := types.CastString(args[3])
	if err != nil {
		return nil, err
	}
	orderMask, err := types.CastInt32(args[4])
	if err != nil {
		return nil, err
	}
	cid, err := types.CastString(args[5])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCancelDerivativeOrder{
		Sender:       sender.String(),
		MarketId:     marketID,
		SubaccountId: subaccountID,
		OrderHash:    orderHash,
		OrderMask:    orderMask,
		Cid:          cid,
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, nil)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgCancelDerivativeOrder{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) batchCancelDerivativeOrders(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, data, err := CastBatchCancelOrdersParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgBatchCancelDerivativeOrders{
		Sender: sender.String(),
		Data:   data,
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, nil)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgBatchCancelDerivativeOrdersResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(resp.Success)
}

/*******************************************************************************
DERIVATIVE QUERIES
*******************************************************************************/

func (ec *ExchangeContract) queryDerivativeOrdersByHashes(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	_ bool,
) ([]byte, error) {

	req, err := CastQueryDerivativeOrdersRequest(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	var resp *exchangetypesv2.QueryDerivativeOrdersByHashesResponse
	err = ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			resp, err = ec.exchangeQueryServer.DerivativeOrdersByHashes(ctx, req)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	solOrders := convertTrimmedDerivativeOrders(resp.Orders)

	return method.Outputs.Pack(solOrders)
}

/*******************************************************************************
SPOT TRANSACTIONS
*******************************************************************************/

func (ec *ExchangeContract) createSpotLimitOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, err := CastCreateSpotOrderParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateSpotLimitOrder{
		Sender: sender.String(),
		Order:  *order,
	}

	hold, err := ec.getSpotOrderHold(order.MarketId, order, evm)
	if err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgCreateSpotLimitOrderResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(resp)
}

func (ec *ExchangeContract) batchCreateSpotLimitOrders(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, orders, err := CastCreateSpotOrdersParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgBatchCreateSpotLimitOrders{
		Sender: sender.String(),
		Orders: orders,
	}

	spendCoins := sdk.Coins{}
	for _, order := range orders {
		hold, err := ec.getSpotOrderHold(order.MarketId, &order, evm)
		if err != nil {
			return nil, err
		}
		spendCoins.Add(hold...)
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, spendCoins)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgBatchCreateSpotLimitOrdersResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(resp)
}

func (ec *ExchangeContract) createSpotMarketOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, err := CastCreateSpotOrderParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateSpotMarketOrder{
		Sender: sender.String(),
		Order:  *order,
	}

	hold, err := ec.getSpotOrderHold(order.MarketId, order, evm)
	if err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgCreateSpotMarketOrderResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	solResp := convertCreateSpotMarketOrderResponse(resp)

	return method.Outputs.Pack(solResp)
}

func (ec *ExchangeContract) cancelSpotOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	marketID, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	subaccountID, err := types.CastString(args[2])
	if err != nil {
		return nil, err
	}
	orderHash, err := types.CastString(args[3])
	if err != nil {
		return nil, err
	}
	cid, err := types.CastString(args[4])
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCancelSpotOrder{
		Sender:       sender.String(),
		MarketId:     marketID,
		SubaccountId: subaccountID,
		OrderHash:    orderHash,
		Cid:          cid,
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, nil)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgCancelSpotOrderResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (ec *ExchangeContract) batchCancelSpotOrders(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, data, err := CastBatchCancelOrdersParams(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgBatchCancelSpotOrders{
		Sender: sender.String(),
		Data:   data,
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, nil)
	if err != nil {
		return nil, err
	}

	resp := exchangetypesv2.MsgBatchCancelSpotOrdersResponse{}
	err = resp.Unmarshal(resBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(resp.Success)
}

/*******************************************************************************
DERIVATIVE QUERIES
*******************************************************************************/

func (ec *ExchangeContract) querySpotOrdersByHashes(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	_ bool,
) ([]byte, error) {

	req, err := CastQuerySpotOrdersRequest(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	var resp *exchangetypesv2.QuerySpotOrdersByHashesResponse
	err = ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			resp, err = ec.exchangeQueryServer.SpotOrdersByHashes(ctx, req)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	solOrders := convertTrimmedSpotOrders(resp.Orders)

	return method.Outputs.Pack(solOrders)
}

/******************************************************************************/

func (ec *ExchangeContract) getDerivativeOrderHold(
	marketID string,
	amount *big.Int,
	evm *vm.EVM,
) (sdk.Coins, error) {
	spendCoins := sdk.Coins{}

	err := ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			marketIDHash := common.HexToHash(marketID)
			market := ec.exchangeKeeper.GetDerivativeMarketByID(
				ctx,
				marketIDHash,
			)
			if market == nil {
				return exchangetypesv1.ErrDerivativeMarketNotFound.Wrapf("derivative market for marketID %s not found", marketID)
			}
			spendCoins = sdk.Coins{
				sdk.NewCoin(market.QuoteDenom, sdkmath.NewIntFromBigInt(amount)),
			}
			return nil
		},
	)

	return spendCoins, err
}

func (ec *ExchangeContract) getSpotOrderHold(
	marketID string,
	order *exchangetypesv2.SpotOrder,
	evm *vm.EVM,
) (sdk.Coins, error) {
	spendCoins := sdk.Coins{}

	err := ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			marketIDHash := common.HexToHash(marketID)
			market := ec.exchangeKeeper.GetSpotMarketByID(
				ctx,
				marketIDHash,
			)
			if market == nil {
				return exchangetypesv1.ErrSpotMarketNotFound.Wrapf("spot market for marketID %s not found", marketID)
			}

			amount, denom := order.GetBalanceHoldAndMarginDenom(market)
			spendCoins = sdk.Coins{
				sdk.NewCoin(denom, amount.TruncateInt()),
			}
			return nil
		},
	)

	return spendCoins, err
}

/******************************************************************************/

func (ec *ExchangeContract) validateAndDispatchMsg(
	evm *vm.EVM,
	caller sdk.AccAddress,
	msg sdk.Msg,
	hold sdk.Coins,
) ([]byte, error) {
	if validateBasic, ok := msg.(sdk.HasValidateBasic); ok {
		if err := validateBasic.ValidateBasic(); err != nil {
			return nil, err
		}
	}

	dispatchResults := [][]byte{}
	err := ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			ctx = ctx.WithValue(exchangetypesv2.ContextKeyHold, hold)
			dispatchResults, err = ec.authzKeeper.DispatchActions(ctx, caller, []sdk.Msg{msg})
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return dispatchResults[0], nil
}

func (ec *ExchangeContract) executeNativeAction(evm *vm.EVM, action func(ctx sdk.Context) error) error {
	stateDB := evm.StateDB.(precompiles.ExtStateDB)
	return stateDB.ExecuteNativeAction(
		ec.Address(),
		nil,
		action,
	)
}
