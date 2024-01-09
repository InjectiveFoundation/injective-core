package bindings

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/feegrant"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// InjectiveQuery contains custom injective queries.
type InjectiveQuery struct {
	AuctionQuery
	AuthzQuery
	ExchangeQuery
	InsuranceQuery
	FeeGrantQuery
	OcrQuery
	StakingQuery
	OracleQuery
	PeggyQuery
	TokenfactoryQuery
	WasmxQuery
}

type AuctionQuery struct{}

type AuthzQuery struct {
	Grants        *authz.QueryGrantsRequest        `json:"grants,omitempty"`
	GranterGrants *authz.QueryGranterGrantsRequest `json:"granter_grants,omitempty"`
	GranteeGrants *authz.QueryGranteeGrantsRequest `json:"grantee_grants,omitempty"`
}

type ExchangeQuery struct {
	ExchangeParams                                  *exchangetypes.QueryExchangeParamsRequest                           `json:"exchange_params,omitempty"`
	SubaccountDeposit                               *exchangetypes.QuerySubaccountDepositRequest                        `json:"subaccount_deposit,omitempty"`
	SpotMarket                                      *exchangetypes.QuerySpotMarketRequest                               `json:"spot_market,omitempty"`
	DerivativeMarket                                *exchangetypes.QueryDerivativeMarketRequest                         `json:"derivative_market,omitempty"`
	SubaccountEffectivePositionInMarket             *exchangetypes.QuerySubaccountEffectivePositionInMarketRequest      `json:"subaccount_effective_position_in_market,omitempty"`
	SubaccountPositionInMarket                      *exchangetypes.QuerySubaccountPositionInMarketRequest               `json:"subaccount_position_in_market,omitempty"`
	SubaccountPositions                             *exchangetypes.QuerySubaccountPositionsRequest                      `json:"subaccount_positions,omitempty"`
	SubaccountOrders                                *exchangetypes.QuerySubaccountOrdersRequest                         `json:"subaccount_orders,omitempty"`
	TraderSpotOrders                                *exchangetypes.QueryTraderSpotOrdersRequest                         `json:"trader_spot_orders,omitempty"`
	TraderSpotOrdersToCancelUpToAmountRequest       *exchangetypes.QueryTraderSpotOrdersToCancelUpToAmountRequest       `json:"trader_spot_orders_to_cancel_up_to_amount,omitempty"`
	TraderDerivativeOrdersToCancelUpToAmountRequest *exchangetypes.QueryTraderDerivativeOrdersToCancelUpToAmountRequest `json:"trader_derivative_orders_to_cancel_up_to_amount,omitempty"`
	TraderTransientSpotOrders                       *exchangetypes.QueryTraderSpotOrdersRequest                         `json:"trader_transient_spot_orders,omitempty"`
	SpotOrderbook                                   *exchangetypes.QuerySpotOrderbookRequest                            `json:"spot_orderbook,omitempty"`
	DerivativeOrderbook                             *exchangetypes.QueryDerivativeOrderbookRequest                      `json:"derivative_orderbook,omitempty"`
	TraderDerivativeOrders                          *exchangetypes.QueryTraderDerivativeOrdersRequest                   `json:"trader_derivative_orders,omitempty"`
	TraderTransientDerivativeOrders                 *exchangetypes.QueryTraderDerivativeOrdersRequest                   `json:"trader_transient_derivative_orders,omitempty"`
	PerpetualMarketInfo                             *exchangetypes.QueryPerpetualMarketInfoRequest                      `json:"perpetual_market_info,omitempty"`
	ExpiryFuturesMarketInfo                         *exchangetypes.QueryExpiryFuturesMarketInfoRequest                  `json:"expiry_futures_market_info,omitempty"`
	PerpetualMarketFunding                          *exchangetypes.QueryPerpetualMarketFundingRequest                   `json:"perpetual_market_funding,omitempty"`
	SpotMarketMidPriceAndTOB                        *exchangetypes.QuerySpotMidPriceAndTOBRequest                       `json:"spot_market_mid_price_and_tob,omitempty"`
	DerivativeMarketMidPriceAndTOB                  *exchangetypes.QueryDerivativeMidPriceAndTOBRequest                 `json:"derivative_market_mid_price_and_tob,omitempty"`
	MarketVolatility                                *exchangetypes.QueryMarketVolatilityRequest                         `json:"market_volatility,omitempty"`
	MarketAtomicExecutionFeeMultiplier              *exchangetypes.QueryMarketAtomicExecutionFeeMultiplierRequest       `json:"market_atomic_execution_fee_multiplier"`
	AggregateMarketVolume                           *exchangetypes.QueryAggregateMarketVolumeRequest                    `json:"aggregate_market_volume"`
	AggregateAccountVolume                          *exchangetypes.QueryAggregateVolumeRequest                          `json:"aggregate_account_volume"`
	DenomDecimal                                    *exchangetypes.QueryDenomDecimalRequest                             `json:"denom_decimal"`
	DenomDecimals                                   *exchangetypes.QueryDenomDecimalsRequest                            `json:"denom_decimals"`
}

type InsuranceQuery struct{}

type OcrQuery struct{}

type StakingQuery struct {
	StakedAmount *StakingDelegationAmount `json:"staked_amount,omitempty"`
}

type OracleQuery struct {
	OracleParams     *oracletypes.QueryParamsRequest           `json:"oracle_params,omitempty"`
	OracleVolatility *oracletypes.QueryOracleVolatilityRequest `json:"oracle_volatility,omitempty"`
	OraclePrice      *oracletypes.QueryOraclePriceRequest      `json:"oracle_price,omitempty"`
	PythPrice        *oracletypes.QueryPythPriceRequest        `json:"pyth_price,omitempty"`
}

type PeggyQuery struct{}

type TokenfactoryQuery struct {
	/// Returns the admin of a denom, if the denom is a Token Factory denom.
	DenomAdmin *DenomAdmin `json:"denom_admin,omitempty"`
	/// Returns a total supply of a denom, if the denom is a Token Factory denom.
	DenomTotalSupply *TotalSupply `json:"token_factory_denom_total_supply"`
	/// Returns a fee required to create a new denom
	DenomCreationFee *DenomCreationFee `json:"token_factory_denom_creation_fee"`
}

type WasmxQuery struct {
	RegisteredContractInfo *RegisteredContractInfo `json:"wasmx_registered_contract_info"`
}

type FeeGrantQuery struct {
	Allowance           *feegrant.QueryAllowanceRequest           `json:"allowance"`
	Allowances          *feegrant.QueryAllowancesRequest          `json:"allowances"`
	AllowancesByGranter *feegrant.QueryAllowancesByGranterRequest `json:"allowances_by_granter"`
}

type SubaccountDepositQueryResponse struct {
	Deposits *exchangetypes.Deposit `json:"deposits"`
}

type DerivativeMarketQueryResponse struct {
	Market *FullDerivativeMarketQuery `protobuf:"bytes,1,opt,name=market,proto3" json:"market,omitempty"`
}

type FullDerivativeMarketQuery struct {
	Market    *exchangetypes.DerivativeMarket                   `protobuf:"bytes,1,opt,name=market,proto3" json:"market,omitempty"`
	Info      *exchangetypes.FullDerivativeMarket_PerpetualInfo `json:"info"`
	MarkPrice sdk.Dec                                           `protobuf:"bytes,4,opt,name=mark_price,json=markPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"mark_price"`
}

type DenomAdmin struct {
	Subdenom string `json:"subdenom"`
}

type TotalSupply struct {
	Denom string `json:"denom"`
}

type DenomCreationFee struct{}

type DenomAdminResponse struct {
	Admin string `json:"admin"`
}

type DenomTotalSupplyResponse struct {
	TotalSupply sdkmath.Int `json:"total_supply"`
}

type DenomCreationFeeResponse struct {
	Fee sdk.Coins `json:"fee"`
}

type RegisteredContractInfo struct {
	ContractAddress string `json:"contract_address"`
}

type StakingDelegationAmount struct {
	DelegatorAddress string `json:"delegator_address"`
	MaxDelegations   uint16 `json:"max_delegations"`
}

type StakingDelegationAmountResponse struct {
	StakedAmount sdkmath.Int `json:"staked_amount"`
}
