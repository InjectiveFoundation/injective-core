package exchangelane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/skip-mev/block-sdk/v2/block"
	skipbase "github.com/skip-mev/block-sdk/v2/block/base"

	injcodectypes "github.com/InjectiveLabs/injective-core/injective-chain/codec/types"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "exchange"
)

func init() {
	encodingConfig := injcodectypes.MakeEncodingConfig()
	authz.RegisterInterfaces(encodingConfig.InterfaceRegistry)
}

// WithCustomMempool returns a LaneOption that sets a custom mempool.
func WithCustomMempool(mempool block.LaneMempool) skipbase.LaneOption {
	return func(l *skipbase.BaseLane) {
		l.LaneMempool = mempool
	}
}

func isExchangeMsg(msgTypeURL string) bool {
	moduleName := sdk.GetModuleNameFromTypeURL(msgTypeURL)
	return moduleName == "exchange"
}

func hasExchangeMsg(msg sdk.Msg) bool {
	if isExchangeMsg(sdk.MsgTypeURL(msg)) {
		return true
	}

	// If the message is an authz.MsgExec, check its inner messages.
	if msgExec, ok := msg.(*authz.MsgExec); ok {
		for _, innerMsg := range msgExec.Msgs {
			if isExchangeMsg(innerMsg.TypeUrl) {
				return true
			}
		}
	}
	return false
}

// ExchangeMatchHandler returns the exchange match handler for the exchange lane.
func ExchangeMatchHandler() skipbase.MatchHandler {
	return func(_ sdk.Context, tx sdk.Tx) bool {
		for _, msg := range tx.GetMsgs() {
			if hasExchangeMsg(msg) {
				return true
			}
		}
		return false
	}
}

// NewExchangeLane returns a new exchange lane. The exchange lane orders
// transactions by the transaction fees. It accepts any transaction, including
// authz transactions, that contains at least one exchange message. The exchange
// lane builds and verifies blocks in a similar fashion to how the CometBFT/Tendermint
// consensus engine builds and verifies blocks pre SDK version 0.47.0.
func NewExchangeLane(
	exchangeKeeper *exchangekeeper.Keeper,
	cfg skipbase.LaneConfig,
) *skipbase.BaseLane {
	exchangeTxPriority := ExchangeTxPriority{
		ExchangeKeeper:  exchangeKeeper,
		SignerExtractor: cfg.SignerExtractor,
	}
	customMempool := skipbase.NewMempool(
		exchangeTxPriority.ExchangeTxPriority(),
		cfg.SignerExtractor,
		cfg.MaxTxs,
	)
	options := []skipbase.LaneOption{
		skipbase.WithMatchHandler(ExchangeMatchHandler()),
		WithCustomMempool(customMempool),
	}

	lane, err := skipbase.NewBaseLane(
		cfg,
		LaneName,
		options...,
	)
	if err != nil {
		panic(err)
	}

	return lane
}
