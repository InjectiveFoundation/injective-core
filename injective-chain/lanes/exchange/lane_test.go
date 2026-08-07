package exchangelane

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	protov2 "google.golang.org/protobuf/proto"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// fakeTx is a minimal sdk.Tx used to drive the exchange lane match handler.
type fakeTx struct{ msgs []sdk.Msg }

func (t fakeTx) GetMsgs() []sdk.Msg                    { return t.msgs }
func (t fakeTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

func TestIsExchangeMsg(t *testing.T) {
	cases := []struct {
		typeURL string
		want    bool
	}{
		{"/injective.exchange.v1beta1.MsgCreateSpotLimitOrder", true},
		{"/injective.exchange.v2.MsgCreateSpotLimitOrder", true},
		{"/injective.exchange.v2.MsgBatchUpdateOrders", true},
		// MsgPrivilegedExecuteContract is a privileged contract call, not an
		// exchange-lane message — it must route through the default lane.
		{"/injective.exchange.v1beta1.MsgPrivilegedExecuteContract", false},
		{"/injective.exchange.v2.MsgPrivilegedExecuteContract", false},
		// Non-exchange modules.
		{"/cosmwasm.wasm.v1.MsgExecuteContract", false},
		{"/cosmos.bank.v1beta1.MsgSend", false},
	}
	for _, tc := range cases {
		if got := isExchangeMsg(tc.typeURL); got != tc.want {
			t.Errorf("isExchangeMsg(%q) = %v, want %v", tc.typeURL, got, tc.want)
		}
	}
}

func TestHasOnlyExchangeMsgs(t *testing.T) {
	privV2, err := codectypes.NewAnyWithValue(&exchangev2.MsgPrivilegedExecuteContract{})
	if err != nil {
		t.Fatalf("pack any: %v", err)
	}

	cases := []struct {
		name string
		msg  sdk.Msg
		want bool
	}{
		{"spot order", &exchangev2.MsgCreateSpotLimitOrder{}, true},
		{"privileged execute v2", &exchangev2.MsgPrivilegedExecuteContract{}, false},
		{"privileged execute v1beta1", &exchangetypes.MsgPrivilegedExecuteContract{}, false},
		{"bank send", &banktypes.MsgSend{}, false},
		// authz.MsgExec wrapping a privileged execute must also be excluded.
		{"authz-wrapped privileged execute", &authz.MsgExec{Msgs: []*codectypes.Any{privV2}}, false},
	}
	for _, tc := range cases {
		if got := hasOnlyExchangeMsgs(tc.msg); got != tc.want {
			t.Errorf("%s: hasOnlyExchangeMsgs = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestExchangeMatchHandler(t *testing.T) {
	match := ExchangeMatchHandler()
	ctx := sdk.Context{}

	// A plain exchange order matches the exchange lane.
	if !match(ctx, fakeTx{msgs: []sdk.Msg{&exchangev2.MsgCreateSpotLimitOrder{}}}) {
		t.Fatal("expected an exchange order to match the exchange lane")
	}
	// A privileged-execute tx must NOT match — it belongs in the default lane.
	if match(ctx, fakeTx{msgs: []sdk.Msg{&exchangev2.MsgPrivilegedExecuteContract{}}}) {
		t.Fatal("MsgPrivilegedExecuteContract must not match the exchange lane")
	}
	// A tx mixing an exchange order with a privileged execute must NOT match.
	mixed := fakeTx{msgs: []sdk.Msg{&exchangev2.MsgCreateSpotLimitOrder{}, &exchangev2.MsgPrivilegedExecuteContract{}}}
	if match(ctx, mixed) {
		t.Fatal("a tx containing MsgPrivilegedExecuteContract must not match the exchange lane")
	}
}
