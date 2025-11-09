package staking

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/bindings/cosmos/precompile/staking"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/types"
)

const (
	DelegateMethodName   = "delegate"
	UndelegateMethodName = "undelegate"
	RedelegateMethodName = "redelegate"
	DelegationQueryName  = "delegation"
	WithdrawRewardsName  = "withdrawDelegatorRewards"
)

var (
	stakingABI                 abi.ABI
	stakingContractAddress     = common.BytesToAddress([]byte{102})
	stakingGasRequiredByMethod = map[[4]byte]uint64{}
)

var (
	ErrPrecompilePanic = errors.New("precompile panic")
)

func init() {
	if err := stakingABI.UnmarshalJSON([]byte(staking.StakingModuleMetaData.ABI)); err != nil {
		panic(err)
	}
	for methodName := range stakingABI.Methods {
		var methodID [4]byte
		copy(methodID[:], stakingABI.Methods[methodName].ID[:4])
		switch methodName {
		case DelegateMethodName:
			stakingGasRequiredByMethod[methodID] = 200_000
		case UndelegateMethodName:
			stakingGasRequiredByMethod[methodID] = 200_000
		case RedelegateMethodName:
			stakingGasRequiredByMethod[methodID] = 200_000
		case WithdrawRewardsName:
			stakingGasRequiredByMethod[methodID] = 200_000
		default:
			stakingGasRequiredByMethod[methodID] = 0
		}
	}
}

type StakingContract struct {
	stakingKeeper         *stakingkeeper.Keeper
	stakingMsgServer      stakingtypes.MsgServer
	stakingQueryServer    stakingtypes.QueryServer
	distributionKeeper    distributionkeeper.Keeper
	distributionMsgServer distributiontypes.MsgServer
	kvGasConfig           storetypes.GasConfig
}

func NewStakingContract(
	stakingKeeper *stakingkeeper.Keeper,
	distributionKeeper distributionkeeper.Keeper,
	kvGasConfig storetypes.GasConfig,
) vm.PrecompiledContract {
	return &StakingContract{
		stakingKeeper:         stakingKeeper,
		stakingMsgServer:      stakingkeeper.NewMsgServerImpl(stakingKeeper),
		stakingQueryServer:    stakingkeeper.NewQuerier(stakingKeeper),
		distributionKeeper:    distributionKeeper,
		distributionMsgServer: distributionkeeper.NewMsgServerImpl(distributionKeeper),
		kvGasConfig:           kvGasConfig,
	}
}

func (sc *StakingContract) ABI() abi.ABI {
	return stakingABI
}

func (sc *StakingContract) Address() common.Address {
	return stakingContractAddress
}

func (*StakingContract) Name() string {
	return "INJ_STAKING"
}

func (sc *StakingContract) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	// base cost to prevent large input size
	baseCost := uint64(len(input)) * sc.kvGasConfig.WriteCostPerByte
	var methodID [4]byte
	copy(methodID[:], input[:4])
	requiredGas, ok := stakingGasRequiredByMethod[methodID]
	if ok {
		return requiredGas + baseCost
	}
	return baseCost
}

func (sc *StakingContract) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	res, err := sc.run(evm, contract, readonly)
	if err != nil {
		return types.RevertReasonAndError(err)
	}
	return res, nil
}

func (sc *StakingContract) run(evm *vm.EVM, contract *vm.Contract, readonly bool) (output []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrPrecompilePanic
			output = nil
		}
	}()

	// parse input
	methodID := contract.Input[:4]
	method, err := stakingABI.MethodById(methodID)
	if err != nil {
		return nil, err
	}

	args, err := method.Inputs.Unpack(contract.Input[4:])
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}

	caller := sdk.AccAddress(contract.Caller().Bytes())

	switch method.Name {
	case DelegateMethodName:
		return sc.delegate(evm, caller, method, args, readonly)
	case UndelegateMethodName:
		return sc.undelegate(evm, caller, method, args, readonly)
	case RedelegateMethodName:
		return sc.redelegate(evm, caller, method, args, readonly)
	case DelegationQueryName:
		return sc.queryDelegation(evm, caller, method, args, readonly)
	case WithdrawRewardsName:
		return sc.withdrawDelegatorRewards(evm, caller, method, args, readonly)
	default:
		return nil, errors.New("unknown method")
	}
}

func (sc *StakingContract) delegate(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	validator, err := types.CastString(args[0])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[1])
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB.(precompiles.ExtStateDB)
	context := stateDB.Context()

	bondDenom, err := sc.stakingKeeper.BondDenom(context)
	if err != nil {
		return nil, err
	}

	msg := &stakingtypes.MsgDelegate{
		DelegatorAddress: caller.String(),
		ValidatorAddress: validator,
		Amount: sdk.NewCoin(
			bondDenom,
			sdkmath.NewIntFromBigInt(amount),
		),
	}

	err = sc.executeNativeAction(
		evm,
		func(ctx sdk.Context) error {
			_, err := sc.stakingMsgServer.Delegate(ctx, msg)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (sc *StakingContract) undelegate(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	validator, err := types.CastString(args[0])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[1])
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB.(precompiles.ExtStateDB)
	context := stateDB.Context()

	bondDenom, err := sc.stakingKeeper.BondDenom(context)
	if err != nil {
		return nil, err
	}

	msg := &stakingtypes.MsgUndelegate{
		DelegatorAddress: caller.String(),
		ValidatorAddress: validator,
		Amount: sdk.NewCoin(
			bondDenom,
			sdkmath.NewIntFromBigInt(amount),
		),
	}

	err = sc.executeNativeAction(
		evm,
		func(ctx sdk.Context) error {
			_, err := sc.stakingMsgServer.Undelegate(ctx, msg)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (sc *StakingContract) redelegate(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	validatorSrc, err := types.CastString(args[0])
	if err != nil {
		return nil, err
	}
	validatorDst, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}
	amount, err := types.CastBigInt(args[2])
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB.(precompiles.ExtStateDB)
	context := stateDB.Context()

	bondDenom, err := sc.stakingKeeper.BondDenom(context)
	if err != nil {
		return nil, err
	}

	msg := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    caller.String(),
		ValidatorSrcAddress: validatorSrc,
		ValidatorDstAddress: validatorDst,
		Amount: sdk.NewCoin(
			bondDenom,
			sdkmath.NewIntFromBigInt(amount),
		),
	}

	err = sc.executeNativeAction(
		evm,
		func(ctx sdk.Context) error {
			_, err := sc.stakingMsgServer.BeginRedelegate(ctx, msg)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (sc *StakingContract) queryDelegation(
	evm *vm.EVM,
	_ sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	_ bool,
) ([]byte, error) {
	delegator, err := types.CastAddress(args[0])
	if err != nil {
		return nil, err
	}
	validator, err := types.CastString(args[1])
	if err != nil {
		return nil, err
	}

	req := &stakingtypes.QueryDelegationRequest{
		DelegatorAddr: delegator.String(),
		ValidatorAddr: validator,
	}

	var resp *stakingtypes.QueryDelegationResponse
	err = sc.executeNativeAction(
		evm,
		func(ctx sdk.Context) (err error) {
			resp, err = sc.stakingQueryServer.Delegation(ctx, req)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(
		types.ConvertLegacyDecToBigInt(resp.DelegationResponse.Delegation.Shares),
		staking.CosmosCoin{
			Denom:  "inj",
			Amount: resp.DelegationResponse.Balance.Amount.BigInt(),
		},
	)
}

func (sc *StakingContract) withdrawDelegatorRewards(
	evm *vm.EVM,
	caller sdk.AccAddress,
	method *abi.Method,
	args []interface{},
	readonly bool,
) ([]byte, error) {
	if readonly {
		return nil, errors.New("the method is not readonly")
	}

	validator, err := types.CastString(args[0])
	if err != nil {
		return nil, err
	}

	msg := &distributiontypes.MsgWithdrawDelegatorReward{
		DelegatorAddress: caller.String(),
		ValidatorAddress: validator,
	}

	var resp *distributiontypes.MsgWithdrawDelegatorRewardResponse
	err = sc.executeNativeAction(
		evm,
		func(ctx sdk.Context) error {
			var err error
			resp, err = sc.distributionMsgServer.WithdrawDelegatorReward(ctx, msg)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	coins := []staking.CosmosCoin{}
	for _, coin := range resp.Amount {
		coins = append(coins, staking.CosmosCoin{
			Denom:  coin.Denom,
			Amount: coin.Amount.BigInt(),
		})
	}

	return method.Outputs.Pack(coins)
}

/******************************************************************************/

func (sc *StakingContract) executeNativeAction(evm *vm.EVM, action func(ctx sdk.Context) error) error {
	stateDB := evm.StateDB.(precompiles.ExtStateDB)
	return stateDB.ExecuteNativeAction(
		sc.Address(),
		nil,
		action,
	)
}
