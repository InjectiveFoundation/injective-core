package simulation

import (
	"bytes"
	"fmt"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
	"github.com/ethereum/go-ethereum/common"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// value to the corresponding EVM type.
func NewDecodeStore() func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:1], types.KeyPrefixStorage):
			storageA := common.BytesToHash(kvA.Value).Hex()
			storageB := common.BytesToHash(kvB.Value).Hex()

			return fmt.Sprintf("%v\n%v", storageA, storageB)
		case bytes.Equal(kvA.Key[:1], types.KeyPrefixCode):
			codeHashA := common.Bytes2Hex(kvA.Value)
			codeHashB := common.Bytes2Hex(kvB.Value)

			return fmt.Sprintf("%v\n%v", codeHashA, codeHashB)
		default:
			panic(fmt.Sprintf("invalid evm key prefix %X", kvA.Key[:1]))
		}
	}
}
