package wasmbinding

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/common"

	auctionkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/keeper"
	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	oraclekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	wasmxkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/wasmbinding/bindings"
)

type QueryPlugin struct {
	authzKeeper        *authzkeeper.Keeper
	bankKeeper         *bankkeeper.BaseKeeper
	auctionKeeper      *auctionkeeper.Keeper
	exchangeKeeper     *exchangekeeper.Keeper
	feegrantKeeper     *feegrantkeeper.Keeper
	oracleKeeper       *oraclekeeper.Keeper
	tokenFactoryKeeper *tokenfactorykeeper.Keeper
	wasmxKeeper        *wasmxkeeper.Keeper
}

// NewQueryPlugin returns a reference to a new QueryPlugin.
func NewQueryPlugin(
	ak *authzkeeper.Keeper,
	auck *auctionkeeper.Keeper,
	ek *exchangekeeper.Keeper,
	ok *oraclekeeper.Keeper,
	bk *bankkeeper.BaseKeeper,
	tfk *tokenfactorykeeper.Keeper,
	wk *wasmxkeeper.Keeper,
	fgk *feegrantkeeper.Keeper,
) *QueryPlugin {
	return &QueryPlugin{
		authzKeeper:        ak,
		bankKeeper:         bk,
		auctionKeeper:      auck,
		exchangeKeeper:     ek,
		feegrantKeeper:     fgk,
		oracleKeeper:       ok,
		tokenFactoryKeeper: tfk,
		wasmxKeeper:        wk,
	}
}

func ForceMarshalJSONAny(msgAny *cdctypes.Any) {
	compatMsg := map[string]interface{}{
		"type_url": msgAny.TypeUrl,
		"value":    msgAny.Value,
	}
	bz, err := json.Marshal(compatMsg)
	if err != nil {
		panic(err)
	}

	if err = msgAny.UnmarshalJSON(bz); err != nil {
		panic(err)
	}
}

func (qp QueryPlugin) HandleAuthzQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.AuthzQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective AuthzQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.Grants != nil:
		req := query.Grants
		var grant *authz.QueryGrantsResponse

		grant, err = qp.authzKeeper.Grants(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "Error querying grants")
		}

		for _, g := range grant.Grants {
			if g.Authorization != nil {
				ForceMarshalJSONAny(g.Authorization)
			}
		}

		bz, err = json.Marshal(&authz.QueryGrantsResponse{
			Grants:     grant.Grants,
			Pagination: grant.Pagination,
		})
	case query.GranterGrants != nil:
		req := query.GranterGrants
		var grant *authz.QueryGranterGrantsResponse

		grant, err = qp.authzKeeper.GranterGrants(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "Error querying granter grants")
		}

		for _, g := range grant.Grants {
			if g.Authorization != nil {
				ForceMarshalJSONAny(g.Authorization)
			}
		}

		bz, err = json.Marshal(&authz.QueryGranteeGrantsResponse{
			Grants:     grant.Grants,
			Pagination: grant.Pagination,
		})
	case query.GranteeGrants != nil:
		req := query.GranteeGrants
		var grant *authz.QueryGranteeGrantsResponse

		grant, err = qp.authzKeeper.GranteeGrants(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "Error querying grantee grants")
		}

		for _, g := range grant.Grants {
			if g.Authorization != nil {
				ForceMarshalJSONAny(g.Authorization)
			}
		}

		bz, err = json.Marshal(&authz.QueryGranteeGrantsResponse{
			Grants:     grant.Grants,
			Pagination: grant.Pagination,
		})
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("unknown authz query variant: %+v", string(queryData))}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleStakingQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.StakingQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective StakingQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.StakedAmount != nil:
		req := query.StakedAmount

		var delegatorAccAddress sdk.AccAddress
		delegatorAccAddress, err = sdk.AccAddressFromBech32(req.DelegatorAddress)

		if err != nil {
			return nil, err
		}

		stakedINJ := qp.exchangeKeeper.CalculateStakedAmountWithoutCache(ctx, delegatorAccAddress, req.MaxDelegations)

		bz, err = json.Marshal(bindings.StakingDelegationAmountResponse{
			StakedAmount: stakedINJ,
		})
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("unknown staking query variant: %+v", string(queryData))}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleOracleQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.OracleQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective OracleQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.OracleVolatility != nil:
		req := query.OracleVolatility
		vol, raw, meta := qp.oracleKeeper.GetOracleVolatility(ctx, req.BaseInfo, req.QuoteInfo, req.OracleHistoryOptions)

		bz, err = json.Marshal(oracletypes.QueryOracleVolatilityResponse{
			Volatility:      vol,
			RawHistory:      raw,
			HistoryMetadata: meta,
		})
	case query.OraclePrice != nil:
		req := query.OraclePrice

		if req.GetOracleType() == oracletypes.OracleType_Provider {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "provider oracle is not supported"}
		}

		pricePairState := qp.oracleKeeper.GetPricePairState(ctx, req.GetOracleType(), req.GetBase(), req.GetQuote(), req.ScalingOptions)

		if pricePairState == nil {
			return nil, oracletypes.ErrOraclePriceNotFound
		}

		bz, err = json.Marshal(oracletypes.QueryOraclePriceResponse{
			PricePairState: pricePairState,
		})
	case query.PythPrice != nil:
		req := query.PythPrice

		if !exchangetypes.IsHexHash(req.PriceId) {
			return nil, errors.Wrap(err, "Error invalid price_id")
		}

		pythPriceState := qp.oracleKeeper.GetPythPriceState(ctx, common.HexToHash(req.PriceId))

		if pythPriceState == nil {
			return nil, oracletypes.ErrOraclePriceNotFound
		}

		bz, err = json.Marshal(oracletypes.QueryPythPriceResponse{
			PriceState: pythPriceState,
		})
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("unknown oracle query variant: %+v", string(queryData))}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleAuctionQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.AuctionQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective AuctionQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.LastAuctionResult != nil:
		result := qp.auctionKeeper.GetLastAuctionResult(ctx)

		bz, err = json.Marshal(auctiontypes.QueryLastAuctionResultResponse{
			LastAuctionResult: result,
		})
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("unknown auction query variant: %+v", string(queryData))}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleExchangeQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.ExchangeQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective ExchangeQuery")
	}

	queryServer := exchangekeeper.NewV1QueryServer(qp.exchangeKeeper)

	switch {

	case query.ExchangeParams != nil:
		response, err := queryServer.QueryExchangeParams(ctx, query.ExchangeParams)
		if err != nil {
			return json.Marshal(exchangetypes.QueryExchangeParamsResponse{})
		}
		return json.Marshal(response)
	case query.SubaccountDeposit != nil:
		response, err := queryServer.SubaccountDeposit(ctx, query.SubaccountDeposit)
		if err != nil {
			return json.Marshal(bindings.SubaccountDepositQueryResponse{})
		}
		return json.Marshal(bindings.SubaccountDepositQueryResponse{Deposits: response.Deposits})
	case query.SpotMarket != nil:
		response, err := queryServer.SpotMarket(ctx, query.SpotMarket)
		if err != nil {
			return json.Marshal(exchangetypes.QuerySpotMarketResponse{})
		}
		return json.Marshal(response)
	case query.DerivativeMarket != nil:
		response, err := queryServer.DerivativeMarket(ctx, query.DerivativeMarket)
		if err != nil {
			return json.Marshal(bindings.DerivativeMarketQueryResponse{Market: nil})
		}
		return json.Marshal(bindings.DerivativeMarketQueryResponse{Market: &bindings.FullDerivativeMarketQuery{
			Market:    response.Market.Market,
			Info:      response.Market.Info.(*exchangetypes.FullDerivativeMarket_PerpetualInfo),
			MarkPrice: response.Market.MarkPrice,
		}})
	case query.SubaccountPositions != nil:
		response, err := queryServer.SubaccountPositions(ctx, query.SubaccountPositions)
		if err != nil {
			return json.Marshal(exchangetypes.QuerySubaccountPositionsResponse{})
		}
		return json.Marshal(response)
	case query.SubaccountPositionInMarket != nil:
		response, err := queryServer.SubaccountPositionInMarket(ctx, query.SubaccountPositionInMarket)
		if err != nil {
			return json.Marshal(exchangetypes.QuerySubaccountPositionInMarketResponse{})
		}
		return json.Marshal(response)
	case query.SubaccountEffectivePositionInMarket != nil:
		response, err := queryServer.SubaccountEffectivePositionInMarket(ctx, query.SubaccountEffectivePositionInMarket)
		if err != nil {
			return json.Marshal(exchangetypes.QuerySubaccountEffectivePositionInMarketResponse{})
		}
		return json.Marshal(response)
	case query.SubaccountOrders != nil:
		response, err := queryServer.SubaccountOrders(ctx, query.SubaccountOrders)
		if err != nil {
			return json.Marshal(exchangetypes.QuerySubaccountOrdersResponse{})
		}
		return json.Marshal(response)
	case query.TraderDerivativeOrders != nil:
		response, err := queryServer.TraderDerivativeOrders(ctx, query.TraderDerivativeOrders)
		if err != nil {
			return json.Marshal(exchangetypes.QueryTraderDerivativeOrdersResponse{})
		}
		return json.Marshal(response)
	case query.TraderSpotOrdersToCancelUpToAmountRequest != nil:
		marketID := common.HexToHash(query.TraderSpotOrdersToCancelUpToAmountRequest.MarketId)
		subaccountID := common.HexToHash(query.TraderSpotOrdersToCancelUpToAmountRequest.SubaccountId)
		market := qp.exchangeKeeper.GetSpotMarket(ctx, marketID, true)

		referencePrice := query.TraderSpotOrdersToCancelUpToAmountRequest.ReferencePrice
		if referencePrice != nil {
			referencePriceChainFormat := market.PriceFromChainFormat(*query.TraderSpotOrdersToCancelUpToAmountRequest.ReferencePrice)
			referencePrice = &referencePriceChainFormat
		}

		ordersToCancel, hasProcessedFullAmount := qp.exchangeKeeper.GetSpotOrdersToCancelUpToAmount(
			ctx,
			market,
			qp.exchangeKeeper.GetAllTraderSpotLimitOrders(ctx, marketID, subaccountID),
			exchangev2.CancellationStrategy(query.TraderSpotOrdersToCancelUpToAmountRequest.Strategy),
			referencePrice,
			market.QuantityFromChainFormat(query.TraderSpotOrdersToCancelUpToAmountRequest.BaseAmount),
			market.NotionalFromChainFormat(query.TraderSpotOrdersToCancelUpToAmountRequest.QuoteAmount),
		)

		if hasProcessedFullAmount {
			ordersToCancelV1 := make([]*exchangetypes.TrimmedSpotLimitOrder, 0, len(ordersToCancel))
			for _, order := range ordersToCancel {
				ordersToCancelV1 = append(ordersToCancelV1, exchangekeeper.NewV1TrimmedSpotLimitOrderFromV2(market, order))
			}

			return json.Marshal(exchangetypes.QueryTraderSpotOrdersResponse{
				Orders: ordersToCancelV1,
			})
		} else {
			return nil, exchangetypes.ErrTransientOrdersUpToCancelNotSupported
		}
	case query.TraderDerivativeOrdersToCancelUpToAmountRequest != nil:
		marketID := common.HexToHash(query.TraderDerivativeOrdersToCancelUpToAmountRequest.MarketId)
		subaccountID := common.HexToHash(query.TraderDerivativeOrdersToCancelUpToAmountRequest.SubaccountId)
		market := qp.exchangeKeeper.GetDerivativeMarket(ctx, marketID, true)
		traderOrders := qp.exchangeKeeper.GetAllTraderDerivativeLimitOrders(ctx, marketID, subaccountID)

		referencePrice := query.TraderDerivativeOrdersToCancelUpToAmountRequest.ReferencePrice
		if referencePrice != nil {
			referencePriceChainFormat := market.PriceFromChainFormat(*query.TraderDerivativeOrdersToCancelUpToAmountRequest.ReferencePrice)
			referencePrice = &referencePriceChainFormat
		}

		ordersToCancel, hasProcessedFullAmount := exchangekeeper.GetDerivativeOrdersToCancelUpToAmount(
			market,
			traderOrders,
			exchangev2.CancellationStrategy(query.TraderDerivativeOrdersToCancelUpToAmountRequest.Strategy),
			referencePrice,
			market.NotionalFromChainFormat(query.TraderDerivativeOrdersToCancelUpToAmountRequest.QuoteAmount),
		)

		if hasProcessedFullAmount {
			ordersToCancelV1 := make([]*exchangetypes.TrimmedDerivativeLimitOrder, 0, len(ordersToCancel))
			for _, order := range ordersToCancel {
				orderV1 := exchangekeeper.NewV1TrimmedDerivativeLimitOrderFromV2(market, *order)
				ordersToCancelV1 = append(ordersToCancelV1, &orderV1)
			}

			return json.Marshal(exchangetypes.QueryTraderDerivativeOrdersResponse{
				Orders: ordersToCancelV1,
			})
		} else {
			return nil, exchangetypes.ErrTransientOrdersUpToCancelNotSupported
		}
	case query.TraderSpotOrders != nil:
		response, err := queryServer.TraderSpotOrders(ctx, query.TraderSpotOrders)
		if err != nil {
			return json.Marshal(exchangetypes.QueryTraderSpotOrdersResponse{})
		}
		return json.Marshal(response)
	case query.TraderTransientSpotOrders != nil:
		response, err := queryServer.TraderSpotTransientOrders(ctx, query.TraderTransientSpotOrders)
		if err != nil {
			return json.Marshal(exchangetypes.QueryTraderSpotOrdersResponse{})
		}
		return json.Marshal(response)
	case query.SpotOrderbook != nil:
		response, err := queryServer.SpotOrderbook(ctx, query.SpotOrderbook)
		if err != nil {
			return json.Marshal(exchangetypes.QuerySpotOrderbookResponse{})
		}
		return json.Marshal(response)
	case query.DerivativeOrderbook != nil:
		response, err := queryServer.DerivativeOrderbook(ctx, query.DerivativeOrderbook)
		if err != nil {
			return json.Marshal(exchangetypes.QueryDerivativeOrderbookResponse{})
		}
		return json.Marshal(response)
	case query.TraderTransientDerivativeOrders != nil:
		response, err := queryServer.TraderDerivativeTransientOrders(ctx, query.TraderTransientDerivativeOrders)
		if err != nil {
			return json.Marshal(exchangetypes.QueryTraderDerivativeOrdersResponse{})
		}
		return json.Marshal(response)
	case query.PerpetualMarketInfo != nil:
		response, err := queryServer.PerpetualMarketInfo(ctx, query.PerpetualMarketInfo)
		if err != nil {
			return json.Marshal(exchangetypes.QueryPerpetualMarketInfoResponse{
				Info: exchangetypes.PerpetualMarketInfo{},
			})
		}
		return json.Marshal(response)
	case query.ExpiryFuturesMarketInfo != nil:
		response, err := queryServer.ExpiryFuturesMarketInfo(ctx, query.ExpiryFuturesMarketInfo)
		if err != nil {
			return json.Marshal(exchangetypes.QueryExpiryFuturesMarketInfoResponse{
				Info: exchangetypes.ExpiryFuturesMarketInfo{},
			})
		}
		return json.Marshal(response)
	case query.PerpetualMarketFunding != nil:
		response, err := queryServer.PerpetualMarketFunding(ctx, query.PerpetualMarketFunding)
		if err != nil {
			return json.Marshal(exchangetypes.QueryPerpetualMarketFundingResponse{
				State: exchangetypes.PerpetualMarketFunding{},
			})
		}
		return json.Marshal(response)
	case query.MarketVolatility != nil:
		response, err := queryServer.MarketVolatility(ctx, query.MarketVolatility)
		if err != nil {
			return json.Marshal(exchangetypes.QueryMarketVolatilityResponse{})
		}
		return json.Marshal(response)
	case query.SpotMarketMidPriceAndTOB != nil:
		response, err := queryServer.SpotMidPriceAndTOB(ctx, query.SpotMarketMidPriceAndTOB)
		if err != nil {
			return json.Marshal(exchangetypes.QuerySpotMidPriceAndTOBResponse{})
		}
		return json.Marshal(response)
	case query.DerivativeMarketMidPriceAndTOB != nil:
		response, err := queryServer.DerivativeMidPriceAndTOB(ctx, query.DerivativeMarketMidPriceAndTOB)
		if err != nil {
			return json.Marshal(exchangetypes.QueryDerivativeMidPriceAndTOBResponse{})
		}
		return json.Marshal(response)
	case query.MarketAtomicExecutionFeeMultiplier != nil:
		response, err := queryServer.MarketAtomicExecutionFeeMultiplier(ctx, query.MarketAtomicExecutionFeeMultiplier)
		if err != nil {
			return nil, err
		}
		return json.Marshal(response)
	case query.AggregateMarketVolume != nil:
		response, err := queryServer.AggregateMarketVolume(ctx, query.AggregateMarketVolume)
		if err != nil {
			return json.Marshal(exchangetypes.QueryAggregateMarketVolumeResponse{})
		}
		return json.Marshal(response)
	case query.AggregateAccountVolume != nil:
		response, err := queryServer.AggregateVolume(ctx, query.AggregateAccountVolume)
		if err != nil {
			return json.Marshal(exchangetypes.QueryAggregateVolumeResponse{})
		}
		return json.Marshal(response)
	case query.DenomDecimal != nil:
		response, err := queryServer.DenomDecimal(ctx, query.DenomDecimal)
		if err != nil {
			return json.Marshal(exchangetypes.QueryDenomDecimalResponse{})
		}
		return json.Marshal(response)
	case query.DenomDecimals != nil:
		response, err := queryServer.DenomDecimals(ctx, query.DenomDecimals)
		if err != nil {
			return json.Marshal(exchangetypes.QueryDenomDecimalsResponse{})
		}
		return json.Marshal(response)
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown exchange query variant"}
	}
}

func (qp QueryPlugin) HandleTokenFactoryQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.TokenfactoryQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective TokenfactoryQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.DenomAdmin != nil:
		var metadata tokenfactorytypes.DenomAuthorityMetadata
		metadata, err = qp.tokenFactoryKeeper.GetAuthorityMetadata(ctx, query.DenomAdmin.Subdenom)
		if err != nil {
			return nil, err
		}

		return json.Marshal(&bindings.DenomAdminResponse{
			Admin: metadata.Admin,
		})
	case query.DenomTotalSupply != nil:
		supply := qp.bankKeeper.GetSupply(ctx, query.DenomTotalSupply.Denom)
		bz, err = json.Marshal(&bindings.DenomTotalSupplyResponse{
			TotalSupply: supply.Amount,
		})
	case query.DenomCreationFee != nil:
		fee := qp.tokenFactoryKeeper.GetParams(ctx).DenomCreationFee
		bz, err = json.Marshal(&bindings.DenomCreationFeeResponse{
			Fee: fee,
		})
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown tokenfactory query variant"}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleWasmxQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.WasmxQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective WasmxQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.RegisteredContractInfo != nil:
		var contractAddress sdk.AccAddress
		contractAddress, err = sdk.AccAddressFromBech32(query.RegisteredContractInfo.ContractAddress)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing contract address")
		}

		contract := qp.wasmxKeeper.GetContractByAddress(ctx, contractAddress)
		bz, err = json.Marshal(wasmxtypes.QueryContractRegistrationInfoResponse{Contract: contract})
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown wasmx query variant"}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleFeeGrantQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.FeeGrantQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective WasmxQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.Allowance != nil:
		var allowance *feegrant.QueryAllowanceResponse
		allowance, err = qp.feegrantKeeper.Allowance(ctx, query.Allowance)
		if err != nil {
			return nil, errors.Wrap(err, "Error retrieving allowance")
		}

		bz, err = json.Marshal(allowance)
	case query.Allowances != nil:
		var allowances *feegrant.QueryAllowancesResponse
		allowances, err = qp.feegrantKeeper.Allowances(ctx, query.Allowances)
		if err != nil {
			return nil, errors.Wrap(err, "Error retrieving allowances")
		}

		bz, err = json.Marshal(allowances)
	case query.AllowancesByGranter != nil:
		var allowancesByGranter *feegrant.QueryAllowancesByGranterResponse
		allowancesByGranter, err = qp.feegrantKeeper.AllowancesByGranter(ctx, query.AllowancesByGranter)
		if err != nil {
			return nil, errors.Wrap(err, "Error retrieving allowances by granter")
		}

		bz, err = json.Marshal(allowancesByGranter)
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown feegrant query variant"}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}
