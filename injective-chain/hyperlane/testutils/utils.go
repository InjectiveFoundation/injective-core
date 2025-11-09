package testutils

import (
	simapp "github.com/InjectiveLabs/injective-core/injective-chain/app"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/types"
)

type TestValidatorAddress struct {
	Moniker string

	PrivateKey cryptotypes.PrivKey

	Address        string
	AccAddress     sdk.AccAddress
	ConsAccAddress sdk.ConsAddress
	ConsAddress    string
}

func EndBlockerAndCommit(app *simapp.InjectiveApp, ctx sdk.Context) (sdk.Context, []abci.Event) {
	response, _ := app.EndBlocker(ctx)

	exchangeTStore := ctx.TransientStore(app.ExchangeKeeper.GetTransientStoreKey())
	iterator := exchangeTStore.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		exchangeTStore.Delete(iterator.Key())
	}

	ocrTStore := ctx.TransientStore(app.OcrKeeper.GetTransientStoreKey())
	iterator = ocrTStore.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		ocrTStore.Delete(iterator.Key())
	}

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	return ctx, response.Events
}

func GenerateTestValidatorAddress(moniker string) TestValidatorAddress {
	a := TestValidatorAddress{}
	a.Moniker = moniker
	a.PrivateKey = secp256k1.GenPrivKeyFromSecret([]byte(moniker))

	a.AccAddress = sdk.AccAddress(a.PrivateKey.PubKey().Address())
	bech32Address, _ := sdk.Bech32ifyAddressBytes(types.Bech32PrefixAccAddr, a.AccAddress)
	a.Address = bech32Address

	a.ConsAccAddress = sdk.ConsAddress(a.PrivateKey.PubKey().Address())
	bech32ConsAddress, _ := sdk.Bech32ifyAddressBytes(types.Bech32PrefixConsAddr, a.ConsAccAddress)
	a.ConsAddress = bech32ConsAddress

	return a
}
