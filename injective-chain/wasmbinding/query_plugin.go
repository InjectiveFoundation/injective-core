package wasmbinding

import (
	"encoding/json"

	"cosmossdk.io/errors"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AuthzRoute        = "authz"
	StakingRoute      = "staking"
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
			return nil, errors.Wrap(err, "Error parsing request data")
		}

		var bz []byte
		var err error

		switch contractQuery.Route {
		case AuthzRoute:
			bz, err = qp.HandleAuthzQuery(ctx, contractQuery.QueryData)
		case StakingRoute:
			bz, err = qp.HandleStakingQuery(ctx, contractQuery.QueryData)
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
