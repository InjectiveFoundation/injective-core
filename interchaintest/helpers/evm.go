package helpers

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// CallReadOnlyMethod calls a no-arg view/pure contract method and returns the result.
//
// Returns:
//   - any: single decoded value if one return item,
//     []any if multiple return values,
//     nil if no return values.
func CallEVMReadOnlyMethod(
	abi abi.ABI,
	client *ethclient.Client,
	contractAddr common.Address,
	methodName string,
) (any, error) {
	contract := bind.NewBoundContract(contractAddr, abi, client, client, client)

	var out []any

	err := contract.Call(
		&bind.CallOpts{
			Context: context.Background(),
		},
		&out,
		methodName,
	)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	switch len(out) {
	case 0:
		return nil, nil
	case 1:
		return out[0], nil
	default:
		result := make([]any, len(out))
		for i, v := range out {
			result[i] = v
		}
		return result, nil
	}
}
