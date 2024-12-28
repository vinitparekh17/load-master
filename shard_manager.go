package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	ROUND_ROBIN        = "round-robin"
	LEAST_CONN         = "least-connections"
	CONSISTENT_HASHING = "consistent-hashing"
)

type ShardManager struct {
	shards             []Shard
	shardCounter       int // will be used to keep track of the current shard index for req. distribution
	wg                 sync.WaitGroup
	globalErrChan      chan error
	globalProxyReqChan chan *ProxyRequest
	mu                 sync.Mutex
	signalChan         chan os.Signal
}

type ProxyRequest struct {
	OriginalRequest *http.Request
	ResponseWriter  http.ResponseWriter
	UpstreamAddrs   []string
}

func NewShardManager() *ShardManager {
	// initiating shards with incremental ids
	shards := make([]Shard, SlbConfig.ShardCount)
	for i := 0; i < len(shards); i++ {
		shards[i] = *NewShard(i)
	}

	return &ShardManager{
		shards:             shards,
		globalErrChan:      make(chan error),
		globalProxyReqChan: make(chan *ProxyRequest),
		signalChan:         make(chan os.Signal, 1),
	}
}

func (sm *ShardManager) Run(ctx context.Context) {
	signal.Notify(sm.signalChan, syscall.SIGINT, syscall.SIGTERM)

	shardCtx, cancelShardCtx := context.WithCancel(ctx)
	defer cancelShardCtx()

	//starting shards
	for i := range sm.shards {
		sm.wg.Add(1)
		go func(s *Shard) {
			defer sm.wg.Done()
			s.Run(shardCtx)
		}(&sm.shards[i])
	}

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.DistributeProxy(shardCtx)
	}()

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.handleError(shardCtx)
	}()

	select {
	case <-ctx.Done():
		slog.Info("context cancelled...")
	case <-sm.signalChan:
		slog.Info("receieved termination signal...")
	}

	cancelShardCtx()
	close(sm.globalErrChan)
	close(sm.globalProxyReqChan)

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	done := make(chan struct{})
	go func() {
		sm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("shutting down shards gracefully")
	case <-shutdownCtx.Done():
		slog.Warn("shards shutdown timeout exceeded")
	}
}

func (sm *ShardManager) handleError(shardCtx context.Context) {
	for {
		select {
		case <-shardCtx.Done():
			return
		case err, ok := <-sm.globalErrChan:
			if !ok {
				return
			}
			slog.Error(err.Error())
		}
	}
}

// DistributeProxy distributes the incoming req across shards to process request
func (sm *ShardManager) DistributeProxy(shardCtx context.Context) {
	for proxyReq := range sm.globalProxyReqChan {
		// using round robin algo to distribute req across chards evenly
		sm.mu.Lock()
		index := sm.shardCounter % len(sm.shards)
		sm.shardCounter = (sm.shardCounter + 1) % len(sm.shards)
		sm.shards[index].proxyReqChan <- proxyReq
		sm.mu.Unlock()
	}
}
