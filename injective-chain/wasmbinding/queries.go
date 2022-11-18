package wasmbinding

import (
	"encoding/json"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/common"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oraclekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/wasmbinding/bindings"
)

type QueryPlugin struct {
	exchangeKeeper     *exchangekeeper.Keeper
	oracleKeeper       *oraclekeeper.Keeper
	bankKeeper         *bankkeeper.BaseKeeper
	tokenFactoryKeeper *tokenfactorykeeper.Keeper
}

// NewQueryPlugin returns a reference to a new QueryPlugin.
func NewQueryPlugin(ek *exchangekeeper.Keeper, ok *oraclekeeper.Keeper, bk *bankkeeper.BaseKeeper, tfk *tokenfactorykeeper.Keeper) *QueryPlugin {
	return &QueryPlugin{
		exchangeKeeper:     ek,
		oracleKeeper:       ok,
		bankKeeper:         bk,
		tokenFactoryKeeper: tfk,
	}
}

func (qp QueryPlugin) HandleOracleQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.OracleQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, sdkerrors.Wrap(err, "Error parsing Injective OracleQuery")
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
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown oracle query variant"}
	}

	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleExchangeQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.ExchangeQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, sdkerrors.Wrap(err, "Error parsing Injective ExchangeQuery")
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

		limit := req.Limit
		if limit == 0 {
			limit = exchangetypes.DefaultQueryOrderbookLimit
		}

		bz, err = json.Marshal(exchangetypes.QuerySpotOrderbookResponse{
			BuysPriceLevel:  qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, true, marketID, true, &limit),
			SellsPriceLevel: qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, true, marketID, false, &limit),
		})
	case query.DerivativeOrderbook != nil:
		req := query.DerivativeOrderbook
		marketID := common.HexToHash(req.MarketId)

		limit := req.Limit
		if limit == 0 {
			limit = exchangetypes.DefaultQueryOrderbookLimit
		}

		bz, err = json.Marshal(exchangetypes.QueryDerivativeOrderbookResponse{
			BuysPriceLevel:  qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, false, marketID, true, &limit),
			SellsPriceLevel: qp.exchangeKeeper.GetOrderbookPriceLevels(ctx, false, marketID, false, &limit),
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
		marketType, err := qp.exchangeKeeper.GetMarketType(ctx, marketID)
		if err != nil {
			return nil, err
		}
		multiplier := qp.exchangeKeeper.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, *marketType)
		// nolint:all //tool is wrong, ofc this assignment is effectual
		bz, err = json.Marshal(exchangetypes.QueryMarketAtomicExecutionFeeMultiplierResponse{
			Multiplier: multiplier,
		})
	default:
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown exchange query variant"}
	}

	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func (qp QueryPlugin) HandleTokenFactoryQuery(ctx sdk.Context, queryData json.RawMessage) ([]byte, error) {
	var query bindings.TokenfactoryQuery
	if err := json.Unmarshal(queryData, &query); err != nil {
		return nil, sdkerrors.Wrap(err, "Error parsing Injective TokenfactoryQuery")
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
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}
