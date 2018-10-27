package main

import (
	"context"
)

// Workers hammer as fast as possbible and RPS is ignored.
func NewWorkerProducer(requester Requester) Producer {
	p := &WorkerProducer{
		requester:   requester,
		concurrency: 0,
	}
	return p
}

type WorkerProducer struct {
	requester            Requester
	concurrency          int
	cancelCurrentWorkers func()
}

func (p *WorkerProducer) Start() error {
	p.startConcurrentWorkers()
	return nil
}

func (p *WorkerProducer) Stop() error {
	p.cancelCurrentWorkers()
	return nil
}

func (p *WorkerProducer) Update(config Config) error {
	p.cancelCurrentWorkers()
	p.concurrency = config.Concurrency
	p.startConcurrentWorkers()
	return nil
}

func (p *WorkerProducer) startConcurrentWorkers() {
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < p.concurrency; i++ {
		go p.workerLoop(ctx)
	}
	p.cancelCurrentWorkers = cancel
}

func (p *WorkerProducer) workerLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		p.requester.Request()
	}
}
