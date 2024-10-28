package manytoone

import (
	"context"
	"log"
	"sync"
	"time"
)

// Consumer receives data and passes it to the Stitcher for aggregation.
type Consumer[T any] struct {
	id        int
	request   chan struct{}
	data      chan T
	timeout   time.Duration
	stitcher  *Stitcher[T]
	parseData func(T) (string, T)
	state     string
	mu        sync.RWMutex
}

// NewConsumer creates a new consumer with a given timeout and parseData function.
func NewConsumer[T any](id int, timeout time.Duration, stitcher *Stitcher[T], parseData func(T) (string, T)) *Consumer[T] {
	return &Consumer[T]{
		id:        id,
		request:   make(chan struct{}),
		data:      make(chan T),
		timeout:   timeout,
		stitcher:  stitcher,
		parseData: parseData,
		state:     "idle",
	}
}

// SetState updates the state of the consumer.
func (c *Consumer[T]) SetState(state string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

// GetState retrieves the current state of the consumer.
func (c *Consumer[T]) GetState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Consume requests and processes data with a timeout.
func (c *Consumer[T]) Consume(ctx context.Context) {
	c.SetState("active")
	for {
		select {
		case <-ctx.Done():
			c.SetState("stopped")
			return
		case c.request <- struct{}{}:
			timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			select {
			case partData := <-c.data:
				// Use the parseData function to extract part name and data
				partName, data := c.parseData(partData)
				c.stitcher.AddPart(partName, data) // Pass data to stitcher
			case <-timeoutCtx.Done():
				if timeoutCtx.Err() == context.DeadlineExceeded {
					log.Printf("Consumer %d timeout exceeded, no data received", c.id)
					c.SetState("timeout")
				}
			}
		}
	}
}
