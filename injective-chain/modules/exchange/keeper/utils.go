package keeper

import (
	"io"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ethereum/go-ethereum/common"
)

func CheckIfExceedDecimals(dec sdk.Dec, maxDecimals uint32) bool {
	powered := dec.Mul(sdk.NewDec(10).Power(uint64(maxDecimals)))
	return !powered.Equal(powered.Ceil())
}

// GetIsOrderLess returns true if the order is less than the other order
func GetIsOrderLess(referencePrice, order1Price, order2Price sdk.Dec, order1IsBuy, order2IsBuy, isSortingFromWorstToBest bool) bool {
	var firstDistanceToReferencePrice, secondDistanceToReferencePrice sdk.Dec

	if order1IsBuy {
		firstDistanceToReferencePrice = referencePrice.Sub(order1Price)
	} else {
		firstDistanceToReferencePrice = order1Price.Sub(referencePrice)
	}

	if order2IsBuy {
		secondDistanceToReferencePrice = referencePrice.Sub(order2Price)
	} else {
		secondDistanceToReferencePrice = order2Price.Sub(referencePrice)
	}

	if isSortingFromWorstToBest {
		return firstDistanceToReferencePrice.GT(secondDistanceToReferencePrice)
	}

	return firstDistanceToReferencePrice.LT(secondDistanceToReferencePrice)
}

func (k *Keeper) checkIfMarketLaunchProposalExist(
	ctx sdk.Context,
	proposalType string,
	marketID common.Hash,
) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	exists := false
	params := k.govKeeper.GetParams(ctx)
	// Note: we do 10 * voting period to iterate all active proposals safely
	endTime := ctx.BlockTime().Add(10 * (*params.VotingPeriod))

	k.govKeeper.IterateActiveProposalsQueue(ctx, endTime, func(proposal v1.Proposal) bool {
		found := proposalAlreadyExists(proposal, proposalType, marketID)

		exists = found
		return found
	})

	return exists
}

func proposalAlreadyExists(prop v1.Proposal, proposalType string, marketID common.Hash) bool {
	msgs, err := tx.GetMsgs(prop.Messages, "proposal")
	if err != nil {
		return false
	}

	for _, msg := range msgs {
		if legacyMsg, ok := msg.(*v1.MsgExecLegacyContent); ok {
			//	1. msg is legacy
			content, err := v1.LegacyContentFromMessage(legacyMsg)
			if err != nil {
				continue
			}
			if content.ProposalType() == proposalType {
				switch proposalType {
				case types.ProposalTypeExpiryFuturesMarketLaunch:
					p := content.(*types.ExpiryFuturesMarketLaunchProposal)
					if marketID == types.NewExpiryFuturesMarketID(p.Ticker, p.QuoteDenom, p.OracleBase, p.OracleQuote, p.OracleType, p.Expiry) {
						return true
					}
				case types.ProposalTypePerpetualMarketLaunch:
					p := content.(*types.PerpetualMarketLaunchProposal)
					if marketID == types.NewPerpetualMarketID(p.Ticker, p.QuoteDenom, p.OracleBase, p.OracleQuote, p.OracleType) {
						return true
					}
				case types.ProposalTypeBinaryOptionsMarketLaunch:
					p := content.(*types.BinaryOptionsMarketLaunchProposal)
					if marketID == types.NewBinaryOptionsMarketID(p.Ticker, p.QuoteDenom, p.OracleSymbol, p.OracleProvider, p.OracleType) {
						return true
					}
				case types.ProposalTypeSpotMarketLaunch:
					p := content.(*types.SpotMarketLaunchProposal)
					if marketID == types.NewSpotMarketID(p.BaseDenom, p.QuoteDenom) {
						return true
					}
				}
			}

		}
	}

	return false
}

// getReadableDec is a test utility function to return a readable representation of decimal strings
func getReadableDec(d sdk.Dec) string {
	if d.IsNil() {
		return d.String()
	}
	dec := strings.TrimRight(d.String(), "0")
	if len(dec) < 2 {
		return dec
	}

	if dec[len(dec)-1:] == "." {
		return dec + "0"
	}
	return dec
}

func ReadFile(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	return b
}

// GetReadableSlice is a test utility function to return a readable representation of any arbitrary slice, by applying formatter function to each slice element
func GetReadableSlice[T any](slice []T, sep string, formatter func(T) string) string {
	stringsArr := make([]string, len(slice))
	for i, t := range slice {
		stringsArr[i] = formatter(t)
	}
	return strings.Join(stringsArr, sep)
}

// reverseSlice will reverse slice contents (in place)
func ReverseSlice[T any](slice []T) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func Count[T any](slice []T, predicate func(T) bool) int {
	var result = 0
	for _, v := range slice {
		if predicate(v) {
			result++
		}
	}
	return result
}

func FindFirst[T any](slice []*T, predicate func(*T) bool) *T {
	for _, v := range slice {
		if predicate(v) {
			return v
		}
	}
	return nil
}

func FilterNotNull[T any](slice []*T) []*T {
	filteredSlice := make([]*T, 0)
	for _, v := range slice {
		if v != nil {
			filteredSlice = append(filteredSlice, v)
		}
	}
	return filteredSlice
}

func SingleElementSlice[T any](element T) []T {
	slice := make([]T, 1)
	slice[0] = element
	return slice
}

// SubtractBitFromPrefix returns a prev prefix. It is calculated by subtracting 1 bit from the start value. Nil is not allowed as prefix.
//
//	Example: []byte{1, 3, 4} becomes []byte{1, 3, 5}
//			 []byte{15, 42, 255, 255} becomes []byte{15, 43, 0, 0}
//
// In case of an overflow the end is set to nil.
//
//	Example: []byte{255, 255, 255, 255} becomes nil
//
// MARK finish-batches: this is where some crazy shit happens
func SubtractBitFromPrefix(prefix []byte) []byte {
	if prefix == nil {
		panic("nil key not allowed")
	}

	// special case: no prefix is whole range
	if len(prefix) == 0 {
		return nil
	}

	// copy the prefix and update last byte
	newPrefix := make([]byte, len(prefix))
	copy(newPrefix, prefix)
	l := len(newPrefix) - 1
	newPrefix[l]--

	// wait, what if that overflowed?....
	for newPrefix[l] == 255 && l > 0 {
		l--
		newPrefix[l]--
	}

	// okay, funny guy, you gave us FFF, no end to this range...
	if l == 0 && newPrefix[0] == 255 {
		newPrefix = nil
	}

	return newPrefix
}

// AddBitToPrefix returns a prefix calculated by adding 1 bit to the start value. Nil is not allowed as prefix.
//
//	Example: []byte{1, 3, 4} becomes []byte{1, 3, 5}
//			 []byte{15, 42, 255, 255} becomes []byte{15, 43, 0, 0}
//
// In case of an overflow the end is set to nil.
//
//	Example: []byte{255, 255, 255, 255} becomes nil
func AddBitToPrefix(prefix []byte) []byte {
	if prefix == nil {
		panic("nil key not allowed")
	}

	// special case: no prefix is whole range
	if len(prefix) == 0 {
		return nil
	}

	// copy the prefix and update last byte
	newPrefix := make([]byte, len(prefix))
	copy(newPrefix, prefix)
	l := len(newPrefix) - 1
	newPrefix[l]++

	// wait, what if that overflowed?....
	for newPrefix[l] == 0 && l > 0 {
		l--
		newPrefix[l]++
	}

	// okay, funny guy, you gave us FFF, no end to this range...
	if l == 0 && newPrefix[0] == 0 {
		newPrefix = nil
	}

	return newPrefix
}
