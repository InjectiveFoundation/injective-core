package types

import (
	"strings"

	"cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/common"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	tftypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

var (
	// DenomPrefix in parity with cosmos/evm and IBCv2 compat
	DenomPrefix = "erc20:"
)

type denomType int

const (
	DenomTypeUnspecified denomType = iota
	DenomTypeERC20
	DenomTypeTokenFactory
	DenomTypeIBC
	DenomTypePeggy
)

func (pair *TokenPair) Validate() error {
	switch GetDenomType(pair.BankDenom) {
	case DenomTypeTokenFactory:
		if _, _, err := tftypes.DeconstructDenom(pair.BankDenom); err != nil {
			return errors.Wrapf(ErrInvalidTokenPair, "token pair has invalid tokenfactory denom: %s", err.Error())
		}
	case DenomTypePeggy:
		if _, err := peggytypes.NewPeggyDenomFromString(pair.BankDenom); err != nil {
			return errors.Wrapf(ErrInvalidTokenPair, "token pair has invalid peggy denom: %s", err.Error())
		}
		if pair.Erc20Address != "" {
			return errors.Wrapf(ErrInvalidTokenPair, "peggy denom pair does not support existing ERC-20 contracts")
		}
	case DenomTypeIBC:
		if err := ibctransfertypes.ValidateIBCDenom(pair.BankDenom); err != nil {
			return errors.Wrapf(ErrInvalidTokenPair, "invalid ibc denom: %s", err.Error())
		}
		if pair.Erc20Address != "" {
			return errors.Wrapf(ErrInvalidTokenPair, "ibc denom pair does not support existing ERC-20 contracts")
		}
	default:
		return errors.Wrapf(ErrInvalidTokenPair, "token pair has unsupported bank denom: %s", pair.BankDenom)
	}
	if pair.Erc20Address != "" && !common.IsHexAddress(pair.Erc20Address) {
		return errors.Wrap(ErrInvalidTokenPair, "token pair has invalid ERC20 address")
	}

	return nil
}

func GetDenomType(bankDenom string) denomType {
	switch {
	case strings.HasPrefix(bankDenom, "ibc/"):
		return DenomTypeIBC
	case strings.HasPrefix(bankDenom, DenomPrefix):
		return DenomTypeERC20
	case strings.HasPrefix(bankDenom, "factory/"):
		return DenomTypeTokenFactory
	case strings.HasPrefix(bankDenom, "peggy"):
		return DenomTypePeggy
	default:
		return DenomTypeUnspecified
	}
}
