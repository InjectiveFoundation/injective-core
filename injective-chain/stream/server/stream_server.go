package server

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	"cosmossdk.io/log"
	"github.com/cometbft/cometbft/libs/pubsub"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	txfeeskeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/stream/types/v2"
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

type QueryContextProvider func(height int64, skip bool) (sdk.Context, error)

type StreamServer struct {
	bufferCapacity       uint
	Bus                  *pubsub.Server
	GrpcServer           *grpc.Server
	listener             net.Listener
	done                 chan struct{}
	exchangeKeeper       *exchangekeeper.Keeper
	txfeesKeeper         *txfeeskeeper.Keeper
	queryContextProvider QueryContextProvider
}

func NewChainStreamServer(
	bus *pubsub.Server,
	appOpts servertypes.AppOptions,
	exchangeKeeper *exchangekeeper.Keeper,
	txfeesKeeper *txfeeskeeper.Keeper,
	contextProvider QueryContextProvider,
) *StreamServer {
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
		Bus:                  bus,
		bufferCapacity:       100,
		exchangeKeeper:       exchangeKeeper,
		txfeesKeeper:         txfeesKeeper,
		queryContextProvider: contextProvider,
	}
	grpcServer := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
	types.RegisterStreamServer(grpcServer, server)
	v2.RegisterStreamServer(grpcServer, server)
	reflection.Register(grpcServer)
	server.GrpcServer = grpcServer
	return server
}

func (s *StreamServer) Serve(address string) (err error) {
	if !s.Bus.IsRunning() {
		return errors.New("publisher is not running. Please start publisher first")
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
	marketFinder := exchangekeeper.NewCachedMarketFinder(s.exchangeKeeper)

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

	v2Req := NewV2StreamRequestFromV1(req)

	return s.listenStream(v2Req, server, ch, marketFinder)
}

func (s *StreamServer) listenStream(
	v2Req v2.StreamRequest, server types.Stream_StreamServer, ch <-chan pubsub.Message, marketFinder *exchangekeeper.CachedMarketFinder,
) error {
	var height uint64
	for {
		select {
		case <-s.done:
			return status.Error(codes.Canceled, "server is shutting down")
		case message := <-ch:
			newHeight, err := s.processMessage(message, v2Req, server, height, marketFinder)
			if err != nil {
				return err
			}
			height = newHeight
		case <-server.Context().Done():
			return nil
		}
	}
}

func (s *StreamServer) processMessage(
	message pubsub.Message, v2Req v2.StreamRequest, server types.Stream_StreamServer,
	height uint64, marketFinder *exchangekeeper.CachedMarketFinder,
) (uint64, error) {
	inResp, newHeight, err := s.validateAndExtractResponse(message, height)
	if err != nil {
		return height, err
	}
	if inResp == nil {
		return height, nil
	}

	return s.processAndSendResponse(inResp, &v2Req, server, newHeight, marketFinder)
}

func (*StreamServer) validateAndExtractResponse(message pubsub.Message, height uint64) (*v2.StreamResponseMap, uint64, error) {
	if err, ok := message.Data().(error); ok {
		return nil, height, status.Error(codes.Internal, err.Error())
	}

	inResp, ok := message.Data().(*v2.StreamResponseMap)
	if !ok {
		return nil, height, nil
	}

	inResp.RLock()
	defer inResp.RUnlock()

	newHeight := height
	if height == 0 {
		newHeight = inResp.BlockHeight
	} else if inResp.BlockHeight != height {
		return nil, height, status.Error(codes.Internal, "block height mismatch")
	}

	return inResp, newHeight, nil
}

func (s *StreamServer) processAndSendResponse(
	inResp *v2.StreamResponseMap, v2Req *v2.StreamRequest,
	server types.Stream_StreamServer, height uint64,
	marketFinder *exchangekeeper.CachedMarketFinder,
) (uint64, error) {
	outResp, err := s.streamResponseFromMap(inResp, v2Req)
	if err != nil {
		return height, err
	}

	// We might be processing events before the Injective app new block is available.
	// To avoid issues we create a query context for the event height - 1.
	ctx, err := s.queryContextProvider(int64(height-1), false)
	if err != nil {
		return height, status.Error(codes.Internal, err.Error())
	}
	v1Response, err := NewV1StreamResponseFromV2(ctx, outResp, marketFinder)
	if err != nil {
		return height, status.Error(codes.Internal, err.Error())
	}

	err = server.Send(v1Response)
	if err != nil {
		return height, status.Error(codes.Internal, err.Error())
	}
	return height + 1, nil
}

func (s *StreamServer) StreamV2(req *v2.StreamRequest, server v2.Stream_StreamV2Server) error {
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

	return s.listenStreamV2(req, server, ch)
}

func (s *StreamServer) listenStreamV2(req *v2.StreamRequest, server v2.Stream_StreamV2Server, ch <-chan pubsub.Message) error {
	var height uint64
	for {
		select {
		case <-s.done:
			return status.Error(codes.Canceled, "server is shutting down")
		case message := <-ch:
			newHeight, err := s.processMessageV2(message, req, server, height)
			if err != nil {
				return err
			}
			height = newHeight
		case <-server.Context().Done():
			return nil
		}
	}
}

func (s *StreamServer) processMessageV2(
	message pubsub.Message, req *v2.StreamRequest, server v2.Stream_StreamV2Server, height uint64,
) (uint64, error) {
	inResp, newHeight, err := s.validateAndExtractResponse(message, height)
	if err != nil {
		return height, err
	}
	if inResp == nil {
		return height, nil
	}

	outResp, err := s.streamResponseFromMap(inResp, req)
	if err != nil {
		return newHeight, err
	}

	err = server.Send(outResp)
	if err != nil {
		return newHeight, status.Error(codes.Internal, err.Error())
	}
	return newHeight + 1, nil
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

func (s *StreamServer) streamResponseFromMap(inResp *v2.StreamResponseMap, req *v2.StreamRequest) (*v2.StreamResponse, error) {
	inResp.RLock()
	defer inResp.RUnlock()

	outResp := v2.NewChainStreamResponse()

	// Set common fields
	outResp.BlockHeight = inResp.BlockHeight
	outResp.BlockTime = inResp.BlockTime.UnixMilli()

	// Process each filter type using helper functions

	processBankBalances(req, inResp, outResp)

	if err := processSpotOrders(req, inResp, outResp); err != nil {
		return nil, err
	}

	if err := processDerivativeOrders(req, inResp, outResp); err != nil {
		return nil, err
	}

	processOrderbooks(req, inResp, outResp)

	if err := processPositions(req, inResp, outResp); err != nil {
		return nil, err
	}

	processSubaccountDeposits(req, inResp, outResp)
	processOraclePrices(req, inResp, outResp)

	if err := processTrades(req, inResp, outResp); err != nil {
		return nil, err
	}

	outResp.GasPrice = s.txfeesKeeper.CurFeeState.GetCurBaseFee().String()

	return outResp, nil
}

// processBankBalances handles bank balance filtering
func processBankBalances(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) {
	if req.BankBalancesFilter != nil && inResp.BankBalancesByAccount != nil {
		outResp.BankBalances = Filter(inResp.BankBalancesByAccount, req.BankBalancesFilter.Accounts)
	}
}

// processSpotOrders handles spot orders filtering
func processSpotOrders(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) error {
	if req.SpotOrdersFilter != nil && inResp.SpotOrdersByMarketID != nil {
		var err error
		outResp.SpotOrders, err = FilterMulti(
			inResp.SpotOrdersByMarketID,
			inResp.SpotOrdersBySubaccount,
			req.SpotOrdersFilter.MarketIds,
			req.SpotOrdersFilter.SubaccountIds,
		)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

// processDerivativeOrders handles derivative orders filtering
func processDerivativeOrders(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) error {
	if req.DerivativeOrdersFilter != nil && inResp.DerivativeOrdersByMarketID != nil {
		var err error
		outResp.DerivativeOrders, err = FilterMulti(
			inResp.DerivativeOrdersByMarketID,
			inResp.DerivativeOrdersBySubaccount,
			req.DerivativeOrdersFilter.MarketIds,
			req.DerivativeOrdersFilter.SubaccountIds,
		)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

// processOrderbooks handles both spot and derivative orderbooks filtering
func processOrderbooks(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) {
	// Process spot orderbooks
	if req.SpotOrderbooksFilter != nil && inResp.SpotOrderbookUpdatesByMarketID != nil {
		outResp.SpotOrderbookUpdates = Filter(
			inResp.SpotOrderbookUpdatesByMarketID,
			req.SpotOrderbooksFilter.MarketIds,
		)
	}

	// Process derivative orderbooks
	if req.DerivativeOrderbooksFilter != nil && inResp.DerivativeOrderbookUpdatesByMarketID != nil {
		outResp.DerivativeOrderbookUpdates = Filter(
			inResp.DerivativeOrderbookUpdatesByMarketID,
			req.DerivativeOrderbooksFilter.MarketIds,
		)
	}
}

// processPositions handles positions filtering
func processPositions(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) error {
	if req.PositionsFilter != nil && inResp.PositionsByMarketID != nil {
		var err error
		outResp.Positions, err = FilterMulti(
			inResp.PositionsByMarketID,
			inResp.PositionsBySubaccount,
			req.PositionsFilter.MarketIds,
			req.PositionsFilter.SubaccountIds,
		)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

// processSubaccountDeposits handles subaccount deposits filtering
func processSubaccountDeposits(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) {
	if req.SubaccountDepositsFilter != nil && inResp.SubaccountDepositsBySubaccountID != nil {
		outResp.SubaccountDeposits = Filter(
			inResp.SubaccountDepositsBySubaccountID,
			req.SubaccountDepositsFilter.SubaccountIds,
		)
	}
}

// processOraclePrices handles oracle price filtering
func processOraclePrices(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) {
	if req.OraclePriceFilter != nil && inResp.OraclePriceBySymbol != nil {
		outResp.OraclePrices = Filter(
			inResp.OraclePriceBySymbol,
			req.OraclePriceFilter.Symbol,
		)
	}
}

// processTrades handles both spot and derivative trades filtering
func processTrades(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) error {
	if err := processSpotTrades(req, inResp, outResp); err != nil {
		return err
	}

	return processDerivativeTrades(req, inResp, outResp)
}

// processSpotTrades handles spot trades filtering
func processSpotTrades(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) error {
	if req.SpotTradesFilter != nil && inResp.SpotTradesByMarketID != nil {
		var err error
		outResp.SpotTrades, err = FilterMulti(
			inResp.SpotTradesByMarketID,
			inResp.SpotTradesBySubaccount,
			req.SpotTradesFilter.MarketIds,
			req.SpotTradesFilter.SubaccountIds,
		)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

// processDerivativeTrades handles derivative trades filtering
func processDerivativeTrades(req *v2.StreamRequest, inResp *v2.StreamResponseMap, outResp *v2.StreamResponse) error {
	if req.DerivativeTradesFilter != nil && inResp.DerivativeTradesByMarketID != nil {
		var err error
		outResp.DerivativeTrades, err = FilterMulti(
			inResp.DerivativeTradesByMarketID,
			inResp.DerivativeTradesBySubaccount,
			req.DerivativeTradesFilter.MarketIds,
			req.DerivativeTradesFilter.SubaccountIds,
		)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}
