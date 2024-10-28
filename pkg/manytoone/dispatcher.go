package manytoone

import (
	"context"
	"log"
)

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
