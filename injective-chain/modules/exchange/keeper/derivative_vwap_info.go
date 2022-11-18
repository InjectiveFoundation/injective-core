package keeper

import (
	"bytes"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type VwapData struct {
	Price    sdk.Dec
	Quantity sdk.Dec
}

func NewVwapData() *VwapData {
	return &VwapData{
		Price:    sdk.ZeroDec(),
		Quantity: sdk.ZeroDec(),
	}
}

func (p *VwapData) ApplyExecution(price, quantity sdk.Dec) *VwapData {
	if p == nil {
		p = NewVwapData()
	}

	if price.IsNil() || quantity.IsNil() || quantity.IsZero() {
		return p
	}

	newQuantity := p.Quantity.Add(quantity)
	newPrice := p.Price.Mul(p.Quantity).Add(price.Mul(quantity)).Quo(newQuantity)

	return &VwapData{
		Price:    newPrice,
		Quantity: newQuantity,
	}
}

type VwapInfo struct {
	MarkPrice *sdk.Dec
	VwapData  *VwapData
}

func NewVwapInfo(markPrice *sdk.Dec) *VwapInfo {
	return &VwapInfo{
		MarkPrice: markPrice,
		VwapData:  NewVwapData(),
	}
}

type DerivativeVwapInfo struct {
	perpetualVwapInfo     map[common.Hash]*VwapInfo
	expiryVwapInfo        map[common.Hash]*VwapInfo
	binaryOptionsVwapInfo map[common.Hash]*VwapInfo
}

func NewDerivativeVwapInfo() DerivativeVwapInfo {
	return DerivativeVwapInfo{
		perpetualVwapInfo:     make(map[common.Hash]*VwapInfo),
		expiryVwapInfo:        make(map[common.Hash]*VwapInfo),
		binaryOptionsVwapInfo: make(map[common.Hash]*VwapInfo),
	}
}

func (p *DerivativeVwapInfo) ApplyVwap(marketID common.Hash, markPrice *sdk.Dec, vwapData *VwapData, marketType types.MarketType) {
	var vwapInfo *VwapInfo

	switch marketType {
	case types.MarketType_Perpetual:
		vwapInfo = p.perpetualVwapInfo[marketID]
		if vwapInfo == nil {
			vwapInfo = NewVwapInfo(markPrice)
			p.perpetualVwapInfo[marketID] = vwapInfo
		}
	case types.MarketType_Expiry:
		vwapInfo = p.expiryVwapInfo[marketID]
		if vwapInfo == nil {
			vwapInfo = NewVwapInfo(markPrice)
			p.expiryVwapInfo[marketID] = vwapInfo
		}
	case types.MarketType_BinaryOption:
		vwapInfo = p.binaryOptionsVwapInfo[marketID]
		if vwapInfo == nil {
			vwapInfo = NewVwapInfo(markPrice)
			p.binaryOptionsVwapInfo[marketID] = vwapInfo
		}
	}

	if !vwapData.Quantity.IsZero() {
		vwapInfo.VwapData = vwapInfo.VwapData.ApplyExecution(vwapData.Price, vwapData.Quantity)
	}
}

func (p *DerivativeVwapInfo) GetSortedPerpetualMarketIDs() []common.Hash {
	perpetualMarketIDs := make([]common.Hash, 0)
	for k := range p.perpetualVwapInfo {
		perpetualMarketIDs = append(perpetualMarketIDs, k)
	}

	sort.SliceStable(perpetualMarketIDs, func(i, j int) bool {
		return bytes.Compare(perpetualMarketIDs[i].Bytes(), perpetualMarketIDs[j].Bytes()) < 0
	})
	return perpetualMarketIDs
}

func (p *DerivativeVwapInfo) GetSortedExpiryFutureMarketIDs() []common.Hash {
	expiryFutureMarketIDs := make([]common.Hash, 0)
	for k := range p.expiryVwapInfo {
		expiryFutureMarketIDs = append(expiryFutureMarketIDs, k)
	}

	sort.SliceStable(expiryFutureMarketIDs, func(i, j int) bool {
		return bytes.Compare(expiryFutureMarketIDs[i].Bytes(), expiryFutureMarketIDs[j].Bytes()) < 0
	})
	return expiryFutureMarketIDs
}

func (p *DerivativeVwapInfo) GetSortedBinaryOptionsMarketIDs() []common.Hash {
	binaryOptionsMarketIDs := make([]common.Hash, 0)
	for k := range p.binaryOptionsVwapInfo {
		binaryOptionsMarketIDs = append(binaryOptionsMarketIDs, k)
	}

	sort.SliceStable(binaryOptionsMarketIDs, func(i, j int) bool {
		return bytes.Compare(binaryOptionsMarketIDs[i].Bytes(), binaryOptionsMarketIDs[j].Bytes()) < 0
	})
	return binaryOptionsMarketIDs
}

// ComputeSyntheticVwapUnitDelta returns (price - markPrice) / markPrice
func (p *DerivativeVwapInfo) ComputeSyntheticVwapUnitDelta(marketID common.Hash) sdk.Dec {
	vwapInfo := p.perpetualVwapInfo[marketID]
	return vwapInfo.VwapData.Price.Sub(*vwapInfo.MarkPrice).Quo(*vwapInfo.MarkPrice)
}
