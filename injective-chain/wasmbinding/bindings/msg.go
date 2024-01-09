package bindings

import (
	"encoding/json"

	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

type InjectiveMsgWrapper struct {
	// specifies which module handler should handle the query
	Route string `json:"route,omitempty"`
	// The msg data that should be parsed into the module query
	MsgData json.RawMessage `json:"msg_data,omitempty"`
}

type InjectiveMsg struct {
	AuctionMsg
	ExchangeMsg
	FeeGrantMsg
	InsuranceMsg
	OcrMsg
	OracleMsg
	PeggyMsg
	TokenFactoryMsg
	WasmxMsg
}

type AuctionMsg struct {
}

type ExchangeMsg struct {
	Deposit                          *exchangetypes.MsgDeposit                          `json:"deposit,omitempty"`
	Withdraw                         *exchangetypes.MsgWithdraw                         `json:"withdraw,omitempty"`
	CreateSpotLimitOrder             *exchangetypes.MsgCreateSpotLimitOrder             `json:"create_spot_limit_order,omitempty"`
	BatchCreateSpotLimitOrders       *exchangetypes.MsgBatchCreateSpotLimitOrders       `json:"batch_create_spot_limit_orders,omitempty"`
	CreateSpotMarketOrder            *exchangetypes.MsgCreateSpotMarketOrder            `json:"create_spot_market_order,omitempty"`
	CancelSpotOrder                  *exchangetypes.MsgCancelSpotOrder                  `json:"cancel_spot_order,omitempty"`
	BatchCancelSpotOrders            *exchangetypes.MsgBatchCancelSpotOrders            `json:"batch_cancel_spot_orders,omitempty"`
	CreateDerivativeLimitOrder       *exchangetypes.MsgCreateDerivativeLimitOrder       `json:"create_derivative_limit_order,omitempty"`
	BatchCreateDerivativeLimitOrders *exchangetypes.MsgBatchCreateDerivativeLimitOrders `json:"batch_create_derivative_limit_orders,omitempty"`
	CreateDerivativeMarketOrder      *exchangetypes.MsgCreateDerivativeMarketOrder      `json:"create_derivative_market_order,omitempty"`
	CancelDerivativeOrder            *exchangetypes.MsgCancelDerivativeOrder            `json:"cancel_derivative_order,omitempty"`
	BatchCancelDerivativeOrders      *exchangetypes.MsgBatchCancelDerivativeOrders      `json:"batch_cancel_derivative_orders,omitempty"`
	SubaccountTransfer               *exchangetypes.MsgSubaccountTransfer               `json:"subaccount_transfer,omitempty"`
	ExternalTransfer                 *exchangetypes.MsgExternalTransfer                 `json:"external_transfer,omitempty"`
	IncreasePositionMargin           *exchangetypes.MsgIncreasePositionMargin           `json:"increase_position_margin,omitempty"`
	LiquidatePosition                *exchangetypes.MsgLiquidatePosition                `json:"liquidate_position,omitempty"`
	InstantSpotMarketLaunch          *exchangetypes.MsgInstantSpotMarketLaunch          `json:"instant_spot_market_launch,omitempty"`
	InstantPerpetualMarketLaunch     *exchangetypes.MsgInstantPerpetualMarketLaunch     `json:"instant_perpetual_market_launch,omitempty"`
	InstantExpiryFuturesMarketLaunch *exchangetypes.MsgInstantExpiryFuturesMarketLaunch `json:"instant_expiry_futures_market_launch,omitempty"`
	BatchUpdateOrders                *exchangetypes.MsgBatchUpdateOrders                `json:"batch_update_orders,omitempty"`
	PrivilegedExecuteContract        *exchangetypes.MsgPrivilegedExecuteContract        `json:"privileged_execute_contract,omitempty"`
	RewardsOptOut                    *exchangetypes.MsgRewardsOptOut                    `json:"rewards_opt_out,omitempty"`
}

type FeeGrantMsg struct {
	GrantAllowance  *feegrant.MsgGrantAllowance  `json:"grant_allowance,omitempty"`
	RevokeAllowance *feegrant.MsgRevokeAllowance `json:"revoke_allowance,omitempty"`
}

type InsuranceMsg struct{}

type OcrMsg struct{}

type OracleMsg struct {
	RelayPythPrices *oracletypes.MsgRelayPythPrices `json:"relay_pyth_prices,omitempty"`
}

type PeggyMsg struct{}

type TokenFactoryMsg struct {
	/// Contracts can create denoms, namespaced under the contract's address.
	/// A contract may create any number of independent sub-denoms.
	CreateDenom *tokenfactorytypes.MsgCreateDenom `json:"create_denom,omitempty"`
	/// Contracts can change the admin of a denom that they are the admin of.
	ChangeAdmin *tokenfactorytypes.MsgChangeAdmin `json:"change_admin,omitempty"`
	/// Contracts can mint native tokens for an existing factory denom
	/// that they are the admin of.
	MintTokens *MintTokens `json:"mint,omitempty"`
	/// Contracts can burn native tokens for an existing factory denom
	/// that they are the admin of.
	/// Currently, the burn from address must be the admin contract.
	BurnTokens *tokenfactorytypes.MsgBurn `json:"burn,omitempty"`
	/// Sets metadata for TF denom
	SetTokenMetadata *TokenMetadata `json:"set_token_metadata,omitempty"`
}

type WasmxMsg struct {
	// update contract params (like gas price or gas limit)
	UpdateContractMsg *wasmxtypes.MsgUpdateContract `json:"update_contract,omitempty"`
	// Deactivate (pause) contract - won't be executed in begin blocker any longer
	DeactivateContractMsg *wasmxtypes.MsgDeactivateContract `json:"deactivate_contract,omitempty"`
	// Reactivate paused contract - will be again executed
	ActivateContractMsg *wasmxtypes.MsgActivateContract `json:"activate_contract,omitempty"`
}

type MintTokens struct {
	Amount sdk.Coin `json:"amount"`
	MintTo string   `json:"mint_to"`
}

type TokenMetadata struct {
	Denom    string `json:"denom"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint32 `json:"decimals"`
}
