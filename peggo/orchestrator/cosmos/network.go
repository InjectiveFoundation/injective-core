package cosmos

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos/client"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos/peggy"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos/tendermint"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/ethereum/keystore"
)

type NetworkConfig struct {
	ChainID,
	ValidatorAddress,
	CosmosGRPC,
	TendermintRPC,
	GasPrice string
}

type Network interface {
	peggy.QueryClient
	peggy.BroadcastClient
	tendermint.Client
}

func NewNetwork(
	k Keyring,
	ethSignFn keystore.PersonalSignFn,
	cfg NetworkConfig,
) (Network, error) {
	addr, err := sdktypes.AccAddressFromBech32(cfg.ValidatorAddress)

	var record *keyring.Record
	if err != nil {
		// failed to parse Bech32, is it a name?
		r, err := k.Key(cfg.ValidatorAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "no key in keyring for name: %s", cfg.ValidatorAddress)
		}
		record = r
	} else {
		r, err := k.KeyByAddress(addr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load key info by address %s", addr.String())
		}
		record = r
	}

	keyInfoAddress, err := record.GetAddress()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load key info by address %s", addr.String())
	}

	opts := []client.ContextOption{
		client.WithChainID(cfg.ChainID),
		client.WithCometURI(cfg.TendermintRPC),
		client.WithKeyring(k),
		client.WithFrom(keyInfoAddress.String()),
		client.WithFromAddress(keyInfoAddress),
		client.WithFromName(record.Name),
	}

	clientCtx, err := client.NewClientContext(cfg.CosmosGRPC, opts...)
	if err != nil {
		return nil, err
	}

	chainClient, err := client.NewChainClient(clientCtx, &client.Options{GasPrices: cfg.GasPrice})
	if err != nil {
		return nil, err
	}

	var (
		query = peggy.NewQueryClient(peggytypes.NewQueryClient(clientCtx.GRPCClient))
		tx    = peggy.NewBroadcastClient(chainClient, ethSignFn)
		tm    = tendermint.NewRPCClient(cfg.TendermintRPC)
	)

	net := struct {
		peggy.QueryClient
		peggy.BroadcastClient
		tendermint.Client
	}{
		query,
		tx,
		tm,
	}

	return net, nil
}
