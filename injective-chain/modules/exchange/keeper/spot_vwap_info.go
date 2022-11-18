package keeper

import (
	"bytes"
	"sort"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type SpotVwapData struct {
	Price    types.Dec
	Quantity types.Dec
}

func NewSpotVwapData() *SpotVwapData {
	return &SpotVwapData{
		Price:    types.ZeroDec(),
		Quantity: types.ZeroDec(),
	}
}

func (p *SpotVwapData) ApplyExecution(price, quantity types.Dec) *SpotVwapData {
	if p == nil {
		p = NewSpotVwapData()
	}

	if price.IsNil() || quantity.IsNil() || quantity.IsZero() {
		return p
	}

	newQuantity := p.Quantity.Add(quantity)
	newPrice := p.Price.Mul(p.Quantity).Add(price.Mul(quantity)).Quo(newQuantity)

	return &SpotVwapData{
		Price:    newPrice,
		Quantity: newQuantity,
	}
}

type SpotVwapInfo map[common.Hash]*SpotVwapData

func NewSpotVwapInfo() SpotVwapInfo {
	return make(SpotVwapInfo)
}

func (p *SpotVwapInfo) ApplyVwap(marketID common.Hash, newVwapData *SpotVwapData) {
	var existingVwapData *SpotVwapData

	existingVwapData = (*p)[marketID]
	if existingVwapData == nil {
		existingVwapData = NewSpotVwapData()
		(*p)[marketID] = existingVwapData
	}

	if !newVwapData.Quantity.IsZero() {
		(*p)[marketID] = existingVwapData.ApplyExecution(newVwapData.Price, newVwapData.Quantity)
	}
}

func (p *SpotVwapInfo) GetSortedSpotMarketIDs() []common.Hash {
	spotMarketIds := make([]common.Hash, 0)
	for k := range *p {
		spotMarketIds = append(spotMarketIds, k)
	}

	sort.SliceStable(spotMarketIds, func(i, j int) bool {
		return bytes.Compare(spotMarketIds[i].Bytes(), spotMarketIds[j].Bytes()) < 0
	})
	return spotMarketIds
}
