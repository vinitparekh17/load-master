package main

import (
	"context"
	"net/http"

	"golang.org/x/sync/semaphore"
)

type WorkerPool struct {
	sem *semaphore.Weighted
}

func NewWorkerPool(size int, ctx context.Context) *WorkerPool {
	return &WorkerPool{sem: semaphore.NewWeighted(int64(size))}
}

func (wp *WorkerPool) Process(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !wp.sem.TryAcquire(1) {
			http.Error(w, "server is busy, try again", http.StatusTooManyRequests)
			return
		}
		defer wp.sem.Release(1)
		handler(w, r)
	}
}
