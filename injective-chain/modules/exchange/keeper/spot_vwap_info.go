package keeper

import (
	"bytes"
	"sort"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
)

type SpotVwapData struct {
	Price    math.LegacyDec
	Quantity math.LegacyDec
}

func NewSpotVwapData() *SpotVwapData {
	return &SpotVwapData{
		Price:    math.LegacyZeroDec(),
		Quantity: math.LegacyZeroDec(),
	}
}

func (p *SpotVwapData) ApplyExecution(price, quantity math.LegacyDec) *SpotVwapData {
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
	spotMarketIDs := make([]common.Hash, 0)
	for k := range *p {
		spotMarketIDs = append(spotMarketIDs, k)
	}

	sort.SliceStable(spotMarketIDs, func(i, j int) bool {
		return bytes.Compare(spotMarketIDs[i].Bytes(), spotMarketIDs[j].Bytes()) < 0
	})
	return spotMarketIDs
}
