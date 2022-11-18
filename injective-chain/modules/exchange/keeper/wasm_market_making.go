package keeper

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) QueryMarketID(
	ctx sdk.Context,
	subaccountID common.Hash,
) (common.Hash, error) {
	type GetMarketIDQuery struct {
		SubaccountID string `json:"subaccount_id"`
	}

	type QueryData struct {
		Data GetMarketIDQuery `json:"get_market_id"`
	}

	queryData := QueryData{
		Data: GetMarketIDQuery{
			SubaccountID: subaccountID.Hex(),
		},
	}
	queryDataBz, err := json.Marshal(queryData)
	if err != nil {
		return common.Hash{}, err
	}

	contractAddress := types.SubaccountIDToSdkAddress(subaccountID)
	bz, err := k.wasmViewKeeper.QuerySmart(ctx, contractAddress, queryDataBz)
	if err != nil {
		return common.Hash{}, err
	}

	type Data struct {
		MarketId string `json:"market_id"`
	}

	var result Data

	if err := json.Unmarshal(bz, &result); err != nil {
		return common.Hash{}, err
	}

	return common.HexToHash(result.MarketId), nil
}

func (k *Keeper) QueryTotalSupply(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	subaccountID common.Hash,
) (sdk.Int, error) {
	type GetTotalSupplyQuery struct {
		SubaccountID string `json:"subaccount_id"`
	}

	type QueryData struct {
		Data GetTotalSupplyQuery `json:"get_total_lp_supply"`
	}

	queryData := QueryData{
		Data: GetTotalSupplyQuery{
			SubaccountID: subaccountID.Hex(),
		},
	}
	queryDataBz, err := json.Marshal(queryData)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	bz, err := k.wasmViewKeeper.QuerySmart(ctx, contractAddress, queryDataBz)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	type Data struct {
		TotalSupply sdk.Int `json:"total_supply"`
	}

	var result Data

	if err := json.Unmarshal(bz, &result); err != nil {
		return sdk.ZeroInt(), err
	}

	return result.TotalSupply, nil
}

func (k *Keeper) QueryTokenBalance(
	ctx sdk.Context,
	cw20ContractAddress sdk.AccAddress,
	user string,
) (sdk.Int, error) {
	type Balance struct {
		Address string `json:"address"`
	}

	type QueryData struct {
		Data Balance `json:"balance"`
	}

	queryData := QueryData{
		Data: Balance{
			Address: user,
		},
	}
	queryDataBz, err := json.Marshal(queryData)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	bz, err := k.wasmViewKeeper.QuerySmart(ctx, cw20ContractAddress, queryDataBz)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	type BalanceResponse struct {
		Balance sdk.Int `json:"balance"`
	}

	var result BalanceResponse

	if err := json.Unmarshal(bz, &result); err != nil {
		return sdk.ZeroInt(), err
	}

	return result.Balance, nil
}
