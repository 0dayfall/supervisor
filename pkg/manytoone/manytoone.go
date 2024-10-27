package manytoone

import (
	"context"
	"log"
	"sync"
	"time"
)

// Producer generates data for the consumer.
type Producer struct {
	id    int
	data  chan interface{}
	state string
	mu    sync.RWMutex
}

func NewProducer(id, bufferSize int) *Producer {
	return &Producer{id: id, data: make(chan interface{}, bufferSize), state: "idle"}
}

// SetState updates the state of the producer.
func (p *Producer) SetState(state string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

// GetState retrieves the current state of the producer.
func (p *Producer) GetState() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Produce method for the Producer to simulate data generation.
func (p *Producer) Produce(ctx context.Context, generateData func(int) interface{}) {
	p.SetState("active")
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			close(p.data)
			return
		case p.data <- generateData(i): // Use callback to generate data
		}
	}
}

// Dispatcher coordinates multiple producers and a single consumer.
type Dispatcher struct {
	producers []*Producer
}

// NewDispatcher creates a dispatcher for the given producers.
func NewDispatcher(producers []*Producer) *Dispatcher {
	return &Dispatcher{
		producers: producers,
	}
}

// Dispatch sends data from producers to the consumer on demand.
func (d *Dispatcher) Dispatch(ctx context.Context, consumer *Consumer) {
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

// Consumer processes data received from producers.
type Consumer struct {
	id              int
	request         chan struct{}
	data            chan interface{}
	timeout         time.Duration
	stitchingLock   sync.Mutex
	stitchData      map[string]interface{}
	processCallback func(map[string]interface{}) // Callback for stitching logic
	requiredParts   []string                     // List of required parts for stitching
	state           string                       // Track consumer state
	mu              sync.RWMutex
}

// NewConsumer creates a new consumer with a given timeout and callback for processing.
func NewConsumer(id int, timeout time.Duration, requiredParts []string, processCallback func(map[string]interface{})) *Consumer {
	return &Consumer{
		id:              id,
		request:         make(chan struct{}),
		data:            make(chan interface{}),
		timeout:         timeout,
		stitchData:      make(map[string]interface{}),
		processCallback: processCallback,
		requiredParts:   requiredParts,
		state:           "idle",
	}
}

func (c *Consumer) SetState(state string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

// GetState retrieves the current state of the consumer.
func (c *Consumer) GetState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// AggregateAndProcess handles data parts and processes them when all required parts are available.
func (c *Consumer) AggregateAndProcess(partName string, data interface{}) {
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
		c.stitchData = make(map[string]interface{}) // Reset for the next aggregation
	}
}

// Consume requests and processes data with a timeout.
func (c *Consumer) Consume(ctx context.Context) {
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
				// Here we assume that the part name is provided along with data.
				// This can be modified based on how the data format needs to be handled.
				if part, ok := partData.(map[string]interface{}); ok {
					if partName, found := part["part"].(string); found {
						c.AggregateAndProcess(partName, part["data"])
					}
				}
			case <-timeoutCtx.Done():
				if timeoutCtx.Err() == context.DeadlineExceeded {
					log.Printf("Consumer %d timeout exceeded, no data received", c.id)
					c.SetState("timeout")
				}
			}
		}
	}
}
