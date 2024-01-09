package stream

import (
	"fmt"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
)

var ErrInvalidParameters = fmt.Errorf("firstMap and secondMap must have the same length")

func Filter[V types.OrderbookUpdate | types.BankBalance | types.OraclePrice | types.SubaccountDeposits](itemMap map[string][]*V, filter []string) (out []*V) {
	wildcard := false
	if len(filter) > 0 {
		wildcard = filter[0] == "*"
	}
	if wildcard {
		for _, items := range itemMap {
			out = append(out, items...)
		}
		return
	}
	for _, marketID := range filter {
		if updates, ok := itemMap[marketID]; ok {
			out = append(out, updates...)
		}
	}
	return
}

func FilterMulti[V types.Position | types.SpotTrade | types.DerivativeTrade | types.DerivativeOrderUpdate | types.SpotOrderUpdate](firstMap, secondMap map[string][]*V, firstFilter, secondFilter []string) (out []*V, err error) {

	if len(firstFilter) == 0 && len(secondFilter) == 0 {
		return
	}
	if len(firstMap) == 0 && len(secondMap) == 0 {
		return
	}

	if len(firstMap) == 0 || len(secondMap) == 0 {
		return nil, ErrInvalidParameters
	}

	firstWildcard, secondWildcard := false, false

	mapToSlice := func(m map[string]*V) []*V {
		outSlice := make([]*V, len(m))
		i := 0
		for _, position := range m {
			outSlice[i] = position
			i++
		}
		return outSlice
	}

	if len(firstFilter) > 0 {
		firstWildcard = firstFilter[0] == "*"
	}
	if len(secondFilter) > 0 {
		secondWildcard = secondFilter[0] == "*"
	}

	outMap := make(map[string]*V)

	// Fill first subset map
	firstSubsetMap := make(map[string]*V)
	if firstWildcard {
		for _, items := range firstMap {
			for _, item := range items {
				firstSubsetMap[getMemAddr(item)] = item
			}
		}
	} else {
		for _, marketID := range firstFilter {
			if items, ok := firstMap[marketID]; ok {
				for _, item := range items {
					firstSubsetMap[getMemAddr(item)] = item
				}
			}
		}
	}

	// Fill second subset map
	secondSubsetMap := make(map[string]*V)
	if secondWildcard {
		for _, items := range secondMap {
			for _, item := range items {
				secondSubsetMap[getMemAddr(item)] = item
			}
		}
	} else {
		for _, subaccountID := range secondFilter {
			if items, ok := secondMap[subaccountID]; ok {
				for _, item := range items {
					secondSubsetMap[getMemAddr(item)] = item
				}
			}
		}
	}

	// If both filters are empty, return empty slice
	if len(firstSubsetMap) == 0 && len(secondSubsetMap) == 0 {
		return
	}

	// If both filters are not empty, intersect maps
	if len(firstFilter) > 0 && len(secondFilter) > 0 {
		// Intersect maps
		for hash, position := range firstSubsetMap {
			if _, ok := secondSubsetMap[hash]; ok {
				outMap[hash] = position
			}
		}
	}

	// If one of the filters is empty, return the other subset map
	if len(firstFilter) > 0 && len(secondFilter) == 0 {
		for hash, position := range firstSubsetMap {
			outMap[hash] = position
		}
	}
	if len(secondFilter) > 0 && len(firstFilter) == 0 {
		for hash, position := range secondSubsetMap {
			outMap[hash] = position
		}
	}

	out = mapToSlice(outMap)
	return
}

func getMemAddr(i interface{}) string {
	return fmt.Sprintf("%p", i)
}
