package client

import (
	"context"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	comethttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	injcodec "github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos/codec"
)

type ContextOption func(c client.Context) client.Context

// NewClientContext creates a new Cosmos Client context, where chainID
// corresponds to Cosmos chain ID, fromSpec is either name of the key, or bech32-address
// of the Cosmos account. Keyring is required to contain the specified key.
func NewClientContext(addr string, opts ...ContextOption) (client.Context, error) {
	cdc := injcodec.Codec()
	cc := client.Context{
		ChainID:           "injective-1", // mainnet
		Codec:             cdc,
		From:              "validator",
		FromName:          "validator",
		InterfaceRegistry: cdc.InterfaceRegistry(),
		Output:            os.Stderr,
		OutputFormat:      "json",
		BroadcastMode:     "sync",
		UseLedger:         false,
		Simulate:          true,
		GenerateOnly:      false,
		Offline:           false,
		SkipConfirm:       true,
		AccountRetriever:  authtypes.AccountRetriever{},
		TxConfig:          tx.NewTxConfig(cdc, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_DIRECT}),
	}

	for _, opt := range opts {
		cc = opt(cc)
	}

	tmRPC, err := comethttp.New(cc.NodeURI)
	if err != nil {
		return client.Context{}, err
	}

	//nolint:staticcheck // breaks clients with grpc.NewClient
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(DialerFunc),
	)

	if err != nil {
		return client.Context{}, errors.Wrapf(err, "failed to connect to cosmos: %s", addr)
	}

	cc = cc.WithClient(tmRPC)
	cc = cc.WithGRPCClient(conn)

	if err := awaitConnection(conn, 10*time.Second); err != nil {
		return client.Context{}, err
	}

	return cc, nil
}

func WithChainID(chainID string) ContextOption {
	return func(c client.Context) client.Context {
		return c.WithChainID(chainID)
	}
}

func WithCometURI(uri string) ContextOption {
	return func(c client.Context) client.Context {
		return c.WithNodeURI(uri)
	}
}

func WithKeyring(k keyring.Keyring) ContextOption {
	return func(c client.Context) client.Context {
		return c.WithKeyring(k)
	}
}

func WithFrom(from string) ContextOption {
	return func(c client.Context) client.Context {
		return c.WithFrom(from)
	}
}

func WithFromAddress(addr cosmostypes.AccAddress) ContextOption {
	return func(c client.Context) client.Context {
		return c.WithFromAddress(addr)
	}
}

func WithFromName(name string) ContextOption {
	return func(c client.Context) client.Context {
		return c.WithFromName(name)
	}
}

func WithCodec(cdc codec.Codec) ContextOption {
	return func(c client.Context) client.Context {
		return c.WithCodec(cdc)
	}
}

func DialerFunc(_ context.Context, addr string) (net.Conn, error) {
	proto, address := ProtocolAndAddress(addr)
	conn, err := net.Dial(proto, address)
	return conn, err
}

// Connect dials the given address and returns a net.Conn. The protoAddr argument should be prefixed with the protocol,
// eg. "tcp://127.0.0.1:8080" or "unix:///tmp/test.sock"
func Connect(protoAddr string) (net.Conn, error) {
	proto, address := ProtocolAndAddress(protoAddr)
	conn, err := net.Dial(proto, address)
	return conn, err
}

// ProtocolAndAddress splits an address into the protocol and address components.
// For instance, "tcp://127.0.0.1:8080" will be split into "tcp" and "127.0.0.1:8080".
// If the address has no protocol prefix, the default is "tcp".
func ProtocolAndAddress(listenAddr string) (protocol, address string) {
	protocol, address = "tcp", listenAddr
	parts := strings.SplitN(address, "://", 2)
	if len(parts) == 2 {
		protocol, address = parts[0], parts[1]
	}
	return protocol, address
}

func awaitConnection(conn *grpc.ClientConn, timeout time.Duration) error {
	ctx, cancelWait := context.WithTimeout(context.Background(), timeout)
	defer cancelWait()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for connection")
		default:
			state := conn.GetState()
			if state != connectivity.Ready && state != connectivity.Idle {
				time.Sleep(1 * time.Second)
				continue
			}

			return nil
		}
	}
}
