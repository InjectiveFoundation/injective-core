package helpers

import (
	"encoding/base64"

	abcitypes "github.com/cometbft/cometbft/abci/types"
)

// From github.com/strangelove-ventures/interchaintest/v8/chain/internal/tendermint
// Cannot import directly because it's internal.
//
// cometAttributeValue returns an event attribute value given the eventType and attribute key tuple.
// In the event of duplicate types and keys, returns the first attribute value found.
// If not found, returns empty string and false.
func cometAttributeValue(events []abcitypes.Event, eventType, attrKey string) string {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key == attrKey {
				return attr.Value
			}

			// tendermint < v0.37-alpha returns base64 encoded strings in events.
			key, err := base64.StdEncoding.DecodeString(attr.Key)
			if err != nil {
				continue
			}
			if string(key) == attrKey {
				value, err := base64.StdEncoding.DecodeString(attr.Value)
				if err != nil {
					continue
				}
				return string(value)
			}
		}
	}
	return ""
}
