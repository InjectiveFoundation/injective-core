package stream

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	"cosmossdk.io/log"
	"github.com/cometbft/cometbft/libs/pubsub"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
)

const (
	FlagStreamServer                    = "chainstream-server"
	FlagStreamServerBufferCapacity      = "chainstream-buffer-cap"
	FlagStreamPublisherBufferCapacity   = "chainstream-publisher-buffer-cap"
	FlagStreamEnforceKeepalive          = "chainstream-enforce-keepalive"
	FlagStreamMinClientPingInterval     = "chainstream-min-client-ping-interval"
	FlagStreamMaxConnectionIdle         = "chainstream-max-connection-idle"
	FlagStreamServerPingInterval        = "chainstream-server-ping-interval"
	FlagStreamServerPingResponseTimeout = "chainstream-server-ping-response-timeout"
)

type StreamServer struct {
	bufferCapacity uint
	Bus            *pubsub.Server
	GrpcServer     *grpc.Server
	listener       net.Listener
	done           chan struct{}
}

func NewChainStreamServer(bus *pubsub.Server, appOpts servertypes.AppOptions) *StreamServer {
	shouldEnforceKeepalive := cast.ToBool(appOpts.Get(FlagStreamEnforceKeepalive))
	keepaliveMinClientPingInterval := cast.ToInt64(appOpts.Get(FlagStreamMinClientPingInterval))
	keepaliveMaxConnectionIdle := cast.ToInt64(appOpts.Get(FlagStreamMaxConnectionIdle))
	keepaliveServerPingInterval := cast.ToInt64(appOpts.Get(FlagStreamServerPingInterval))
	keepaliveServerPingResponseTimeout := cast.ToInt64(appOpts.Get(FlagStreamServerPingResponseTimeout))

	var kaep = keepalive.EnforcementPolicy{}
	var kasp = keepalive.ServerParameters{}

	if shouldEnforceKeepalive {
		kaep.MinTime = time.Duration(keepaliveMinClientPingInterval) * time.Second
		kasp.MaxConnectionIdle = time.Duration(keepaliveMaxConnectionIdle) * time.Second
		kasp.Time = time.Duration(keepaliveServerPingInterval) * time.Second
		kasp.Timeout = time.Duration(keepaliveServerPingResponseTimeout) * time.Second
	}

	server := &StreamServer{
		Bus:            bus,
		bufferCapacity: 100,
	}
	grpcServer := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
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
	logger := log.NewLogger(os.Stderr)
	logger.Info("stream server started", "address", address)
	go func() {
		if err := s.GrpcServer.Serve(s.listener); err != nil {
			logger.Error("failed to start chainstream server", "address", address, "error", err)
		}
	}()
	return nil
}

func (s *StreamServer) Stop() {
	log.NewLogger(os.Stderr).Info("stopping stream server")
	s.GrpcServer.Stop()
}

func (s *StreamServer) Stream(req *types.StreamRequest, server types.Stream_StreamServer) error {
	if err := req.Validate(); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	clientId := uuid.New().String()
	sub, err := s.Bus.Subscribe(context.Background(), clientId, types.Empty{}, int(s.bufferCapacity))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to topic: %s", err.Error())
	}

	defer func() {
		err = s.Bus.Unsubscribe(context.Background(), clientId, types.Empty{})
		if err != nil {
			log.NewLogger(os.Stderr).Error("failed to unsubscribe from topic", "error", err, "clientId", clientId)
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
				return status.Error(codes.Internal, err.Error())
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
					return status.Error(codes.Internal, err.Error())
				}
			}
			if req.DerivativeOrdersFilter != nil && inResp.DerivativeOrdersByMarketID != nil {
				outResp.DerivativeOrders, err = FilterMulti[types.DerivativeOrderUpdate](inResp.DerivativeOrdersByMarketID, inResp.DerivativeOrdersBySubaccount, req.DerivativeOrdersFilter.MarketIds, req.DerivativeOrdersFilter.SubaccountIds)
				if err != nil {
					return status.Error(codes.Internal, err.Error())
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
					return status.Error(codes.Internal, err.Error())
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
					return status.Error(codes.Internal, err.Error())
				}
			}
			if req.DerivativeTradesFilter != nil && inResp.DerivativeTradesByMarketID != nil {
				outResp.DerivativeTrades, err = FilterMulti[types.DerivativeTrade](inResp.DerivativeTradesByMarketID, inResp.DerivativeTradesBySubaccount, req.DerivativeTradesFilter.MarketIds, req.DerivativeTradesFilter.SubaccountIds)
				if err != nil {
					return status.Error(codes.Internal, err.Error())
				}
			}
			err = server.Send(outResp)
			if err != nil {
				return status.Error(codes.Internal, err.Error())
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
