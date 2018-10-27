package main

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Producer interface {
	Start() error
	Update(Config) error
	Stop() error
}

type Config struct {
	RPS         int
	Concurrency int
}

type Requester interface {
	// Request should block until the request is finished.
	Request()
	Update(time.Duration)
}

var (
	requestCollector = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "loadgen_requests_total",
			Help: "Total requests measured at transport.",
		},
	)

	latencyCollector = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:       "loadgen_request_latency",
			Help:       "The latency of requests.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
	)
)

func init() {
	prometheus.MustRegister(requestCollector)
	prometheus.MustRegister(latencyCollector)
	http.Handle("/metrics", prometheus.Handler())
}

var events = []Event{
	UpdateRequester(100 * time.Millisecond),
	UpdateProducer(Config{RPS: 10}),
	Sleep(60 * time.Second),
	UpdateProducer(Config{RPS: 100}),
	Sleep(60 * time.Second),
	UpdateProducer(Config{RPS: 1000}),
	Sleep(60 * time.Second),
	UpdateProducer(Config{RPS: 10000}),
	Sleep(60 * time.Second),
}

func main() {
	requester := NewBenchmarkRequester()
	runner := &EventRunner{
		Producer:  NewQueueProducer(requester),
		Requester: requester,
	}

	go func() {
		err := runner.Run(events)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Blocking and so that prometheus can scrape us for metrics.
	http.ListenAndServe(":8080", nil)
}

func NewBenchmarkRequester() Requester {
	return &BenchmarkRequester{latency: 0}
}

type BenchmarkRequester struct {
	latency time.Duration
}

func (r *BenchmarkRequester) Update(latency time.Duration) {
	r.latency = latency
}

func (r *BenchmarkRequester) Request() {
	// Measure before we inject latency to simulate measurement at the transport.
	requestCollector.Inc()
	// Insert fake latency.
	time.Sleep(r.latency)
	latencyCollector.Observe(float64(r.latency / time.Millisecond))
}

type EventRunner struct {
	Producer  Producer
	Requester Requester
}

func (r *EventRunner) Run(events []Event) error {
	err := r.Producer.Start()
	if err != nil {
		return err
	}
	for i, perform := range events {
		log.Printf("Ran event %d!", i)
		err := perform(r)
		if err != nil {
			return err
		}
	}
	return r.Producer.Stop()
}

type Event func(*EventRunner) error

func UpdateProducer(config Config) Event {
	return func(r *EventRunner) error {
		return r.Producer.Update(config)
	}
}

func UpdateRequester(latency time.Duration) Event {
	return func(r *EventRunner) error {
		r.Requester.Update(latency)
		return nil
	}
}

func Sleep(duration time.Duration) Event {
	return func(r *EventRunner) error {
		time.Sleep(duration)
		return nil
	}
}
