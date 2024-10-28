package genstage

import (
	"context"
	"sync"
)

/* demand-driven, pull-based flow control */

// Producer represents a data generator that feeds into the dispatcher.
type Producer[T any] struct {
	data         chan T
	state        string
	mu           sync.RWMutex
	generateFunc func() T
}

// NewProducer creates a new producer with a specified buffer size and generation function.
func NewProducer[T any](bufferSize int, generateFunc func() T) *Producer[T] {
	return &Producer[T]{
		data:         make(chan T, bufferSize),
		state:        "idle",
		generateFunc: generateFunc,
	}
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

// Produce generates data items by calling the provided generation function.
func (p *Producer[T]) Produce(ctx context.Context) {
	p.SetState("active")
	defer func() {
		p.SetState("stopped")
		close(p.data)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case p.data <- p.generateFunc():
		}
	}
}

// Dispatcher coordinates data flow from producer to consumer.
type Dispatcher[T any] struct {
	producer *Producer[T]
	state    string
	mu       sync.RWMutex
}

// NewDispatcher creates a dispatcher for the given producer.
func NewDispatcher[T any](producer *Producer[T]) *Dispatcher[T] {
	return &Dispatcher[T]{
		producer: producer,
		state:    "idle",
	}
}

// SetState updates the state of the dispatcher.
func (d *Dispatcher[T]) SetState(state string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.state = state
}

// GetState retrieves the current state of the dispatcher.
func (d *Dispatcher[T]) GetState() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

// Dispatch sends data from producer to consumer on demand.
func (d *Dispatcher[T]) Dispatch(ctx context.Context, consumer *Consumer[T]) {
	d.SetState("active")
	for {
		select {
		case <-ctx.Done():
			d.SetState("stopped")
			return
		case <-consumer.request:
			select {
			case data, ok := <-d.producer.data:
				if !ok {
					return
				}
				consumer.data <- data
			case <-ctx.Done():
				d.SetState("stopped")
				return
			}
		}
	}
}

// Consumer represents a data consumer that processes items on demand.
type Consumer[T any] struct {
	request     chan struct{}
	data        chan T
	id          int
	state       string
	mu          sync.RWMutex
	processFunc func(T)
}

// NewConsumer creates a new consumer with a processing function and ID.
func NewConsumer[T any](id int, processFunc func(T)) *Consumer[T] {
	return &Consumer[T]{
		request:     make(chan struct{}),
		data:        make(chan T),
		id:          id,
		state:       "idle",
		processFunc: processFunc,
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

// Consume processes data items received from the dispatcher.
func (c *Consumer[T]) Consume(ctx context.Context) {
	c.SetState("active")
	for {
		select {
		case <-ctx.Done():
			c.SetState("stopped")
			return
		case c.request <- struct{}{}:
			select {
			case data := <-c.data:
				c.processFunc(data)
			case <-ctx.Done():
				c.SetState("stopped")
				return
			}
		}
	}
}
