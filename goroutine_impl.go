package main

import (
	"context"
	"time"
)

// One Goroutine per request. Concurrency is ignored.
func NewGoroutineProducer(requester Requester) Producer {
	p := &GoroutineProducer{
		requester: requester,
		rps:       0,
	}
	return p
}

type GoroutineProducer struct {
	requester      Requester
	rps            int
	cancelTickLoop func()
}

func (p *GoroutineProducer) Start() error {
	p.startTickLoop()
	return nil
}

func (p *GoroutineProducer) Stop() error {
	p.cancelTickLoop()
	return nil
}

func (p *GoroutineProducer) Update(config Config) error {
	p.cancelTickLoop()
	p.rps = config.RPS
	p.startTickLoop()
	return nil
}

func (p *GoroutineProducer) startTickLoop() error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelTickLoop = cancel
	if p.rps == 0 {
		return nil
	}
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(p.rps))
		for {
			select {
			case <-ticker.C:
				go p.requester.Request()
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}
