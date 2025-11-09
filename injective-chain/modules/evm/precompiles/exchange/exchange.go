package exchange

import (
	"errors"
	"math/big"
	"time"

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

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/bindings/cosmos/precompile/exchange"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/types"
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
	exchangeABI             abi.ABI
	exchangeContractAddress = common.BytesToAddress([]byte{101})
)

var (
	ErrPrecompilePanic = errors.New("precompile panic")
)

func init() {
	if err := exchangeABI.UnmarshalJSON([]byte(exchange.ExchangeModuleMetaData.ABI)); err != nil {
		panic(err)
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

func (*ExchangeContract) Name() string {
	return "INJ_EXCHANGE"
}

func (ec *ExchangeContract) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	// base cost to prevent large input size
	cost := uint64(len(input)) * ec.kvGasConfig.WriteCostPerByte

	method, err := exchangeABI.MethodById(input[:4])
	if err != nil {
		return cost
	}

	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return cost
	}

	switch method.Name {
	case ApproveMethodName:
		cost += 200_000
	case RevokeMethodName:
		cost += 200_000
	case DepositMethodName:
		cost += exchangekeeper.MsgDepositGas
	case WithdrawMethodName:
		cost += exchangekeeper.MsgWithdrawGas
	case SubaccountTransferMethodName:
		cost += exchangekeeper.MsgSubaccountTransferGas
	case ExternalTransferMethodName:
		cost += exchangekeeper.MsgExternalTransferGas
	case CreateDerivativeLimitOrderMethodName:
		cost += exchangekeeper.MsgCreateDerivativeLimitOrderGas
	case CreateDerivativeMarketOrderMethodName:
		cost += exchangekeeper.MsgCreateDerivativeMarketOrderGas
	case CancelDerivativeOrderMethodName:
		cost += exchangekeeper.MsgCancelDerivativeOrderGas
	case IncreasePositionMarginMethodName:
		cost += exchangekeeper.MsgIncreasePositionMarginGas
	case DecreasePositionMarginMethodName:
		cost += exchangekeeper.MsgDecreasePositionMarginGas
	case CreateSpotLimitOrderMethodName:
		cost += exchangekeeper.MsgCreateSpotLimitOrderGas
	case CreateSpotMarketOrderMethodName:
		cost += exchangekeeper.MsgCreateSpotMarketOrderGas
	case CancelSpotOrderMethodName:
		cost += exchangekeeper.MsgCancelSpotOrderGas
	}

	switch method.Name {
	case BatchCreateDerivativeLimitOrdersMethodName:
		count, err := countCreateDerivativeOrdersParams(method.Inputs, args)
		if err != nil {
			return cost
		}
		cost += exchangekeeper.MsgCreateDerivativeLimitOrderGas * uint64(count)
	case BatchCancelDerivativeOrdersMethodName:
		_, orders, err := castBatchCancelOrdersParams(method.Inputs, args)
		if err != nil {
			return cost
		}
		cost += exchangekeeper.MsgCancelDerivativeOrderGas * uint64(len(orders))
	case BatchCreateSpotLimitOrdersMethodName:
		count, err := countCreateSpotOrdersParams(method.Inputs, args)
		if err != nil {
			return cost
		}
		cost += exchangekeeper.MsgCreateSpotLimitOrderGas * uint64(count)
	case BatchCancelSpotOrdersMethodName:
		_, orders, err := castBatchCancelOrdersParams(method.Inputs, args)
		if err != nil {
			return cost
		}
		cost += exchangekeeper.MsgCancelSpotOrderGas * uint64(len(orders))
	case BatchUpdateOrdersMethodName:
		counts, err := countBatchUpdateOrdersParams(method.Inputs, args)
		if err != nil {
			return cost
		}
		cost += uint64(counts.DerivativeOrdersToCancel) * exchangekeeper.MsgCancelDerivativeOrderGas
		cost += uint64(counts.DerivativeOrdersToCreate) * exchangekeeper.MsgCreateDerivativeLimitOrderGas
		cost += uint64(counts.DerivativeMarketIdsToCancelAll) * 100_000
		cost += uint64(counts.SpotOrdersToCancel) * exchangekeeper.MsgCancelSpotOrderGas
		cost += uint64(counts.SpotOrdersToCreate) * exchangekeeper.MsgCreateSpotLimitOrderGas
		cost += uint64(counts.SpotMarketIdsToCancelAll) * 100_000
	}

	return cost
}

func (ec *ExchangeContract) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	res, err := ec.run(evm, contract, readonly)
	if err != nil {
		return types.RevertReasonAndError(err)
	}
	return res, nil
}

func (ec *ExchangeContract) run(evm *vm.EVM, contract *vm.Contract, readonly bool) (output []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrPrecompilePanic
			output = nil
		}
	}()

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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	params, err := castApproveParams(method.Inputs, args)
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

	grantee, msgTypes, err := castRevokeParams(args)
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
	args []any,
	_ bool,
) ([]byte, error) {
	params, err := castAllowanceParams(args)
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
	args []any,
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
	args []any,
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
	args []any,
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
	args []any,
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	msg, hold, err := ec.castIncreasePositionParams(args, evm)
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	msg, hold, err := ec.castDecreasePositionParams(args, evm)
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	_, msg, hold, err := ec.castBatchUpdateOrdersParams(method.Inputs, args, evm)
	if err != nil {
		return nil, err
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
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
	args []any,
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

// ATTENTION: unlike other methods, returned amounts are in human-readable format
func (ec *ExchangeContract) querySubaccountDeposits(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []any,
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
	args []any,
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

	solResults, err := ec.convertSubaccountPositionsResponse(resp, evm)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(solResults)
}

/*******************************************************************************
DERIVATIVE TRANSACTIONS
*******************************************************************************/

func (ec *ExchangeContract) createDerivativeLimitOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, hold, _, err := ec.castCreateDerivativeOrderParams(method.Inputs, args, evm)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateDerivativeLimitOrder{
		Sender: sender.String(),
		Order:  *order,
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, orders, hold, err := ec.castCreateDerivativeOrdersParams(method.Inputs, args, evm)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgBatchCreateDerivativeLimitOrders{
		Sender: sender.String(),
		Orders: orders,
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, hold, market, err := ec.castCreateDerivativeOrderParams(method.Inputs, args, evm)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateDerivativeMarketOrder{
		Sender: sender.String(),
		Order:  *order,
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

	solResp := convertCreateDerivativeMarketOrderResponse(resp, market)

	return method.Outputs.Pack(solResp)
}

func (ec *ExchangeContract) cancelDerivativeOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []any,
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

	resp := exchangetypesv2.MsgCancelDerivativeOrderResponse{}
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, data, err := castBatchCancelOrdersParams(method.Inputs, args)
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
	args []any,
	_ bool,
) ([]byte, error) {

	req, market, err := ec.castQueryDerivativeOrdersRequest(method.Inputs, args, evm)
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

	solOrders := convertTrimmedDerivativeOrders(resp.Orders, market)

	return method.Outputs.Pack(solOrders)
}

/*******************************************************************************
SPOT TRANSACTIONS
*******************************************************************************/

func (ec *ExchangeContract) createSpotLimitOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, hold, _, err := ec.castCreateSpotOrderParams(method.Inputs, args, evm)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateSpotLimitOrder{
		Sender: sender.String(),
		Order:  *order,
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, orders, hold, err := ec.castCreateSpotOrdersParams(method.Inputs, args, evm)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgBatchCreateSpotLimitOrders{
		Sender: sender.String(),
		Orders: orders,
	}

	resBytes, err := ec.validateAndDispatchMsg(evm, caller, msg, hold)
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, order, hold, market, err := ec.castCreateSpotOrderParams(method.Inputs, args, evm)
	if err != nil {
		return nil, err
	}

	msg := &exchangetypesv2.MsgCreateSpotMarketOrder{
		Sender: sender.String(),
		Order:  *order,
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

	solResp := ec.convertCreateSpotMarketOrderResponse(resp, market)

	return method.Outputs.Pack(solResp)
}

func (ec *ExchangeContract) cancelSpotOrder(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []any,
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
	args []any,
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	sender, data, err := castBatchCancelOrdersParams(method.Inputs, args)
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
SPOT QUERIES
*******************************************************************************/

func (ec *ExchangeContract) querySpotOrdersByHashes(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []any,
	_ bool,
) ([]byte, error) {

	req, market, err := ec.castQuerySpotOrdersRequest(method.Inputs, args, evm)
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

	solOrders := ec.convertTrimmedSpotOrders(resp.Orders, market)

	return method.Outputs.Pack(solOrders)
}

/******************************************************************************/

func (ec *ExchangeContract) getDerivativeMarket(
	marketID string,
	evm *vm.EVM,
) (*exchangetypesv2.DerivativeMarket, error) {
	var market *exchangetypesv2.DerivativeMarket
	err := ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			marketIDHash := common.HexToHash(marketID)
			market = ec.exchangeKeeper.GetDerivativeMarketByID(
				ctx,
				marketIDHash,
			)
			return nil
		},
	)
	if market == nil {
		return nil, exchangetypesv1.ErrDerivativeMarketNotFound.Wrapf("derivative market for marketID %s not found. err: %v", marketID, err)
	}
	return market, nil
}

func (ec *ExchangeContract) getSpotMarket(
	marketID string,
	evm *vm.EVM,
) (*exchangetypesv2.SpotMarket, error) {
	var market *exchangetypesv2.SpotMarket
	err := ec.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			marketIDHash := common.HexToHash(marketID)
			market = ec.exchangeKeeper.GetSpotMarketByID(
				ctx,
				marketIDHash,
			)
			return nil
		},
	)
	if market == nil {
		return nil, exchangetypesv1.ErrSpotMarketNotFound.Wrapf("spot market for marketID %s not found. err: %v", marketID, err)
	}
	return market, nil
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
