package stream

import (
	"context"
	"fmt"
	"github.com/cometbft/cometbft/libs/pubsub"
	"github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/google/uuid"
	log "github.com/xlab/suplog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"net"

	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
)

type StreamServer struct {
	bufferCapacity uint
	Bus            *pubsub.Server
	GrpcServer     *grpc.Server
	listener       net.Listener
	done           chan struct{}
}

func NewChainStreamServer(bus *pubsub.Server) *StreamServer {
	server := &StreamServer{
		Bus:            bus,
		bufferCapacity: 100,
	}
	grpcServer := grpc.NewServer()
	types.RegisterStreamServer(grpcServer, server)
	reflection.Register(grpcServer)
	server.GrpcServer = grpcServer
	return server
}

func (s *StreamServer) Serve(address string) (err error) {
	if !s.Bus.IsRunning() {
		return fmt.Errorf("publisher is not running. Please start publisher first")
	}
	// init tcp server
	s.listener, err = net.Listen("tcp", address)
	if err != nil {
		return err
	}
	log.Infoln("stream server started at", address)
	go func() {
		if err := s.GrpcServer.Serve(s.listener); err != nil {
			log.WithError(err).Errorf("failed to start chainstream server at %s. Error: %s", address, err.Error())
		}
	}()
	return nil
}

func (s *StreamServer) Stop() {
	log.Infoln("stopping stream server")
	s.GrpcServer.Stop()
}

func (s *StreamServer) Stream(req *types.StreamRequest, server types.Stream_StreamServer) error {
	if err := req.Validate(); err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	clientId := uuid.New().String()
	sub, err := s.Bus.Subscribe(context.Background(), clientId, query.Empty{}, int(s.bufferCapacity))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to topic: %s", err.Error())
	}

	defer func() {
		err = s.Bus.Unsubscribe(context.Background(), clientId, query.Empty{})
		if err != nil {
			log.WithError(err).Errorln("failed to unsubscribe from topic", StreamEvents)
		}
	}()

	ch := sub.Out()

	var height uint64
	for {
		select {
		case <-s.done:
			return status.Errorf(codes.Canceled, "server is shutting down")
		case message := <-ch:
			if err, ok := message.Data().(error); ok {
				return status.Errorf(codes.Internal, err.Error())
			}

			inResp, ok := message.Data().(*types.StreamResponseMap)
			if !ok {
				continue
			}

			if height == 0 {
				height = inResp.BlockHeight
			} else if inResp.BlockHeight != height {
				return status.Errorf(codes.Internal, "block height mismatch")
			}

			outResp := types.NewChainStreamResponse()

			outResp.BlockHeight = height
			outResp.BlockTime = inResp.BlockTime.UnixMilli()

			if req.BankBalancesFilter != nil && inResp.BankBalancesByAccount != nil {
				outResp.BankBalances = Filter[types.BankBalance](inResp.BankBalancesByAccount, req.BankBalancesFilter.Accounts)
			}
			if req.SpotOrdersFilter != nil && inResp.SpotOrdersByMarketID != nil {
				outResp.SpotOrders, err = FilterMulti[types.SpotOrderUpdate](inResp.SpotOrdersByMarketID, inResp.SpotOrdersBySubaccount, req.SpotOrdersFilter.MarketIds, req.SpotOrdersFilter.SubaccountIds)
				if err != nil {
					return status.Errorf(codes.Internal, err.Error())
				}
			}
			if req.DerivativeOrdersFilter != nil && inResp.DerivativeOrdersByMarketID != nil {
				outResp.DerivativeOrders, err = FilterMulti[types.DerivativeOrderUpdate](inResp.DerivativeOrdersByMarketID, inResp.DerivativeOrdersBySubaccount, req.DerivativeOrdersFilter.MarketIds, req.DerivativeOrdersFilter.SubaccountIds)
				if err != nil {
					return status.Errorf(codes.Internal, err.Error())
				}
			}
			if req.SpotOrderbooksFilter != nil && inResp.SpotOrderbookUpdatesByMarketID != nil {
				outResp.SpotOrderbookUpdates = Filter[types.OrderbookUpdate](inResp.SpotOrderbookUpdatesByMarketID, req.SpotOrderbooksFilter.MarketIds)
			}
			if req.DerivativeOrderbooksFilter != nil && inResp.DerivativeOrderbookUpdatesByMarketID != nil {
				outResp.DerivativeOrderbookUpdates = Filter[types.OrderbookUpdate](inResp.DerivativeOrderbookUpdatesByMarketID, req.DerivativeOrderbooksFilter.MarketIds)
			}
			if req.PositionsFilter != nil && inResp.PositionsByMarketID != nil {
				outResp.Positions, err = FilterMulti[types.Position](inResp.PositionsByMarketID, inResp.PositionsBySubaccount, req.PositionsFilter.MarketIds, req.PositionsFilter.SubaccountIds)
				if err != nil {
					return status.Errorf(codes.Internal, err.Error())
				}
			}
			if req.SubaccountDepositsFilter != nil && inResp.SubaccountDepositsBySubaccountID != nil {
				outResp.SubaccountDeposits = Filter[types.SubaccountDeposits](inResp.SubaccountDepositsBySubaccountID, req.SubaccountDepositsFilter.SubaccountIds)
			}
			if req.OraclePriceFilter != nil && inResp.OraclePriceBySymbol != nil {
				outResp.OraclePrices = Filter[types.OraclePrice](inResp.OraclePriceBySymbol, req.OraclePriceFilter.Symbol)
			}
			if req.SpotTradesFilter != nil && inResp.SpotTradesByMarketID != nil {
				outResp.SpotTrades, err = FilterMulti[types.SpotTrade](inResp.SpotTradesByMarketID, inResp.SpotTradesBySubaccount, req.SpotTradesFilter.MarketIds, req.SpotTradesFilter.SubaccountIds)
				if err != nil {
					return status.Errorf(codes.Internal, err.Error())
				}
			}
			if req.DerivativeTradesFilter != nil && inResp.DerivativeTradesByMarketID != nil {
				outResp.DerivativeTrades, err = FilterMulti[types.DerivativeTrade](inResp.DerivativeTradesByMarketID, inResp.DerivativeTradesBySubaccount, req.DerivativeTradesFilter.MarketIds, req.DerivativeTradesFilter.SubaccountIds)
				if err != nil {
					return status.Errorf(codes.Internal, err.Error())
				}
			}
			err = server.Send(outResp)
			if err != nil {
				return status.Errorf(codes.Internal, err.Error())
			}
			height += 1
		case <-server.Context().Done():
			return nil
		}
	}
}

func (s *StreamServer) WithBufferCapacity(capacity uint) {
	s.bufferCapacity = capacity
}

func (s *StreamServer) GetCurrentServerPort() int {
	if s.listener == nil {
		return 0
	}
	return s.listener.Addr().(*net.TCPAddr).Port
}
