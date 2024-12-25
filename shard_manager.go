package main

import (
	"context"
	"net/http"
	"os"
)

type ShardManager struct {
	shards            []shard
	globalErrChan     chan error
	globalRequestChan chan *http.Request
	signalChan        chan os.Signal
}

type shard struct {
	id          string
	requestChan chan *http.Request
	errChan     chan error
}

func NewShardManager() *ShardManager {
	return &ShardManager{
		shards:            make([]shard, SlbConfig.ShardCount),
		globalErrChan:     make(chan error),
		globalRequestChan: make(chan *http.Request),
		signalChan:        make(chan os.Signal, 1),
	}
}

func (sm *ShardManager) Run(ctx context.Context) {

}
