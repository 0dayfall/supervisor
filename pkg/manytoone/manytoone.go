package manytoone

import (
	"context"
	"log"
	"sync"
	"time"
)

// Producer generates data for the consumer with a customizable data type.
type Producer[T any] struct {
	id    int
	data  chan T
	state string
	mu    sync.RWMutex
}

// NewProducer creates a new producer with a specified buffer size.
func NewProducer[T any](id, bufferSize int) *Producer[T] {
	return &Producer[T]{id: id, data: make(chan T, bufferSize), state: "idle"}
}

// SetState updates the state of the producer.
func (p *Producer[T]) SetState(state string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

// GetState retrieves the current state of the producer.
func (p *Producer[T]) GetState() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Produce generates data using a custom generation function.
func (p *Producer[T]) Produce(ctx context.Context, generateData func(int) T) {
	p.SetState("active")
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			p.SetState("stopped")
			close(p.data)
			return
		case p.data <- generateData(i): // Use callback to generate data
		}
	}
}

// Dispatcher coordinates data flow from multiple producers to a single consumer.
type Dispatcher[T any] struct {
	producers []*Producer[T]
}

// NewDispatcher creates a dispatcher for the given producers.
func NewDispatcher[T any](producers []*Producer[T]) *Dispatcher[T] {
	return &Dispatcher[T]{producers: producers}
}

// Dispatch sends data from producers to the consumer on demand.
func (d *Dispatcher[T]) Dispatch(ctx context.Context, consumer *Consumer[T]) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-consumer.request:
			for _, producer := range d.producers {
				select {
				case data, ok := <-producer.data:
					if !ok {
						return
					}
					consumer.data <- data
					log.Printf("Dispatched data from Producer %d to Consumer", producer.id)
				case <-ctx.Done():
					return
				default:
				}
			}
		}
	}
}

// Consumer processes data received from producers with a customizable aggregation and processing logic.
type Consumer[T any] struct {
	id              int
	request         chan struct{}
	data            chan T
	timeout         time.Duration
	stitchingLock   sync.Mutex
	stitchData      map[string]T
	processCallback func(map[string]T) // Callback for stitching logic
	requiredParts   []string           // List of required parts for stitching
	state           string             // Track consumer state
	mu              sync.RWMutex
}

// NewConsumer creates a new consumer with a given timeout and processing callback.
func NewConsumer[T any](id int, timeout time.Duration, requiredParts []string, processCallback func(map[string]T)) *Consumer[T] {
	return &Consumer[T]{
		id:              id,
		request:         make(chan struct{}),
		data:            make(chan T),
		timeout:         timeout,
		stitchData:      make(map[string]T),
		processCallback: processCallback,
		requiredParts:   requiredParts,
		state:           "idle",
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

// AggregateAndProcess handles data parts and processes them when all required parts are available.
func (c *Consumer[T]) AggregateAndProcess(partName string, data T) {
	c.stitchingLock.Lock()
	defer c.stitchingLock.Unlock()

	// Store the part in stitchData
	c.stitchData[partName] = data

	// Check if all required parts are present
	allPartsPresent := true
	for _, part := range c.requiredParts {
		if _, exists := c.stitchData[part]; !exists {
			allPartsPresent = false
			break
		}
	}

	// Process if all parts are present
	if allPartsPresent {
		c.processCallback(c.stitchData)
		c.stitchData = make(map[string]T) // Reset for the next aggregation
	}
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
				// Here we assume part name and data are structured appropriately.
				// This could be customized depending on data structure.
				partName, data := c.parseData(partData) // Implement parseData based on your requirements
				c.AggregateAndProcess(partName, data)
			case <-timeoutCtx.Done():
				if timeoutCtx.Err() == context.DeadlineExceeded {
					log.Printf("Consumer %d timeout exceeded, no data received", c.id)
					c.SetState("timeout")
				}
			}
		}
	}
}

// parseData simulates parsing of data to get part name and data.
// In a real implementation, you'd need to define how part names are derived from data items.
func (c *Consumer[T]) parseData(data T) (string, T) {
	// Customize this method to extract part name from your data type
	// For example, if T is a map or struct, access fields to determine partName
	partName := "example_part" // Placeholder; update based on actual data structure
	return partName, data
}
