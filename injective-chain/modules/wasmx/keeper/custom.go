package keeper

import (
	"encoding/json"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

func (k Keeper) InjectiveExec(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	funds sdk.Coins,
	msg *types.InjectiveExecMsg,
) ([]byte, error) {
	execBz, err := json.Marshal(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	res, err := k.wasmContractOpsKeeper.Execute(ctx, contractAddress, contractAddress, execBz, funds)
	if err != nil {
		k.logger.Debugln("result", res, "err", err)
		metrics.ReportFuncError(k.svcTags)
		return res, err
	}

	k.logger.Debugln("InjectiveExec result:", string(res))
	return res, nil
}

func (k Keeper) PinContract(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
) (err error) {

	contractInfo := k.wasmViewKeeper.GetContractInfo(ctx, contractAddress)
	err = k.wasmContractOpsKeeper.PinCode(ctx, contractInfo.CodeID)
	if err != nil {
		// Wasmer runtime error
		k.logger.Errorln("❌ Error while pinning the contract", err)
		return
	}

	k.logger.Debugln("✅ Pinned the contract successfully", contractAddress)
	return nil
}

func (k Keeper) UnpinContract(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
) (err error) {
	contractInfo := k.wasmViewKeeper.GetContractInfo(ctx, contractAddress)
	err = k.wasmContractOpsKeeper.UnpinCode(ctx, contractInfo.CodeID)
	if err != nil {
		// Wasmer runtime error
		k.logger.Errorln("❌ Error while unpinning the contract", err)
		return
	}

	k.logger.Debugln("✅ Unpinned the contract successfully", contractAddress)
	return nil
}
