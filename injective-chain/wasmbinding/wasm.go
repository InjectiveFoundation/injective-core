package wasmbinding

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	wasmxkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	oraclekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
)

func RegisterCustomPlugins(
	bankBaseKeeper bankkeeper.BaseKeeper,
	exchangeKeeper *exchangekeeper.Keeper,
	oracleKeeper *oraclekeeper.Keeper,
	tokenFactoryKeeper *tokenfactorykeeper.Keeper,
	wasmxKeeper *wasmxkeeper.Keeper,
	router wasmkeeper.MessageRouter,
) []wasmkeeper.Option {
	wasmQueryPlugin := NewQueryPlugin(exchangeKeeper, oracleKeeper, &bankBaseKeeper, tokenFactoryKeeper, wasmxKeeper)

	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})

	messengerDecoratorOpt := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(router, bankBaseKeeper, exchangeKeeper, tokenFactoryKeeper),
	)

	return []wasm.Option{
		queryPluginOpt,
		messengerDecoratorOpt,
	}
}
