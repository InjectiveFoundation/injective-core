package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

/*
Values below show gas usage when sending a tx with 1 exchange msg
- consumed: the amount of gas consumed by the MsgServer
- total: the amount of gas consumed by the entire tx (same as `gasUsed`)
*/
var (
	//	consumed=81530 total=182038
	//	consumed=81722 total=167522
	//	consumed=81815 total=167615
	//	consumed=81851 total=167651
	//	consumed=81734 total=167554
	MsgCreateDerivativeLimitOrderGas = storetypes.Gas(82_000)

	//	consumed=111614 total=197394
	//	consumed=111746 total=197526
	//	consumed=111749 total=197529
	//	consumed=111782 total=197562
	//	consumed=111815 total=197605
	MsgCreateDerivativeLimitPostOnlyOrderGas = storetypes.Gas(112_000)

	//	consumed=66172 total=152022
	//	consumed=66169 total=152009
	//	consumed=66169 total=152009
	//	consumed=66163 total=152003
	//	consumed=66163 total=152003
	MsgCreateDerivativeMarketOrderGas = storetypes.Gas(67_000)

	//	consumed=72782 total=157222
	//	consumed=72752 total=157192
	//	consumed=72623 total=157063
	//	consumed=60812 total=145252
	//	consumed=60845 total=145285
	MsgCancelDerivativeOrderGas = storetypes.Gas(73_000)

	//	consumed=60882 total=146322 (constant across iterations)
	MsgCreateSpotLimitOrderGas = storetypes.Gas(61_000)

	//	consumed=91331 total=176771
	//	consumed=91238 total=176678
	//	consumed=91238 total=176678
	//	consumed=91235 total=176675
	//	consumed=91067 total=176507
	MsgCreateSpotLimitPostOnlyOrderGas = storetypes.Gas(92_000)

	//	consumed=49651 total=135951 (constant across iterations)
	MsgCreateSpotMarketOrderGas = storetypes.Gas(50_000)

	//	consumed=50590 total=134970
	//	consumed=50590 total=134970
	//	consumed=50590 total=134970
	//	consumed=50590 total=134970
	//	consumed=50590 total=134970
	MsgCancelSpotOrderGas = storetypes.Gas(51_000)

	// NOTE: binary option orders are handled identically as derivative orders

	MsgCreateBinaryOptionsLimitOrderGas         = MsgCreateDerivativeLimitOrderGas
	MsgCreateBinaryOptionsLimitPostOnlyOrderGas = MsgCreateDerivativeLimitPostOnlyOrderGas
	MsgCreateBinaryOptionsMarketOrderGas        = MsgCreateDerivativeMarketOrderGas
	MsgCancelBinaryOptionsOrderGas              = MsgCancelDerivativeOrderGas

	//	consumed=37265 total=135493
	//	consumed=37556 total=121076
	//	consumed=37556 total=121076
	//	consumed=37556 total=121076
	//	consumed=37556 total=121076
	MsgDepositGas = storetypes.Gas(38_000)

	//	consumed=34842 total=118372
	//	consumed=34842 total=118372
	//	consumed=34842 total=118372
	//	consumed=34842 total=118372
	//	consumed=32466 total=115996
	MsgWithdrawGas = storetypes.Gas(35_000)

	//	consumed=14392 total=98702
	//	consumed=14608 total=98918
	//	consumed=14608 total=98918
	//	consumed=14608 total=98918
	//	consumed=12232 total=96542
	MsgSubaccountTransferGas = storetypes.Gas(15_000)

	//	consumed=39449 total=123739
	//	consumed=39461 total=123751
	//	consumed=39461 total=123751
	//	consumed=39461 total=123751
	//	consumed=37085 total=121375
	MsgExternalTransferGas = storetypes.Gas(40_000)

	//	consumed=50939 total=135969 (constant across iterations)
	MsgIncreasePositionMarginGas = storetypes.Gas(51_000)

	//	consumed=59892 total=144922 (constant across iterations)
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
