package cosmwasm

import (
	"context"
	"errors"
	"math"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles"
	bindings "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/bindings/cosmos/precompile/cosmwasm"
	precomptypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/types"
)

const (
	// QueryMethodName is the generic read-only smart query:
	//   query(string contractAddress, bytes req) view returns (bytes response)
	QueryMethodName = "query"

	// QueryBaseGas is the flat EVM gas charged per call (plus a per-byte calldata cost).
	//
	// This is NOT the safety bound: actual wasm compute is hard-capped per call by the wasm module's
	// QueryGasLimit (see runQuery) — the same limit wasmd already exposes to the public internet for
	// free via gRPC — so any positive fee is already stricter than the status quo and there is no DoS
	// to price against. It is therefore a spam-deterrent / consistency knob, set to align with the
	// heaviest existing precompile methods (cf. exchange's ~200-250k). Tunable; a candidate for a
	// governance param.
	QueryBaseGas uint64 = 200_000
)

var (
	cosmwasmABI             abi.ABI
	cosmwasmContractAddress = common.HexToAddress("0x0000000000000000000000000000000000000068")
	gasRequiredByMethod     = map[[4]byte]uint64{}
)

var (
	ErrPrecompilePanic  = errors.New("cosmwasm precompile panic")
	ErrContractNotFound = errors.New("cosmwasm contract not found")
	ErrQueryOutOfGas    = errors.New("cosmwasm query out of gas")
)

func init() {
	if err := cosmwasmABI.UnmarshalJSON([]byte(bindings.CosmwasmModuleMetaData.ABI)); err != nil {
		panic(err)
	}
	for name, gas := range map[string]uint64{
		QueryMethodName: QueryBaseGas,
	} {
		method, ok := cosmwasmABI.Methods[name]
		if !ok {
			panic(name + " method not found in ABI")
		}
		var id [4]byte
		copy(id[:], method.ID[:4])
		gasRequiredByMethod[id] = gas
	}
}

// WasmViewKeeper is the read-only subset of wasmkeeper.Keeper this precompile needs.
// *wasmkeeper.Keeper satisfies it (all methods have value receivers).
type WasmViewKeeper interface {
	QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
	HasContractInfo(ctx context.Context, contractAddr sdk.AccAddress) bool
	QueryGasLimit() storetypes.Gas
}

type Contract struct {
	wasmKeeper  WasmViewKeeper
	kvGasConfig storetypes.GasConfig
}

func NewContract(wasmKeeper WasmViewKeeper, kvGasConfig storetypes.GasConfig) vm.PrecompiledContract {
	return &Contract{
		wasmKeeper:  wasmKeeper,
		kvGasConfig: kvGasConfig,
	}
}

func (*Contract) ABI() abi.ABI            { return cosmwasmABI }
func (*Contract) Address() common.Address { return cosmwasmContractAddress }
func (*Contract) Name() string            { return "INJ_COSMWASM" }

func (c *Contract) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	// Reject oversized calldata before ABI decoding.
	if len(input) > precomptypes.MAX_ABI_ENCODED_CALLDATA_LENGTH {
		return math.MaxUint64
	}

	baseCost := uint64(len(input)) * c.kvGasConfig.WriteCostPerByte
	var methodID [4]byte
	copy(methodID[:], input[:4])
	if methodGas, ok := gasRequiredByMethod[methodID]; ok {
		return methodGas + baseCost
	}
	return baseCost
}

func (c *Contract) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	res, err := c.execute(evm, contract, readonly)
	if err != nil {
		return precomptypes.RevertReasonAndError(err)
	}
	return res, nil
}

func (c *Contract) execute(evm *vm.EVM, contract *vm.Contract, _ bool) (output []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errorsmod.Wrapf(ErrPrecompilePanic, "%v", r)
			output = nil
		}
	}()

	method, err := cosmwasmABI.MethodById(contract.Input[:4])
	if err != nil {
		return nil, err
	}

	stateDB, ok := evm.StateDB.(precompiles.ExtStateDB)
	if !ok {
		return nil, errors.New("state DB does not implement ExtStateDB")
	}
	defer func(origCtx sdk.Context) { *stateDB.ContextPtr() = origCtx }(stateDB.Context())
	defer stateDB.Meter().FuncTiming(stateDB.ContextPtr(), "execute",
		metrics.Tag("svc", "cosmwasmpc"), metrics.Tag("method", method.Name))()

	args, err := method.Inputs.Unpack(contract.Input[4:])
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}

	// Read-only: run against a cache context; nothing is committed back to state.
	ctx := stateDB.CacheContext()
	switch method.Name {
	case QueryMethodName:
		return c.query(ctx, method, args)
	}
	return nil, errors.New("unknown method")
}

// query(string contractAddress, bytes req) view returns (bytes response)
func (c *Contract) query(ctx sdk.Context, m *abi.Method, args []any) ([]byte, error) {
	addrStr, err := precomptypes.CastString(args[0])
	if err != nil {
		return nil, err
	}
	req, ok := args[1].([]byte)
	if !ok {
		return nil, errors.New("could not cast req to bytes")
	}
	res, err := c.runQuery(ctx, addrStr, req)
	if err != nil {
		return nil, err
	}
	return m.Outputs.Pack(res)
}

// runQuery bounds and executes a read-only smart query. The EVM executes under an infinite
// Cosmos gas meter, so we install a fresh bounded meter sized to the wasm module's QueryGasLimit
// — exactly what wasmd's own GrpcQuerier does — otherwise the query would be unbounded. A wasm
// out-of-gas surfaces as a Cosmos ErrorOutOfGas panic, which we recover into a clean revert so it
// can never escape and halt the chain.
func (c *Contract) runQuery(ctx sdk.Context, addrStr string, req []byte) (out []byte, err error) {
	contractAddr, err := sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		return nil, err
	}
	if !c.wasmKeeper.HasContractInfo(ctx, contractAddr) {
		return nil, ErrContractNotFound
	}

	qCtx := ctx.WithGasMeter(storetypes.NewGasMeter(c.wasmKeeper.QueryGasLimit()))
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(storetypes.ErrorOutOfGas); ok {
				out, err = nil, ErrQueryOutOfGas
				return
			}
			panic(r) // real panic: re-raise to the outer recover in execute()
		}
	}()

	return c.wasmKeeper.QuerySmart(qCtx, contractAddr, req)
}
