package server

import (
	"context"
	"strings"
	"time"

	"github.com/cometbft/cometbft/libs/service"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

const (
	ServiceName = "EVMIndexerService"

	NewBlockWaitTimeout = 60 * time.Second

	// https://github.com/cometbft/cometbft/blob/v0.37.4/rpc/core/env.go#L193
	NotFoundErr          = "is not available"
	ErrorBackoffDuration = 1 * time.Second
)

// EVMIndexerService indexes transactions for json-rpc service.
type EVMIndexerService struct {
	service.BaseService

	txIdxr   chaintypes.EVMTxIndexer
	client   rpcclient.Client
	allowGap bool
}

// NewEVMIndexerService returns a new service instance.
func NewEVMIndexerService(
	txIdxr chaintypes.EVMTxIndexer,
	client rpcclient.Client,
	allowGap bool,
) *EVMIndexerService {
	is := &EVMIndexerService{txIdxr: txIdxr, client: client, allowGap: allowGap}
	is.BaseService = *service.NewBaseService(nil, ServiceName, is)
	return is
}

// OnStart implements service.Service by subscribing for new blocks
// and indexing them by events.
func (eis *EVMIndexerService) OnStart() error {
	ctx := context.Background()
	status, err := eis.client.Status(ctx)
	if err != nil {
		return err
	}
	latestBlock := status.SyncInfo.LatestBlockHeight
	newBlockSignal := make(chan struct{}, 1)

	// Use SubscribeUnbuffered here to ensure both subscriptions does not get
	// canceled due to not pulling messages fast enough. Cause this might
	// sometimes happen when there are no other subscribers.
	blockHeadersChan, err := eis.client.Subscribe(
		ctx,
		ServiceName,
		types.QueryForEvent(types.EventNewBlockHeader).String(),
		0)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg := <-blockHeadersChan
			eventDataHeader := msg.Data.(types.EventDataNewBlockHeader)
			if eventDataHeader.Header.Height > latestBlock {
				latestBlock = eventDataHeader.Header.Height
				// notify
				select {
				case newBlockSignal <- struct{}{}:
				default:
				}
			}
		}
	}()

	lastBlock, err := eis.txIdxr.LastIndexedBlock()
	if err != nil {
		return err
	}
	if lastBlock == -1 {
		lastBlock = latestBlock
	} else if lastBlock < status.SyncInfo.EarliestBlockHeight {
		if !eis.allowGap {
			panic("Block gap detected, please recover the missing data")
		}
		// to avoid infinite failed to fetch block error when lastBlock is smaller than earliest
		lastBlock = status.SyncInfo.EarliestBlockHeight
	}
	// to avoid height must be greater than 0 error
	if lastBlock <= 0 {
		lastBlock = 1
	}

	for {
		if latestBlock <= lastBlock {
			// nothing to index. wait for signal of new block

			select {
			case <-newBlockSignal:
			case <-time.After(NewBlockWaitTimeout):
			}
			continue
		}
		var (
			err         error
			block       *ctypes.ResultBlock
			blockResult *ctypes.ResultBlockResults
		)
		for i := lastBlock + 1; i <= latestBlock; i++ {
			block, err = eis.client.Block(ctx, &i)
			if err != nil {
				if eis.allowGap && strings.Contains(err.Error(), NotFoundErr) {
					continue
				}
				eis.Logger.Error("failed to fetch block", "height", i, "err", err)
				break
			}
			blockResult, err = eis.client.BlockResults(ctx, &i)
			if err != nil {
				if eis.allowGap && strings.Contains(err.Error(), NotFoundErr) {
					continue
				}
				eis.Logger.Error("failed to fetch block result", "height", i, "err", err)
				break
			}
			if err := eis.txIdxr.IndexBlock(block.Block, blockResult.TxResults); err != nil {
				eis.Logger.Error("failed to index block", "height", i, "err", err)
			}
			lastBlock = blockResult.Height
		}
		if err != nil {
			time.Sleep(ErrorBackoffDuration)
		}
	}
}
