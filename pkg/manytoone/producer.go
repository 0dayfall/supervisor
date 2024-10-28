package manytoone

import (
	"context"
	"sync"
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

// Produce continuously generates data using a custom generation function.
func (p *Producer[T]) Produce(ctx context.Context, generateData func() T) {
	p.SetState("active")
	defer func() {
		p.SetState("stopped")
		close(p.data)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case p.data <- generateData(): // Generate data without an index
		}
	}
}
