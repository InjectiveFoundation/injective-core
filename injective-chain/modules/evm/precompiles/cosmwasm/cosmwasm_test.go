package cosmwasm

import (
	"bytes"
	"context"
	"math"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// mockWasmKeeper is a stand-in for wasmkeeper.Keeper covering the read-only surface the
// precompile uses. It lets us exercise runQuery without a full app + store.
type mockWasmKeeper struct {
	has        bool
	gasLimit   storetypes.Gas
	querySmart func(ctx context.Context, addr sdk.AccAddress, req []byte) ([]byte, error)
}

func (m mockWasmKeeper) HasContractInfo(_ context.Context, _ sdk.AccAddress) bool { return m.has }
func (m mockWasmKeeper) QueryGasLimit() storetypes.Gas                            { return m.gasLimit }
func (m mockWasmKeeper) QuerySmart(ctx context.Context, addr sdk.AccAddress, req []byte) ([]byte, error) {
	return m.querySmart(ctx, addr, req)
}

func newTestCtx() sdk.Context {
	return sdk.Context{}.WithGasMeter(storetypes.NewGasMeter(10_000_000))
}

func bech32Of(b byte) string {
	return sdk.AccAddress(bytes.Repeat([]byte{b}, 20)).String()
}

func TestMetadata(t *testing.T) {
	c := NewContract(mockWasmKeeper{}, storetypes.TransientGasConfig())
	require.Equal(t, common.HexToAddress("0x0000000000000000000000000000000000000068"), c.Address())
	require.Contains(t, cosmwasmABI.Methods, QueryMethodName)
	require.Len(t, cosmwasmABI.Methods, 1, "precompile exposes only the generic query method")
}

func TestRequiredGas(t *testing.T) {
	c := &Contract{wasmKeeper: mockWasmKeeper{}, kvGasConfig: storetypes.TransientGasConfig()}

	require.Equal(t, uint64(0), c.RequiredGas([]byte{0x01}), "sub-selector input is free")

	oversized := make([]byte, 10_001)
	require.Equal(t, uint64(math.MaxUint64), c.RequiredGas(oversized), "oversized calldata is rejected")

	sel := cosmwasmABI.Methods[QueryMethodName].ID[:4]
	input := append(append([]byte{}, sel...), bytes.Repeat([]byte{0x00}, 64)...)
	want := QueryBaseGas + uint64(len(input))*storetypes.TransientGasConfig().WriteCostPerByte
	require.Equal(t, want, c.RequiredGas(input))
}

func TestRunQuery_HappyPath(t *testing.T) {
	expected := []byte(`{"count":42}`)
	c := &Contract{
		wasmKeeper: mockWasmKeeper{
			has:      true,
			gasLimit: 3_000_000,
			querySmart: func(_ context.Context, _ sdk.AccAddress, req []byte) ([]byte, error) {
				require.Equal(t, `{"count":{}}`, string(req))
				return expected, nil
			},
		},
		kvGasConfig: storetypes.TransientGasConfig(),
	}
	out, err := c.runQuery(newTestCtx(), bech32Of(0x01), []byte(`{"count":{}}`))
	require.NoError(t, err)
	require.Equal(t, expected, out)
}

func TestRunQuery_ContractNotFound(t *testing.T) {
	c := &Contract{
		wasmKeeper:  mockWasmKeeper{has: false, gasLimit: 3_000_000},
		kvGasConfig: storetypes.TransientGasConfig(),
	}
	_, err := c.runQuery(newTestCtx(), bech32Of(0x01), []byte(`{}`))
	require.ErrorIs(t, err, ErrContractNotFound)
}

func TestRunQuery_InvalidAddress(t *testing.T) {
	c := &Contract{
		wasmKeeper:  mockWasmKeeper{has: true, gasLimit: 3_000_000},
		kvGasConfig: storetypes.TransientGasConfig(),
	}
	_, err := c.runQuery(newTestCtx(), "not-a-bech32-address", []byte(`{}`))
	require.Error(t, err)
}

func TestRunQuery_OutOfGasRecovered(t *testing.T) {
	c := &Contract{
		wasmKeeper: mockWasmKeeper{
			has:      true,
			gasLimit: 1_000_000,
			querySmart: func(ctx context.Context, _ sdk.AccAddress, _ []byte) ([]byte, error) {
				// Simulate an expensive query overrunning the installed gas meter.
				sdk.UnwrapSDKContext(ctx).GasMeter().ConsumeGas(2_000_000, "boom")
				return []byte("unreached"), nil
			},
		},
		kvGasConfig: storetypes.TransientGasConfig(),
	}
	_, err := c.runQuery(newTestCtx(), bech32Of(0x01), []byte(`{}`))
	require.ErrorIs(t, err, ErrQueryOutOfGas, "wasm OOG must be recovered into a clean error, not a panic")
}
