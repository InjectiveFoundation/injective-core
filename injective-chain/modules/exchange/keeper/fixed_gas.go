package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
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
	case *exchangetypes.MsgCreateSpotLimitOrder:
		if msg.Order.OrderType.IsPostOnly() {
			return MsgCreateSpotLimitPostOnlyOrderGas
		}
		return MsgCreateSpotLimitOrderGas
	case *exchangetypes.MsgCreateSpotMarketOrder:
		return MsgCreateSpotMarketOrderGas
	case *exchangetypes.MsgCancelSpotOrder:
		return MsgCancelSpotOrderGas
	case *exchangetypes.MsgBatchCreateSpotLimitOrders:
		sum := uint64(0)
		for _, order := range msg.Orders {
			if order.OrderType.IsPostOnly() {
				sum += MsgCreateSpotLimitPostOnlyOrderGas
			} else {
				sum += MsgCreateSpotLimitOrderGas
			}
		}

		return sum
	case *exchangetypes.MsgBatchCancelSpotOrders:
		panic("developer error: MsgBatchCancelSpotOrders gas already determined in msg server impl")
	case *exchangetypes.MsgCreateDerivativeLimitOrder:
		if msg.Order.OrderType.IsPostOnly() {
			return MsgCreateDerivativeLimitPostOnlyOrderGas
		}

		return MsgCreateDerivativeLimitOrderGas
	case *exchangetypes.MsgCreateDerivativeMarketOrder:
		return MsgCreateDerivativeMarketOrderGas
	case *exchangetypes.MsgCancelDerivativeOrder:
		return MsgCancelDerivativeOrderGas
	case *exchangetypes.MsgBatchCreateDerivativeLimitOrders:
		sum := uint64(0)
		for _, order := range msg.Orders {
			if order.OrderType.IsPostOnly() {
				sum += MsgCreateDerivativeLimitPostOnlyOrderGas
			} else {
				sum += MsgCreateDerivativeLimitOrderGas
			}
		}

		return sum
	case *exchangetypes.MsgBatchCancelDerivativeOrders:
		panic("developer error: MsgBatchCancelDerivativeOrders gas already determined in msg server impl")
	case *exchangetypes.MsgCreateBinaryOptionsLimitOrder:
		if msg.Order.OrderType.IsPostOnly() {
			return MsgCreateBinaryOptionsLimitPostOnlyOrderGas
		}
		return MsgCreateBinaryOptionsLimitOrderGas
	case *exchangetypes.MsgCreateBinaryOptionsMarketOrder:
		return MsgCreateBinaryOptionsMarketOrderGas
	case *exchangetypes.MsgCancelBinaryOptionsOrder:
		return MsgCancelBinaryOptionsOrderGas
	case *exchangetypes.MsgBatchCancelBinaryOptionsOrders:
		panic("developer error: MsgBatchCancelBinaryOptionsOrders gas already determined in msg server impl")
	//	MISCELLANEOUS //
	case *exchangetypes.MsgDeposit:
		return MsgDepositGas
	case *exchangetypes.MsgWithdraw:
		return MsgWithdrawGas
	case *exchangetypes.MsgSubaccountTransfer:
		return MsgSubaccountTransferGas
	case *exchangetypes.MsgExternalTransfer:
		return MsgExternalTransferGas
	case *exchangetypes.MsgIncreasePositionMargin:
		return MsgIncreasePositionMarginGas
	case *exchangetypes.MsgDecreasePositionMargin:
		return MsgDecreasePositionMarginGas
	default:
		panic(fmt.Sprintf("developer error: unknown message type: %T", msg))
	}
}
