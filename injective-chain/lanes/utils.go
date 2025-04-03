package lanes

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/v2/block/utils"
)

func CheckTxGasLimit(
	ctx sdk.Context,
	tx sdk.Tx,
	maxTxGas uint64,
	txInfoGetter func(ctx sdk.Context, tx sdk.Tx) (utils.TxWithInfo, error),
) (bool, error) {
	txInfo, err := txInfoGetter(ctx, tx)
	if err != nil {
		return false, fmt.Errorf("error getting TxInfo: %w", err)
	}

	if txInfo.GasLimit > maxTxGas {
		return false, errors.New("tx gas limit is greater than max tx gas limit for lane")
	}

	return true, nil
}
