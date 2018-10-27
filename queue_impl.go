package main

import (
	"context"
	"time"
)

// One Goroutine per request. Concurrency is ignored.
func NewQueueProducer(requester Requester) Producer {
	p := &QueueProducer{
		requester: requester,
		rps:       0,
		queue:     make(chan struct{}, 1000),
	}
	return p
}

type QueueProducer struct {
	requester      Requester
	rps            int
	cancelTickLoop func()
	cancelWorkers  func()
	queue          chan struct{}
}

func (p *QueueProducer) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelWorkers = cancel
	for i := 0; i < 10000; i++ {
		go p.spawnWorker(ctx)
	}

	p.startTickLoop()

	return nil
}

func (p *QueueProducer) Stop() error {
	p.cancelTickLoop()
	p.cancelWorkers()
	return nil
}

func (p *QueueProducer) Update(config Config) error {
	p.cancelTickLoop()
	p.rps = config.RPS
	p.startTickLoop()
	return nil
}

func (p *QueueProducer) startTickLoop() error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelTickLoop = cancel
	if p.rps == 0 {
		return nil
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				for i := 0; i < p.rps; i++ {
					p.queue <- struct{}{}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (p *QueueProducer) spawnWorker(ctx context.Context) {
	for {
		select {
		case <-p.queue:
			// Perform another request.
		case <-ctx.Done():
			return
		}
		p.requester.Request()
	}
}
