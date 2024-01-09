package wasmbinding

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/ethereum/go-ethereum/common"

	wasmxkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"

	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oraclekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/wasmbinding/bindings"
)

type QueryPlugin struct {
	authzKeeper        *authzkeeper.Keeper
	bankKeeper         *bankkeeper.BaseKeeper
	exchangeKeeper     *exchangekeeper.Keeper
	feegrantKeeper     *feegrantkeeper.Keeper
	oracleKeeper       *oraclekeeper.Keeper
	tokenFactoryKeeper *tokenfactorykeeper.Keeper
	wasmxKeeper        *wasmxkeeper.Keeper
}

// NewQueryPlugin returns a reference to a new QueryPlugin.
func NewQueryPlugin(
	ak *authzkeeper.Keeper,
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

	err = msgAny.UnmarshalJSON(bz)
	if err != nil {
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

		pricePairState := qp.oracleKeeper.GetPricePairState(ctx, req.GetOracleType(), req.GetBase(), req.GetQuote())

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

func (qp QueryPlugin) HandleExchangeQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.ExchangeQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, errors.Wrap(err, "Error parsing Injective ExchangeQuery")
	}

	var bz []byte
	var err error

	switch {
	case query.ExchangeParams != nil:
		params := qp.exchangeKeeper.GetParams(ctx)
		bz, err = json.Marshal(exchangetypes.QueryExchangeParamsResponse{Params: params})
	case query.SubaccountDeposit != nil:
		deposit := qp.exchangeKeeper.GetDeposit(ctx, common.HexToHash(query.SubaccountDeposit.SubaccountId), query.SubaccountDeposit.Denom)
		bz, err = json.Marshal(bindings.SubaccountDepositQueryResponse{Deposits: deposit})
	case query.SpotMarket != nil:
		market := qp.exchangeKeeper.GetSpotMarketByID(ctx, common.HexToHash(query.SpotMarket.MarketId))
		bz, err = json.Marshal(exchangetypes.QuerySpotMarketResponse{Market: market})
	case query.DerivativeMarket != nil:
		market := qp.exchangeKeeper.GetFullDerivativeMarket(ctx, common.HexToHash(query.DerivativeMarket.MarketId), true)
		if market != nil {
			bz, err = json.Marshal(bindings.DerivativeMarketQueryResponse{Market: &bindings.FullDerivativeMarketQuery{
				Market:    market.Market,
				Info:      market.Info.(*exchangetypes.FullDerivativeMarket_PerpetualInfo),
				MarkPrice: market.MarkPrice,
			}})
		} else {
			bz, err = json.Marshal(bindings.DerivativeMarketQueryResponse{Market: nil})
		}
	case query.SubaccountPositions != nil:
		positions := qp.exchangeKeeper.GetAllActivePositionsBySubaccountID(ctx, common.HexToHash(query.SubaccountPositions.SubaccountId))
		bz, err = json.Marshal(exchangetypes.QuerySubaccountPositionsResponse{State: positions})
	case query.SubaccountPositionInMarket != nil:
		position := qp.exchangeKeeper.GetPosition(ctx, common.HexToHash(query.SubaccountPositionInMarket.MarketId), common.HexToHash(query.SubaccountPositionInMarket.SubaccountId))
		bz, err = json.Marshal(exchangetypes.QuerySubaccountPositionInMarketResponse{State: position})
	case query.SubaccountEffectivePositionInMarket != nil:
		marketID := common.HexToHash(query.SubaccountEffectivePositionInMarket.MarketId)
		position := qp.exchangeKeeper.GetPosition(ctx, marketID, common.HexToHash(query.SubaccountEffectivePositionInMarket.SubaccountId))

		if position == nil {
			bz, err = json.Marshal(exchangetypes.QuerySubaccountEffectivePositionInMarketResponse{State: nil})
		} else {
			_, markPrice := qp.exchangeKeeper.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
			funding := qp.exchangeKeeper.GetPerpetualMarketFunding(ctx, marketID)

			effectivePosition := exchangetypes.EffectivePosition{
				IsLong:          position.IsLong,
				EntryPrice:      position.EntryPrice,
				Quantity:        position.Quantity,
				EffectiveMargin: position.GetEffectiveMargin(funding, markPrice),
			}
			bz, err = json.Marshal(exchangetypes.QuerySubaccountEffectivePositionInMarketResponse{State: &effectivePosition})
		}
	case query.SubaccountOrders != nil:
		marketID := common.HexToHash(query.SubaccountOrders.MarketId)
		subaccountID := common.HexToHash(query.SubaccountOrders.SubaccountId)

		buyOrders := qp.exchangeKeeper.GetSubaccountOrders(ctx, marketID, subaccountID, true, false)
		sellOrders := qp.exchangeKeeper.GetSubaccountOrders(ctx, marketID, subaccountID, false, false)

		bz, err = json.Marshal(exchangetypes.QuerySubaccountOrdersResponse{
			BuyOrders:  buyOrders,
			SellOrders: sellOrders,
		})
	case query.TraderDerivativeOrders != nil:
		marketID := common.HexToHash(query.TraderDerivativeOrders.MarketId)
		subaccountID := common.HexToHash(query.TraderDerivativeOrders.SubaccountId)
		orders := qp.exchangeKeeper.GetAllTraderDerivativeLimitOrders(ctx, marketID, subaccountID)

		bz, err = json.Marshal(exchangetypes.QueryTraderDerivativeOrdersResponse{
			Orders: orders,
		})
	case query.TraderSpotOrdersToCancelUpToAmountRequest != nil:
		marketID := common.HexToHash(query.TraderSpotOrdersToCancelUpToAmountRequest.MarketId)
		subaccountID := common.HexToHash(query.TraderSpotOrdersToCancelUpToAmountRequest.SubaccountId)
		market := qp.exchangeKeeper.GetSpotMarket(ctx, marketID, true)

		traderOrders := qp.exchangeKeeper.GetAllTraderSpotLimitOrders(ctx, marketID, subaccountID)
		ordersToCancel, hasProcessedFullAmount := exchangekeeper.GetSpotOrdersToCancelUpToAmount(
			market,
			traderOrders,
			query.TraderSpotOrdersToCancelUpToAmountRequest.Strategy,
			query.TraderSpotOrdersToCancelUpToAmountRequest.ReferencePrice,
			query.TraderSpotOrdersToCancelUpToAmountRequest.BaseAmount,
			query.TraderSpotOrdersToCancelUpToAmountRequest.QuoteAmount,
		)

		if hasProcessedFullAmount {
			bz, err = json.Marshal(exchangetypes.QueryTraderSpotOrdersResponse{
				Orders: ordersToCancel,
			})
		} else {
			err = exchangetypes.ErrTransientOrdersUpToCancelNotSupported
		}
	case query.TraderDerivativeOrdersToCancelUpToAmountRequest != nil:
		marketID := common.HexToHash(query.TraderDerivativeOrdersToCancelUpToAmountRequest.MarketId)
		subaccountID := common.HexToHash(query.TraderDerivativeOrdersToCancelUpToAmountRequest.SubaccountId)
		market := qp.exchangeKeeper.GetDerivativeMarket(ctx, marketID, true)

		traderOrders := qp.exchangeKeeper.GetAllTraderDerivativeLimitOrders(ctx, marketID, subaccountID)
		ordersToCancel, hasProcessedFullAmount := exchangekeeper.GetDerivativeOrdersToCancelUpToAmount(
			market,
			traderOrders,
			query.TraderDerivativeOrdersToCancelUpToAmountRequest.Strategy,
			query.TraderDerivativeOrdersToCancelUpToAmountRequest.ReferencePrice,
			query.TraderDerivativeOrdersToCancelUpToAmountRequest.QuoteAmount,
		)

		if hasProcessedFullAmount {
			bz, err = json.Marshal(exchangetypes.QueryTraderDerivativeOrdersResponse{
				Orders: ordersToCancel,
			})
		} else {
			err = exchangetypes.ErrTransientOrdersUpToCancelNotSupported
		}
	case query.TraderSpotOrders != nil:
		marketId := common.HexToHash(query.TraderSpotOrders.MarketId)
		subaccountId := common.HexToHash(query.TraderSpotOrders.SubaccountId)
		orders := qp.exchangeKeeper.GetAllTraderSpotLimitOrders(ctx, marketId, subaccountId)

		bz, err = json.Marshal(exchangetypes.QueryTraderSpotOrdersResponse{
			Orders: orders,
		})
	case query.TraderTransientSpotOrders != nil:
		marketId := common.HexToHash(query.TraderTransientSpotOrders.MarketId)
		subaccountId := common.HexToHash(query.TraderTransientSpotOrders.SubaccountId)
		orders := qp.exchangeKeeper.GetAllTransientTraderSpotLimitOrders(ctx, marketId, subaccountId)

		bz, err = json.Marshal(exchangetypes.QueryTraderSpotOrdersResponse{
			Orders: orders,
		})
	case query.SpotOrderbook != nil:
		req := query.SpotOrderbook
		marketID := common.HexToHash(req.MarketId)

		var limit *uint64
		if req.Limit > 0 {
			limit = &req.Limit
		} else if req.LimitCumulativeNotional == nil && req.LimitCumulativeQuantity == nil {
			defaultLimit := exchangetypes.DefaultQueryOrderbookLimit
			limit = &defaultLimit
		}
		buysLevels := make([]*exchangetypes.Level, 0)
		if req.OrderSide == exchangetypes.OrderSide_Buy || req.OrderSide == exchangetypes.OrderSide_Side_Unspecified {
			buysLevels = qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, true, marketID, true, limit, req.LimitCumulativeNotional, req.LimitCumulativeQuantity)
		}
		sellLevels := make([]*exchangetypes.Level, 0)
		if req.OrderSide == exchangetypes.OrderSide_Sell || req.OrderSide == exchangetypes.OrderSide_Side_Unspecified {
			sellLevels = qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, true, marketID, false, limit, req.LimitCumulativeNotional, req.LimitCumulativeQuantity)
		}
		bz, err = json.Marshal(exchangetypes.QuerySpotOrderbookResponse{
			BuysPriceLevel:  buysLevels,
			SellsPriceLevel: sellLevels,
		})
	case query.DerivativeOrderbook != nil:
		req := query.DerivativeOrderbook
		marketID := common.HexToHash(req.MarketId)

		var limit *uint64
		if req.Limit > 0 {
			limit = &req.Limit
		} else if req.LimitCumulativeNotional == nil {
			defaultLimit := exchangetypes.DefaultQueryOrderbookLimit
			limit = &defaultLimit
		}

		bz, err = json.Marshal(exchangetypes.QueryDerivativeOrderbookResponse{
			BuysPriceLevel:  qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, false, marketID, true, limit, req.LimitCumulativeNotional, nil),
			SellsPriceLevel: qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, false, marketID, false, limit, req.LimitCumulativeNotional, nil),
		})
	case query.TraderTransientDerivativeOrders != nil:
		marketId := common.HexToHash(query.TraderTransientDerivativeOrders.MarketId)
		subaccountId := common.HexToHash(query.TraderTransientDerivativeOrders.SubaccountId)
		orders := qp.exchangeKeeper.GetAllTransientTraderDerivativeLimitOrders(ctx, marketId, subaccountId)

		bz, err = json.Marshal(exchangetypes.QueryTraderDerivativeOrdersResponse{
			Orders: orders,
		})
	case query.PerpetualMarketInfo != nil:
		info := qp.exchangeKeeper.GetPerpetualMarketInfo(ctx, common.HexToHash(query.PerpetualMarketInfo.MarketId))
		if info != nil {
			bz, err = json.Marshal(exchangetypes.QueryPerpetualMarketInfoResponse{
				Info: *info,
			})
		} else {
			bz, err = json.Marshal(exchangetypes.QueryPerpetualMarketInfoResponse{
				Info: exchangetypes.PerpetualMarketInfo{},
			})
		}
	case query.ExpiryFuturesMarketInfo != nil:
		info := qp.exchangeKeeper.GetExpiryFuturesMarketInfo(ctx, common.HexToHash(query.ExpiryFuturesMarketInfo.MarketId))
		if info != nil {
			bz, err = json.Marshal(exchangetypes.QueryExpiryFuturesMarketInfoResponse{
				Info: *info,
			})
		} else {
			bz, err = json.Marshal(exchangetypes.QueryExpiryFuturesMarketInfoResponse{
				Info: exchangetypes.ExpiryFuturesMarketInfo{},
			})
		}
	case query.PerpetualMarketFunding != nil:
		funding := qp.exchangeKeeper.GetPerpetualMarketFunding(ctx, common.HexToHash(query.PerpetualMarketFunding.MarketId))
		if funding != nil {
			bz, err = json.Marshal(exchangetypes.QueryPerpetualMarketFundingResponse{
				State: *funding,
			})
		} else {
			bz, err = json.Marshal(exchangetypes.QueryPerpetualMarketFundingResponse{
				State: exchangetypes.PerpetualMarketFunding{},
			})
		}
	case query.MarketVolatility != nil:
		req := query.MarketVolatility
		vol, rawHistory, meta := qp.exchangeKeeper.GetMarketVolatility(ctx, common.HexToHash(req.MarketId), req.TradeHistoryOptions)
		bz, err = json.Marshal(exchangetypes.QueryMarketVolatilityResponse{
			Volatility:      vol,
			RawHistory:      rawHistory,
			HistoryMetadata: meta,
		})
	case query.SpotMarketMidPriceAndTOB != nil:
		req := query.SpotMarketMidPriceAndTOB
		midPrice, bestBuyPrice, bestSellPrice := qp.exchangeKeeper.GetSpotMidPriceAndTOB(ctx, common.HexToHash(req.MarketId))

		bz, err = json.Marshal(exchangetypes.QuerySpotMidPriceAndTOBResponse{
			MidPrice:      midPrice,
			BestBuyPrice:  bestBuyPrice,
			BestSellPrice: bestSellPrice,
		})
	case query.DerivativeMarketMidPriceAndTOB != nil:
		req := query.DerivativeMarketMidPriceAndTOB
		midPrice, bestBuyPrice, bestSellPrice := qp.exchangeKeeper.GetDerivativeMidPriceAndTOB(ctx, common.HexToHash(req.MarketId))

		bz, err = json.Marshal(exchangetypes.QueryDerivativeMidPriceAndTOBResponse{
			MidPrice:      midPrice,
			BestBuyPrice:  bestBuyPrice,
			BestSellPrice: bestSellPrice,
		})
	case query.MarketAtomicExecutionFeeMultiplier != nil:
		req := query.MarketAtomicExecutionFeeMultiplier
		marketID := common.HexToHash(req.MarketId)
		marketType, err := qp.exchangeKeeper.GetMarketType(ctx, marketID, true)
		if err != nil {
			return nil, err
		}
		multiplier := qp.exchangeKeeper.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, *marketType)
		// nolint:all //tool is wrong, ofc this assignment is effectual
		bz, err = json.Marshal(exchangetypes.QueryMarketAtomicExecutionFeeMultiplierResponse{
			Multiplier: multiplier,
		})
	case query.AggregateMarketVolume != nil:
		req := query.AggregateMarketVolume
		marketID := common.HexToHash(req.MarketId)
		volume := qp.exchangeKeeper.GetMarketAggregateVolume(ctx, marketID)
		res := &exchangetypes.QueryAggregateMarketVolumeResponse{
			Volume: volume,
		}
		bz, err = json.Marshal(res)
	case query.AggregateAccountVolume != nil:
		req := query.AggregateAccountVolume
		if exchangetypes.IsHexHash(req.Account) {
			subaccountID := common.HexToHash(req.Account)
			volumes := qp.exchangeKeeper.GetAllSubaccountMarketAggregateVolumesBySubaccount(ctx, subaccountID)
			res := &exchangetypes.QueryAggregateVolumeResponse{AggregateVolumes: volumes}
			bz, err = json.Marshal(res)
		} else {
			accAddress, err2 := sdk.AccAddressFromBech32(req.Account)
			if err2 != nil {
				return nil, err2
			}
			volumes := qp.exchangeKeeper.GetAllSubaccountMarketAggregateVolumesByAccAddress(ctx, accAddress)
			res := &exchangetypes.QueryAggregateVolumeResponse{
				AggregateVolumes: volumes,
			}
			bz, err = json.Marshal(res)
		}
	case query.DenomDecimal != nil:
		var res *exchangetypes.QueryDenomDecimalResponse
		res, err = qp.exchangeKeeper.DenomDecimal(sdk.WrapSDKContext(ctx), query.DenomDecimal)
		if err != nil {
			return nil, err
		}
		bz, err = json.Marshal(res)
	case query.DenomDecimals != nil:
		var res *exchangetypes.QueryDenomDecimalsResponse
		res, err = qp.exchangeKeeper.DenomDecimals(sdk.WrapSDKContext(ctx), query.DenomDecimals)
		if err != nil {
			return nil, err
		}
		bz, err = json.Marshal(res)
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown exchange query variant"}
	}

	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
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

		bz, err = json.Marshal(&bindings.DenomAdminResponse{
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
