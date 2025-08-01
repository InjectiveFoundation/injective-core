package watchdog

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
)

type WatchdogProcess interface {
	Start() error
	Stop() error
	IsRunning() bool
	IsHealthy() bool
}

type watchdogProcess struct {
	bftRPCAddr   string
	rpcClient    rpcclient.Client
	logger       log.Logger
	rootCtx      context.Context
	rootCancel   context.CancelFunc
	stateMux     *sync.RWMutex
	isRunning    bool
	isHealthy    bool
	maxStuckTime time.Duration

	lastBlock          int64
	timestampLastBlock time.Time
	watchdogDump       bool
}

// bftRPCAddr is the address of the cometbft rpc server, usually tcp://localhost:26657
func NewWatchdogProcess(
	ctx context.Context,
	logger log.Logger,
	maxStuckTime time.Duration,
	bftRPCAddr string,
	watchdogDump bool,
) *watchdogProcess {
	rootCtx, rootCancel := context.WithCancel(ctx)
	return &watchdogProcess{
		bftRPCAddr:   bftRPCAddr,
		logger:       logger,
		rootCtx:      rootCtx,
		rootCancel:   rootCancel,
		stateMux:     new(sync.RWMutex),
		isRunning:    false,
		isHealthy:    true,
		lastBlock:    -1,
		maxStuckTime: maxStuckTime,
		watchdogDump: watchdogDump,
	}
}

func (w *watchdogProcess) lastBlockFromRPC(ctx context.Context) (int64, time.Time, error) {
	if w.rpcClient == nil {
		rpcClient, err := rpchttp.NewWithTimeout(w.bftRPCAddr, 10)
		if err != nil {
			err = errors.Wrap(err, "failed to init rpcClient")
			return 0, time.Time{}, err
		}
		w.rpcClient = rpcClient
	}

	status, err := w.rpcClient.Status(ctx)
	if err != nil {
		return -1, time.Time{}, err
	}

	height := status.SyncInfo.LatestBlockHeight
	timestamp := status.SyncInfo.LatestBlockTime

	return height, timestamp, nil
}

// Start must be run in a goroutine and do not catch panics.
func (w *watchdogProcess) Start() error {
	time.Sleep(10 * time.Second)

	var lastBlock int64
	var timestamp time.Time
	var err error

	// Initial init loop
	for {
		lastBlock, timestamp, err = w.lastBlockFromRPC(w.rootCtx)
		if err != nil {
			w.logger.Error("===== WATCHDOG: failed to get last block from rpc", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		w.stateMux.Lock()
		w.lastBlock = lastBlock
		w.timestampLastBlock = timestamp
		w.isRunning = true
		w.stateMux.Unlock()
		break
	}

	w.logger.Info("===== WATCHDOG: read last block from rpc", "last_block", lastBlock, "timestamp_last_block", timestamp)

	// Wait for the first block change to consider chain progressing.
	previousKnownBlock := lastBlock

	for {
		lastBlock, timestamp, err := w.lastBlockFromRPC(w.rootCtx)
		if err != nil {
			w.logger.Info("===== WATCHDOG: failed to get last block from rpc", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if lastBlock > previousKnownBlock {
			if timestamp.After(time.Now().Add(30 * -time.Second)) {
				// the block must be fresh enough to be considered as live, not just syncing

				w.logger.Info("===== WATCHDOG: chain is progressing", "last_block", lastBlock, "timestamp_last_block", timestamp)

				// set the last block
				w.stateMux.Lock()
				w.lastBlock = lastBlock
				w.timestampLastBlock = timestamp
				w.isHealthy = true
				w.stateMux.Unlock()

				time.Sleep(10 * time.Second)
				break
			}

			previousKnownBlock = lastBlock
		}

		// wait for the next check in 10 seconds
		// this can wait for live blocks for some time.
		time.Sleep(10 * time.Second)
	}

	// At this moment, chain is live, and we must check occasionally if it's not stuck.

	for {
		lastBlock, timestamp, err := w.lastBlockFromRPC(w.rootCtx)
		if err != nil {
			// doesn't consider stuck here
			w.logger.Info("===== WATCHDOG: failed to get last block from rpc", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		w.stateMux.RLock()
		previousKnownBlock := w.lastBlock
		previousKnownTimestamp := w.timestampLastBlock
		isHealthy := w.isHealthy
		w.stateMux.RUnlock()

		if !isHealthy {
			// if the chain is not healthy, we don't need to check if it's stuck
			w.logger.Info("===== WATCHDOG: chain is not healthy, skipping stuck check", "last_block", lastBlock, "timestamp_last_block", timestamp)
			break
		}

		if lastBlock == previousKnownBlock {
			// block didn't change

			// and timeout is reached from last seen timestamp
			if timestamp.Before(time.Now().Add(-w.maxStuckTime)) {
				w.logger.Error("===== WATCHDOG: DETECTED A STUCK BLOCK =====", "last_block", previousKnownBlock, "timestamp_last_block", previousKnownTimestamp)

				// simply exist if dump not requested
				if !w.watchdogDump {
					w.logger.Error("===== WATCHDOG: RESTARTING")
					os.Exit(1)
					return nil
				}

				w.logger.Error("===== PLEASE SHARE STACK TRACE BELOW WITH INJECTIVE TEAM =====")

				if err := w.stopProcessSIGQUIT(); err != nil {
					w.logger.Error("failed to stop process via SIGQUIT, making a panic in goroutine; set GOTRACEBACK=all to see the stack trace of all goroutines", "error", err)
					panic("watchdog restart")
				}
			}
		} else {
			// set the last block
			w.stateMux.Lock()
			w.lastBlock = lastBlock
			w.timestampLastBlock = timestamp
			w.stateMux.Unlock()
		}

		time.Sleep(10 * time.Second)
	}

	return nil
}

func (w *watchdogProcess) Stop() error {
	w.stateMux.Lock()
	defer w.stateMux.Unlock()

	if !w.isRunning {
		return nil
	}

	w.rootCancel()
	w.isRunning = false
	return nil
}

func (w *watchdogProcess) IsRunning() bool {
	w.stateMux.RLock()
	defer w.stateMux.RUnlock()
	return w.isRunning
}

func (w *watchdogProcess) IsHealthy() bool {
	w.stateMux.RLock()
	defer w.stateMux.RUnlock()
	return w.isHealthy
}

func (w *watchdogProcess) stopProcessSIGQUIT() error {
	// Get the current process ID
	pid := os.Getpid()

	// Create a process object for the current process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find own process %d: %w", pid, err)
	}

	// Send SIGQUIT signal to the process
	if err := process.Signal(syscall.SIGQUIT); err != nil {
		return fmt.Errorf("failed to send SIGQUIT to own process %d: %w", pid, err)
	}

	return nil
}
