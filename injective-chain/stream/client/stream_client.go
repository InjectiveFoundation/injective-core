package main

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
)

func main() {
	cc, err := grpc.Dial("localhost:9999", grpc.WithTransportCredentials(insecure.NewCredentials()))
	// nolint:staticcheck //ignored on purpose
	defer cc.Close()
	if err != nil {
		panic(err)
	}
	client := types.NewStreamClient(cc)

	ctx := context.Background()
	stream, err := client.Stream(ctx, &types.StreamRequest{
		BankBalancesFilter: &types.BankBalancesFilter{
			Accounts: []string{"*"},
		},
		SpotOrdersFilter: &types.OrdersFilter{
			MarketIds:     []string{"*"},
			SubaccountIds: []string{"*"},
		},
		DerivativeOrdersFilter: &types.OrdersFilter{
			MarketIds:     []string{"*"},
			SubaccountIds: []string{"*"},
		},
		SpotTradesFilter: &types.TradesFilter{
			MarketIds:     []string{"*"},
			SubaccountIds: []string{"*"},
		},
		SubaccountDepositsFilter: &types.SubaccountDepositsFilter{
			SubaccountIds: []string{"*"},
		},
		DerivativeOrderbooksFilter: &types.OrderbookFilter{
			MarketIds: []string{"*"},
		},
		SpotOrderbooksFilter: &types.OrderbookFilter{
			MarketIds: []string{"*"},
		},
		PositionsFilter: &types.PositionsFilter{
			SubaccountIds: []string{"*"},
			MarketIds:     []string{"*"},
		},
		DerivativeTradesFilter: &types.TradesFilter{
			SubaccountIds: []string{"*"},
			MarketIds:     []string{"*"},
		},
		OraclePriceFilter: &types.OraclePriceFilter{
			Symbol: []string{"*"},
		},
	})
	if err != nil {
		panic(err)
	}

	for {
		res, err := stream.Recv()
		if err != nil {
			panic(err)
		}
		bz, _ := json.Marshal(res)
		fmt.Println(string(bz))
	}
}
