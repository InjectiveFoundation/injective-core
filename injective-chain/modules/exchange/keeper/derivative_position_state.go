package keeper

import (
	"bytes"
	"sort"

	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/ethereum/go-ethereum/common"
)

type PositionState struct {
	Position *v2.Position
}

func NewPositionStates() map[common.Hash]*PositionState {
	return make(map[common.Hash]*PositionState)
}

// ApplyFundingAndGetUpdatedPositionState updates the position to account for any funding payment and returns a PositionState.
func ApplyFundingAndGetUpdatedPositionState(p *v2.Position, funding *v2.PerpetualMarketFunding) *PositionState {
	p.ApplyFunding(funding)
	positionState := &PositionState{
		Position: p,
	}
	return positionState
}

func GetSortedSubaccountKeys(p map[common.Hash]*PositionState) []common.Hash {
	subaccountKeys := make([]common.Hash, 0)
	for k := range p {
		subaccountKeys = append(subaccountKeys, k)
	}
	sort.SliceStable(subaccountKeys, func(i, j int) bool {
		return bytes.Compare(subaccountKeys[i].Bytes(), subaccountKeys[j].Bytes()) < 0
	})
	return subaccountKeys
}

func GetPositionSliceData(p map[common.Hash]*PositionState) ([]*v2.Position, []common.Hash) {
	positionSubaccountIDs := GetSortedSubaccountKeys(p)
	positions := make([]*v2.Position, 0, len(positionSubaccountIDs))

	nonNilPositionSubaccountIDs := make([]common.Hash, 0)
	for idx := range positionSubaccountIDs {
		subaccountID := positionSubaccountIDs[idx]
		position := p[subaccountID]
		if position.Position != nil {
			positions = append(positions, position.Position)
			nonNilPositionSubaccountIDs = append(nonNilPositionSubaccountIDs, subaccountID)
		}

		// else {
		// 	fmt.Println("❌ position is nil for subaccount", subaccountID.Hex())
		// }
	}

	return positions, nonNilPositionSubaccountIDs
}
