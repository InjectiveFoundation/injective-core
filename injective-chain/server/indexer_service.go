package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/cometbft/cometbft/libs/service"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

const (
	ServiceName = "EVMIndexerService"

	NewBlockWaitTimeout  = 20 * time.Second
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
	latestBlockMux := new(sync.RWMutex)
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
		defer func() {
			if e := recover(); e != nil {
				eis.Logger.Error("evm indexer stopped: panic", "err", e)
				return
			}
		}()

		for {
			msg := <-blockHeadersChan
			eventDataHeader := msg.Data.(types.EventDataNewBlockHeader)

			latestBlockMux.Lock()

			if eventDataHeader.Header.Height > latestBlock {
				latestBlock = eventDataHeader.Header.Height
				latestBlockMux.Unlock()

				// notify
				select {
				case newBlockSignal <- struct{}{}:
				default:
				}
			} else {
				latestBlockMux.Unlock()
			}
		}
	}()

	lastBlock, err := eis.txIdxr.LastIndexedBlock()
	if err != nil {
		return err
	}
	if lastBlock == -1 {
		latestBlockMux.RLock()
		lastBlock = latestBlock
		latestBlockMux.RUnlock()
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
		latestBlockMux.RLock()
		fetchedLatestBlock := latestBlock
		latestBlockMux.RUnlock()

		if fetchedLatestBlock <= lastBlock {
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
		for i := lastBlock + 1; i <= fetchedLatestBlock; i++ {
			err = retry.Do(
				func() error {
					block, err = eis.client.Block(ctx, &i)
					if err != nil {
						return err
					} else if block == nil {
						return fmt.Errorf("block is nil: %d", i)
					}

					return nil
				},
				retry.Attempts(2),
				retry.Delay(100*time.Millisecond),
				retry.DelayType(retry.FixedDelay),
				retry.Context(ctx),
			)
			if err != nil {
				if eis.allowGap {
					eis.Logger.Info("failed to fetch block, skipping", "height", i, "err", err)
					continue
				}

				eis.Logger.Error("failed to fetch block", "height", i, "err", err)
				break
			}

			err = retry.Do(
				func() error {
					blockResult, err = eis.client.BlockResults(ctx, &i)
					if err != nil {
						return err
					} else if blockResult == nil {
						return fmt.Errorf("block result is nil: %d", i)
					}

					return nil
				},
				retry.Attempts(2),
				retry.Delay(100*time.Millisecond),
				retry.DelayType(retry.FixedDelay),
				retry.Context(ctx),
			)
			if err != nil {
				if eis.allowGap {
					eis.Logger.Info("failed to fetch block result, skipping", "height", i, "err", err)
					continue
				}

				eis.Logger.Error("failed to fetch block result", "height", i, "err", err)
				break
			}

			if err = eis.txIdxr.IndexBlock(block.Block, blockResult.TxResults); err != nil {
				// internal indexer error is not recoverable
				eis.Logger.Error("failed to index block", "height", i, "err", err)
				break
			}

			lastBlock = blockResult.Height
		}
		if err != nil {
			// sleep after breaking the inner loop
			time.Sleep(ErrorBackoffDuration)
		}
	}
}
