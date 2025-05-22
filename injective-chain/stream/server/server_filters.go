package server

import (
	"errors"
	"fmt"

	v2 "github.com/InjectiveLabs/injective-core/injective-chain/stream/types/v2"
)

var ErrInvalidParameters = errors.New("firstMap and secondMap must have the same length")

func Filter[V v2.OrderbookUpdate | v2.BankBalance | v2.OraclePrice | v2.SubaccountDeposits](
	itemMap map[string][]*V, filter []string,
) (out []*V) {
	wildcard := false
	if len(filter) > 0 {
		wildcard = filter[0] == "*"
	}
	if wildcard {
		for _, items := range itemMap {
			out = append(out, items...)
		}
		return out
	}
	for _, marketID := range filter {
		if updates, ok := itemMap[marketID]; ok {
			out = append(out, updates...)
		}
	}
	return out
}

func FilterMulti[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	firstMap, secondMap map[string][]*V, firstFilter, secondFilter []string,
) (out []*V, err error) {
	// Check early return conditions
	shouldReturn, returnErr := checkEarlyReturnConditions(firstMap, secondMap, firstFilter, secondFilter)
	if shouldReturn {
		return nil, returnErr
	}

	firstSubsetMap := buildFilteredMap(firstMap, firstFilter)
	secondSubsetMap := buildFilteredMap(secondMap, secondFilter)

	// If both subset maps are empty, return empty slice
	if hasNoResults(firstSubsetMap, secondSubsetMap) {
		return nil, nil
	}

	outMap := combineSubsetMaps(firstSubsetMap, secondSubsetMap, firstFilter, secondFilter)
	out = mapToSlice(outMap)
	return out, nil
}

func checkEarlyReturnConditions[V any](firstMap, secondMap map[string][]*V, firstFilter, secondFilter []string) (bool, error) {
	// Early returns for empty cases
	noFilters := len(firstFilter) == 0 && len(secondFilter) == 0
	noData := len(firstMap) == 0 && len(secondMap) == 0
	if noFilters || noData {
		return true, nil
	}

	invalidParams := len(firstMap) == 0 || len(secondMap) == 0
	if invalidParams {
		return true, ErrInvalidParameters
	}

	return false, nil
}

func isWildcard(filter []string) bool {
	return len(filter) > 0 && filter[0] == "*"
}

func buildFilteredMap[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	itemMap map[string][]*V, filter []string,
) map[string]*V {
	if isWildcard(filter) {
		return buildWildcardMap(itemMap)
	}
	return buildSpecificMap(itemMap, filter)
}

func buildWildcardMap[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	itemMap map[string][]*V,
) map[string]*V {
	subsetMap := make(map[string]*V)
	for _, items := range itemMap {
		for _, item := range items {
			subsetMap[getMemAddr(item)] = item
		}
	}
	return subsetMap
}

func buildSpecificMap[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	itemMap map[string][]*V, filter []string,
) map[string]*V {
	subsetMap := make(map[string]*V)
	for _, id := range filter {
		if items, ok := itemMap[id]; ok {
			for _, item := range items {
				subsetMap[getMemAddr(item)] = item
			}
		}
	}
	return subsetMap
}

func combineSubsetMaps[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	firstSubsetMap, secondSubsetMap map[string]*V, firstFilter, secondFilter []string,
) map[string]*V {
	outMap := make(map[string]*V)

	hasFirstFilter := len(firstFilter) > 0
	hasSecondFilter := len(secondFilter) > 0

	switch {
	case hasFirstFilter && hasSecondFilter:
		return intersectMaps(firstSubsetMap, secondSubsetMap)
	case hasFirstFilter:
		return copyMap(firstSubsetMap)
	case hasSecondFilter:
		return copyMap(secondSubsetMap)
	}

	return outMap
}

func intersectMaps[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	firstMap, secondMap map[string]*V,
) map[string]*V {
	result := make(map[string]*V)
	for hash, item := range firstMap {
		if _, ok := secondMap[hash]; ok {
			result[hash] = item
		}
	}
	return result
}

func copyMap[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	sourceMap map[string]*V,
) map[string]*V {
	result := make(map[string]*V)
	for hash, item := range sourceMap {
		result[hash] = item
	}
	return result
}

func mapToSlice[V v2.Position | v2.SpotTrade | v2.DerivativeTrade | v2.DerivativeOrderUpdate | v2.SpotOrderUpdate](
	m map[string]*V,
) []*V {
	outSlice := make([]*V, len(m))
	i := 0
	for _, position := range m {
		outSlice[i] = position
		i++
	}
	return outSlice
}

func getMemAddr(i interface{}) string {
	return fmt.Sprintf("%p", i)
}

func hasNoResults[V any](firstMap, secondMap map[string]*V) bool {
	return len(firstMap) == 0 && len(secondMap) == 0
}
