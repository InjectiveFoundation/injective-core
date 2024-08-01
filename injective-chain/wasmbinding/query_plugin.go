package wasmbinding

import (
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/golang/protobuf/proto" //nolint:staticcheck // this dependency is still used in cosmos sdk

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

const (
	AuthzRoute        = "authz"
	StakingRoute      = "staking"
	AuctionRoute      = "auction"
	OracleRoute       = "oracle"
	ExchangeRoute     = "exchange"
	TokenFactoryRoute = "tokenfactory"
	WasmxRoute        = "wasmx"
	FeeGrant          = "feegrant"
)

type InjectiveQueryWrapper struct {
	// specifies which module handler should handle the query
	Route string `json:"route,omitempty"`
	// The query data that should be parsed into the module query
	QueryData json.RawMessage `json:"query_data,omitempty"`
}

// CustomQuerier dispatches custom CosmWasm bindings queries.
func CustomQuerier(qp *QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var contractQuery InjectiveQueryWrapper
		if err := json.Unmarshal(request, &contractQuery); err != nil {
			return nil, errorsmod.Wrap(err, "Error parsing request data")
		}

		var bz []byte
		var err error

		switch contractQuery.Route {
		case AuthzRoute:
			bz, err = qp.HandleAuthzQuery(ctx, contractQuery.QueryData)
		case StakingRoute:
			bz, err = qp.HandleStakingQuery(ctx, contractQuery.QueryData)
		case AuctionRoute:
			bz, err = qp.HandleAuctionQuery(ctx, contractQuery.QueryData)
		case OracleRoute:
			bz, err = qp.HandleOracleQuery(ctx, contractQuery.QueryData)
		case ExchangeRoute:
			bz, err = qp.HandleExchangeQuery(ctx, contractQuery.QueryData)
		case TokenFactoryRoute:
			bz, err = qp.HandleTokenFactoryQuery(ctx, contractQuery.QueryData)
		case WasmxRoute:
			bz, err = qp.HandleWasmxQuery(ctx, contractQuery.QueryData)
		case FeeGrant:
			bz, err = qp.HandleFeeGrantQuery(ctx, contractQuery.QueryData)
		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "Unknown Injective Query Route"}
		}

		return bz, err
	}
}

type AcceptedStargateQueries map[string]proto.Message

func getWhitelistedQueries() wasmkeeper.AcceptedQueries {
	return wasmkeeper.AcceptedQueries{
		// auth
		"/cosmos.auth.v1beta1.Query/Account": &authtypes.QueryAccountResponse{},
		"/cosmos.auth.v1beta1.Query/Params":  &authtypes.QueryParamsResponse{},

		// bank
		"/cosmos.bank.v1beta1.Query/Balance":       &banktypes.QueryBalanceResponse{},
		"/cosmos.bank.v1beta1.Query/DenomMetadata": &banktypes.QueryDenomsMetadataResponse{},
		"/cosmos.bank.v1beta1.Query/Params":        &banktypes.QueryParamsResponse{},
		"/cosmos.bank.v1beta1.Query/SupplyOf":      &banktypes.QuerySupplyOfResponse{},

		// Injective queries
		// Exchange
		"/injective.exchange.v1beta1.Query/QueryExchangeParams":                 &exchangetypes.QueryExchangeParamsResponse{},
		"/injective.exchange.v1beta1.Query/SubaccountDeposit":                   &exchangetypes.QuerySubaccountDepositResponse{},
		"/injective.exchange.v1beta1.Query/DerivativeMarket":                    &exchangetypes.QueryDerivativeMarketResponse{},
		"/injective.exchange.v1beta1.Query/SpotMarket":                          &exchangetypes.QuerySpotMarketResponse{},
		"/injective.exchange.v1beta1.Query/SubaccountEffectivePositionInMarket": &exchangetypes.QuerySubaccountEffectivePositionInMarketResponse{},
		"/injective.exchange.v1beta1.Query/SubaccountPositionInMarket":          &exchangetypes.QuerySubaccountPositionInMarketResponse{},
		"/injective.exchange.v1beta1.Query/TraderDerivativeOrders":              &exchangetypes.QueryTraderDerivativeOrdersResponse{},
		"/injective.exchange.v1beta1.Query/TraderDerivativeTransientOrders":     &exchangetypes.QueryTraderDerivativeOrdersResponse{},
		"/injective.exchange.v1beta1.Query/TraderSpotTransientOrders":           &exchangetypes.QueryTraderSpotOrdersResponse{},
		"/injective.exchange.v1beta1.Query/TraderSpotOrders":                    &exchangetypes.QueryTraderSpotOrdersResponse{},
		"/injective.exchange.v1beta1.Query/PerpetualMarketInfo":                 &exchangetypes.QueryPerpetualMarketInfoResponse{},
		"/injective.exchange.v1beta1.Query/PerpetualMarketFunding":              &exchangetypes.QueryPerpetualMarketFundingResponse{},
		"/injective.exchange.v1beta1.Query/MarketVolatility":                    &exchangetypes.QueryMarketVolatilityResponse{},
		"/injective.exchange.v1beta1.Query/SpotMidPriceAndTOB":                  &exchangetypes.QuerySpotMidPriceAndTOBResponse{},
		"/injective.exchange.v1beta1.Query/DerivativeMidPriceAndTOB":            &exchangetypes.QueryDerivativeMidPriceAndTOBResponse{},
		"/injective.exchange.v1beta1.Query/AggregateMarketVolume":               &exchangetypes.QueryAggregateMarketVolumeResponse{},
		"/injective.exchange.v1beta1.Query/SpotOrderbook":                       &exchangetypes.QuerySpotOrderbookResponse{},
		"/injective.exchange.v1beta1.Query/DerivativeOrderbook":                 &exchangetypes.QueryDerivativeOrderbookResponse{},
		"/injective.exchange.v1beta1.Query/MarketAtomicExecutionFeeMultiplier":  &exchangetypes.QueryMarketAtomicExecutionFeeMultiplierResponse{},
		// Oracle
		"/injective.oracle.v1beta1.Query/OracleVolatility": &oracletypes.QueryOracleVolatilityResponse{},
		"/injective.oracle.v1beta1.Query/OraclePrice":      &oracletypes.QueryOraclePriceResponse{},
		"/injective.oracle.v1beta1.Query/PythPrice":        &oracletypes.QueryPythPriceResponse{},
		// Auction
		"/injective.auction.v1beta1.Query/LastAuctionResult":    &auctiontypes.QueryLastAuctionResultResponse{},
		"/injective.auction.v1beta1.Query/AuctionParams":        &auctiontypes.QueryAuctionParamsResponse{},
		"/injective.auction.v1beta1.Query/CurrentAuctionBasket": &auctiontypes.QueryCurrentAuctionBasketResponse{},
		// Authz
		"/cosmos.authz.v1beta1.Query/GranteeGrants": &authz.QueryGranteeGrantsResponse{},
		"/cosmos.authz.v1beta1.Query/GranterGrants": &authz.QueryGranterGrantsResponse{},
		"/cosmos.authz.v1beta1.Query/Grants":        &authz.QueryGrantsResponse{},
	}
}

// StargateQuerier dispatches whitelisted stargate queries
func StargateQuerier(queryRouter baseapp.GRPCQueryRouter, codecInterface codec.Codec) func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	acceptList := getWhitelistedQueries()
	return func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		protoResponse, accepted := acceptList[request.Path]
		if !accepted {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", request.Path)}
		}

		route := queryRouter.Route(request.Path)
		if route == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := route(ctx, &abci.RequestQuery{
			Data: request.Data,
			Path: request.Path,
		})
		if err != nil {
			return nil, err
		}

		return ConvertProtoToJSONMarshal(codecInterface, protoResponse, res.Value)
	}
}

// ConvertProtoToJsonMarshal  unmarshals the given bytes into a proto message and then marshals it to json.
// This is done so that clients calling stargate queries do not need to define their own proto unmarshalers,
// being able to use response directly by json marshalling, which is supported in cosmwasm.
func ConvertProtoToJSONMarshal(cdc codec.Codec, protoResponse proto.Message, bz []byte) ([]byte, error) {
	// unmarshal binary into stargate response data structure
	err := cdc.Unmarshal(bz, protoResponse)
	if err != nil {
		return nil, errorsmod.Wrap(err, "to proto")
	}

	bz, err = cdc.MarshalJSON(protoResponse)
	if err != nil {
		return nil, errorsmod.Wrap(err, "to json")
	}

	protoResponse.Reset()
	return bz, nil
}
