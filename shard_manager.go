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
	globalProxyReqChan chan ProxyRequest
	mu                 sync.Mutex
	signalChan         chan os.Signal
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
		globalProxyReqChan: make(chan ProxyRequest),
		signalChan:         make(chan os.Signal, 1),
	}
}

func (sm *ShardManager) Run(ctx context.Context) {
	signal.Notify(sm.signalChan, syscall.SIGINT, syscall.SIGTERM)

	shardCtx, cancelShardCtx := context.WithCancel(ctx)
	defer cancelShardCtx()

	//starting shards
	for i := 0; i < len(sm.shards); i++ {
		sm.wg.Add(1)
		shard := &sm.shards[i] // Create a copy of the pointer to avoid capturing the loop variable.
		go func(s *Shard) {
			defer sm.wg.Done()
			s.Run(shardCtx)
		}(shard)
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

	close(sm.globalProxyReqChan)
	sm.wg.Wait()
}

func (sm *ShardManager) handleError(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(sm.globalErrChan)
			return
		case err, ok := <-sm.globalErrChan:
			if !ok {
				return
			}
			slog.Error(err.Error())
		}
	}
}
func (sm *ShardManager) DistributeProxy(shardCtx context.Context) {
	for proxyReq := range sm.globalProxyReqChan {
		// Check if the request context is already canceled
		if proxyReq.OriginalRequest.Context().Err() != nil {
			slog.Info("Request already canceled during distribution",
				"error", proxyReq.OriginalRequest.Context().Err())
			continue
		}

		var shardIndex int
		if len(sm.shards) > 1 {
			sm.mu.Lock()
			shardIndex = sm.shardCounter % len(sm.shards)
			sm.shardCounter = (sm.shardCounter + 1) % len(sm.shards)
			sm.mu.Unlock()
		}

		// Send to shard with context awareness
		select {
		case sm.shards[shardIndex].proxyReqChan <- proxyReq:
			// Successfully forwarded to shard
		case <-proxyReq.OriginalRequest.Context().Done():
			slog.Info("Request canceled while forwarding to shard",
				"error", proxyReq.OriginalRequest.Context().Err())
		case <-time.After(2 * time.Second):
			slog.Error("Timeout forwarding to shard")
			http.Error(proxyReq.ResponseWriter, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}
