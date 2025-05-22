package types

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func CastAddress(input interface{}) (sdk.AccAddress, error) {
	ethAddr, ok := input.(common.Address)
	if !ok {
		return sdk.AccAddress{}, errors.New("could not cast input to address")
	}
	return sdk.AccAddress(ethAddr.Bytes()), nil
}

func CastString(input interface{}) (string, error) {
	res, ok := input.(string)
	if !ok {
		return "", errors.New("could not cast input to string")
	}
	return res, nil
}

func CastStringArray(input interface{}) ([]string, error) {
	res, ok := input.([]string)
	if !ok {
		return nil, errors.New("could not cast input to string array")
	}
	return res, nil
}

func CastBigInt(input interface{}) (*big.Int, error) {
	res, ok := input.(*big.Int)
	if !ok {
		return nil, errors.New("could not cast input to big.Int")
	}
	return res, nil
}

func CastUint32(input interface{}) (uint32, error) {
	res, ok := input.(uint32)
	if !ok {
		return 0, errors.New("could not cast input to uint32")
	}
	return res, nil
}

func CastInt32(input interface{}) (int32, error) {
	res, ok := input.(int32)
	if !ok {
		return 0, errors.New("could not cast input to int32")
	}
	return res, nil
}

// ConvertLegacyDecToBigInt removes the scaling factor from the LegacyDec
func ConvertLegacyDecToBigInt(in sdkmath.LegacyDec) *big.Int {
	return in.RoundInt().BigInt()
}
