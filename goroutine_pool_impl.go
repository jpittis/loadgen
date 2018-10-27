package main

import (
	"context"
	"time"
)

// One Goroutine per request. Concurrency is ignored.
func NewGoroutinePoolProducer(requester Requester) Producer {
	p := &GoroutinePoolProducer{
		requester: requester,
		rps:       0,
	}
	return p
}

type GoroutinePoolProducer struct {
	requester      Requester
	rps            int
	cancelTickLoop func()
}

func (p *GoroutinePoolProducer) Start() error {
	p.startTickLoop()
	return nil
}

func (p *GoroutinePoolProducer) Stop() error {
	p.cancelTickLoop()
	return nil
}

func (p *GoroutinePoolProducer) Update(config Config) error {
	p.cancelTickLoop()
	p.rps = config.RPS
	p.startTickLoop()
	return nil
}

func (p *GoroutinePoolProducer) startTickLoop() error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelTickLoop = cancel
	if p.rps == 0 {
		return nil
	}

	pool := make(chan chan struct{}, 1000)

	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(p.rps))
		for {
			select {
			case <-ticker.C:
				select {
				case worker := <-pool:
					worker <- struct{}{}
				default:
					go p.spawnWorker(ctx, pool)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (p *GoroutinePoolProducer) spawnWorker(ctx context.Context, pool chan chan struct{}) {
	worker := make(chan struct{}, 1)
	for {
		p.requester.Request()
		pool <- worker
		select {
		case <-worker:
			// Perform another request.
		case <-ctx.Done():
			return
		}
	}
}
