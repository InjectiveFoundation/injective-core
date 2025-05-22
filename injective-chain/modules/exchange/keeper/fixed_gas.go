package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	exchangev2types "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

var (
	MsgCreateDerivativeLimitOrderGas         = storetypes.Gas(120_000)
	MsgCreateDerivativeLimitPostOnlyOrderGas = storetypes.Gas(140_000)
	MsgCreateDerivativeMarketOrderGas        = storetypes.Gas(105_000)
	MsgCancelDerivativeOrderGas              = storetypes.Gas(70_000)

	MsgCreateSpotLimitOrderGas         = storetypes.Gas(100_000)
	MsgCreateSpotLimitPostOnlyOrderGas = storetypes.Gas(120_000)
	MsgCreateSpotMarketOrderGas        = storetypes.Gas(50_000)
	MsgCancelSpotOrderGas              = storetypes.Gas(65_000)

	GTBOrdersGasMultiplier = math.LegacyMustNewDecFromStr("1.1")

	// NOTE: binary option orders are handled identically as derivative orders
	MsgCreateBinaryOptionsLimitOrderGas         = MsgCreateDerivativeLimitOrderGas
	MsgCreateBinaryOptionsLimitPostOnlyOrderGas = MsgCreateDerivativeLimitPostOnlyOrderGas
	MsgCreateBinaryOptionsMarketOrderGas        = MsgCreateDerivativeMarketOrderGas
	MsgCancelBinaryOptionsOrderGas              = MsgCancelDerivativeOrderGas

	MsgDepositGas                = storetypes.Gas(38_000)
	MsgWithdrawGas               = storetypes.Gas(35_000)
	MsgSubaccountTransferGas     = storetypes.Gas(15_000)
	MsgExternalTransferGas       = storetypes.Gas(40_000)
	MsgIncreasePositionMarginGas = storetypes.Gas(51_000)
	MsgDecreasePositionMarginGas = storetypes.Gas(60_000)
)

//nolint:revive //this is fine
func DetermineGas(msg sdk.Msg) uint64 {
	switch msg := msg.(type) {
	case *exchangev2types.MsgCreateSpotLimitOrder:
		requiredGas := MsgCreateSpotLimitOrderGas
		if msg.Order.OrderType.IsPostOnly() {
			requiredGas = MsgCreateSpotLimitPostOnlyOrderGas
		}
		if msg.Order.ExpirationBlock > 0 {
			requiredGas = storetypes.Gas(GTBOrdersGasMultiplier.Mul(math.LegacyNewDec(int64(requiredGas))).TruncateInt64())
		}
		return requiredGas
	case *exchangev2types.MsgCreateSpotMarketOrder:
		return MsgCreateSpotMarketOrderGas
	case *exchangev2types.MsgCancelSpotOrder:
		return MsgCancelSpotOrderGas
	case *exchangev2types.MsgBatchCreateSpotLimitOrders:
		sum := uint64(0)
		for _, order := range msg.Orders {
			requiredGas := MsgCreateSpotLimitOrderGas
			if order.OrderType.IsPostOnly() {
				requiredGas = MsgCreateSpotLimitPostOnlyOrderGas
			}
			if order.ExpirationBlock > 0 {
				requiredGas = storetypes.Gas(GTBOrdersGasMultiplier.Mul(math.LegacyNewDec(int64(requiredGas))).TruncateInt64())
			}
			sum += requiredGas
		}

		return sum
	case *exchangev2types.MsgBatchCancelSpotOrders:
		panic("developer error: MsgBatchCancelSpotOrders gas already determined in msg server impl")
	case *exchangev2types.MsgCreateDerivativeLimitOrder:
		requiredGas := MsgCreateDerivativeLimitOrderGas
		if msg.Order.OrderType.IsPostOnly() {
			requiredGas = MsgCreateDerivativeLimitPostOnlyOrderGas
		}
		if msg.Order.ExpirationBlock > 0 {
			requiredGas = storetypes.Gas(GTBOrdersGasMultiplier.Mul(math.LegacyNewDec(int64(requiredGas))).TruncateInt64())
		}
		return requiredGas
	case *exchangev2types.MsgCreateDerivativeMarketOrder:
		return MsgCreateDerivativeMarketOrderGas
	case *exchangev2types.MsgCancelDerivativeOrder:
		return MsgCancelDerivativeOrderGas
	case *exchangev2types.MsgBatchCreateDerivativeLimitOrders:
		sum := uint64(0)
		for _, order := range msg.Orders {
			requiredGas := MsgCreateDerivativeLimitOrderGas
			if order.OrderType.IsPostOnly() {
				requiredGas = MsgCreateDerivativeLimitPostOnlyOrderGas
			}
			if order.ExpirationBlock > 0 {
				requiredGas = storetypes.Gas(GTBOrdersGasMultiplier.Mul(math.LegacyNewDec(int64(requiredGas))).TruncateInt64())
			}
			sum += requiredGas
		}

		return sum
	case *exchangev2types.MsgBatchCancelDerivativeOrders:
		panic("developer error: MsgBatchCancelDerivativeOrders gas already determined in msg server impl")
	case *exchangev2types.MsgCreateBinaryOptionsLimitOrder:
		requiredGas := MsgCreateBinaryOptionsLimitOrderGas
		if msg.Order.OrderType.IsPostOnly() {
			requiredGas = MsgCreateBinaryOptionsLimitPostOnlyOrderGas
		}
		if msg.Order.ExpirationBlock > 0 {
			requiredGas = storetypes.Gas(GTBOrdersGasMultiplier.Mul(math.LegacyNewDec(int64(requiredGas))).TruncateInt64())
		}
		return requiredGas
	case *exchangev2types.MsgCreateBinaryOptionsMarketOrder:
		return MsgCreateBinaryOptionsMarketOrderGas
	case *exchangev2types.MsgCancelBinaryOptionsOrder:
		return MsgCancelBinaryOptionsOrderGas
	case *exchangev2types.MsgBatchCancelBinaryOptionsOrders:
		panic("developer error: MsgBatchCancelBinaryOptionsOrders gas already determined in msg server impl")
	//	MISCELLANEOUS //
	case *exchangev2types.MsgDeposit:
		return MsgDepositGas
	case *exchangev2types.MsgWithdraw:
		return MsgWithdrawGas
	case *exchangev2types.MsgSubaccountTransfer:
		return MsgSubaccountTransferGas
	case *exchangev2types.MsgExternalTransfer:
		return MsgExternalTransferGas
	case *exchangev2types.MsgIncreasePositionMargin:
		return MsgIncreasePositionMarginGas
	case *exchangev2types.MsgDecreasePositionMargin:
		return MsgDecreasePositionMarginGas
	default:
		panic(fmt.Sprintf("developer error: unknown message type: %T", msg))
	}
}
