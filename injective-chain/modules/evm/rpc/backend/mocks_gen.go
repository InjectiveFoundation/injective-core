//go:generate mockery --name QueryClient --dir ../../types --filename evm_query_client.go --structname EVMQueryClient
//go:generate mockery --name CometRPCClient --dir . --filename client.go --structname Client

package backend

import (
	cmrpcclient "github.com/cometbft/cometbft/rpc/client"
)

type CometRPCClient interface {
	cmrpcclient.Client
}
