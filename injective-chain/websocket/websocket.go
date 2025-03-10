package stream

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"cosmossdk.io/log"

	"github.com/InjectiveLabs/injective-core/injective-chain/stream"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
	rpcserver "github.com/cometbft/cometbft/rpc/jsonrpc/server"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"google.golang.org/grpc"
)

const (
	FlagWebsocketServer = "websocket-server"

	ResponseSuccess = "success"
)

type WebsocketServer struct {
	streamSvr     *stream.StreamServer
	manager       *rpcserver.WebsocketManager
	subscriptions map[string]map[string]*WsStream
	mux           *sync.RWMutex
	logger        log.Logger
}

type WsStream struct {
	ctx      context.Context
	id       rpctypes.JSONRPCIntID
	cancelFn func()
	wsConn   rpctypes.WSRPCConnection
	grpc.ServerStream
}

func NewWebsocketServer(streamSvr *stream.StreamServer, logger log.Logger) *WebsocketServer {
	s := &WebsocketServer{
		streamSvr:     streamSvr,
		subscriptions: map[string]map[string]*WsStream{},
		mux:           new(sync.RWMutex),
		logger:        logger,
	}
	fnMap := map[string]*rpcserver.RPCFunc{
		"subscribe":   rpcserver.NewWSRPCFunc(s.subscribe, "q"),
		"unsubscribe": rpcserver.NewWSRPCFunc(s.unsubscribe, "q"),
	}
	s.manager = rpcserver.NewWebsocketManager(
		fnMap,
		rpcserver.OnDisconnect(s.onDisconnect),
	)
	return s
}

func (ws *WsStream) Send(sr *types.StreamResponse) error {
	return ws.wsConn.WriteRPCResponse(ws.ctx, rpctypes.NewRPCSuccessResponse(ws.id, *sr))
}

func (ws *WsStream) Context() context.Context {
	return ws.ctx
}

func (s *WebsocketServer) HasSubscriber(subscriber string) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	_, exist := s.subscriptions[subscriber]
	return exist
}

func (s *WebsocketServer) GetAllSubscriptions(subscriber string) (map[string]*WsStream, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	subscriptions, exist := s.subscriptions[subscriber]
	if !exist {
		return nil, false
	}

	return subscriptions, true
}

func (s *WebsocketServer) GetSubscription(subscriber string, subscriptionHash []byte) (*WsStream, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	subscriptions, exist := s.subscriptions[subscriber]
	if !exist {
		return nil, false
	}

	ws, ok := subscriptions[string(subscriptionHash)]
	return ws, ok
}

func (s *WebsocketServer) SetSubscription(subscriber string, subscriptionHash []byte, ws *WsStream) {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, exist := s.subscriptions[subscriber]
	if !exist {
		s.subscriptions[subscriber] = map[string]*WsStream{}
	}

	s.subscriptions[subscriber][string(subscriptionHash)] = ws
}

func (s *WebsocketServer) DeleteSubscription(subscriber string, subscriptionHash []byte) {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, exist := s.subscriptions[subscriber]
	if !exist {
		return
	}

	delete(s.subscriptions[subscriber], string(subscriptionHash))
}

func (s *WebsocketServer) DeleteSubscriber(subscriber string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, exist := s.subscriptions[subscriber]
	if !exist {
		return
	}

	delete(s.subscriptions, subscriber)
}

func (s *WebsocketServer) subscribe(ctx *rpctypes.Context, q *types.StreamRequest) (string, error) {
	requestID, ok := ctx.JSONReq.ID.(rpctypes.JSONRPCIntID)
	if !ok {
		return "", fmt.Errorf("invalid request: expected non-negative int as id")
	}

	subscriber := ctx.RemoteAddr()
	bz, _ := json.Marshal(q)
	rqHash := sha256.Sum256(bz)
	cancelCtx, cancelFn := context.WithCancel(context.Background())
	if stream, exist := s.GetSubscription(subscriber, rqHash[:]); exist {
		cancelFn()
		return "", fmt.Errorf("request exists, id: %d", stream.id)
	}

	ws := &WsStream{
		ctx:      cancelCtx,
		id:       requestID,
		cancelFn: cancelFn,
		wsConn:   ctx.WSConn,
	}
	s.SetSubscription(subscriber, rqHash[:], ws)
	go func() {
		if err := s.streamSvr.Stream(q, ws); err != nil {
			ctx.WSConn.WriteRPCResponse(ws.ctx, rpctypes.NewRPCErrorResponse(requestID, 1, fmt.Sprintf("stream error: %s", err.Error()), ""))
			return
		}
	}()

	return ResponseSuccess, nil
}

func (s *WebsocketServer) unsubscribe(ctx *rpctypes.Context, q *types.StreamRequest) (string, error) {
	_, ok := ctx.JSONReq.ID.(rpctypes.JSONRPCIntID)
	if !ok {
		return "", fmt.Errorf("invalid request: expected non-negative int as id")
	}
	subscriber := ctx.RemoteAddr()
	bz, _ := json.Marshal(q)
	rqHash := sha256.Sum256(bz)
	ws, subscriptionExist := s.GetSubscription(subscriber, rqHash[:])
	if subscriptionExist {
		ws.cancelFn()
		s.DeleteSubscription(subscriber, rqHash[:])
		return ResponseSuccess, nil
	}
	return "", fmt.Errorf("subscription does not exist")
}

func (s *WebsocketServer) onDisconnect(subscriber string) {
	subscriptions, exist := s.GetAllSubscriptions(subscriber)
	if !exist {
		return
	}

	for _, r := range subscriptions {
		r.cancelFn()
	}
	s.DeleteSubscriber(subscriber)
}

func (s *WebsocketServer) Serve(addr string) error {
	http.HandleFunc("/ws", s.manager.WebsocketHandler)
	go func() {
		s.logger.Info("Websocket server started at " + addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			panic(err)
		}
	}()
	return nil
}
