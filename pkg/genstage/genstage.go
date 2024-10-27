package genstage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Producer produces data items.
type Producer struct {
	data chan int
}

func NewProducer(bufferSize int) *Producer {
	return &Producer{data: make(chan int, bufferSize)}
}

func (p *Producer) Produce(ctx context.Context) {
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			close(p.data)
			return
		case p.data <- i: // Producing data
			log.Printf("Produced: %d", i)
			time.Sleep(500 * time.Millisecond) // Simulate production delay
		}
	}
}

// Dispatcher manages demand and routes data to consumers.
type Dispatcher struct {
	producer *Producer
	demand   chan int
}

func NewDispatcher(producer *Producer) *Dispatcher {
	return &Dispatcher{
		producer: producer,
		demand:   make(chan int),
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, consumer *Consumer) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-consumer.request:
			select {
			case data, ok := <-d.producer.data:
				if !ok {
					return
				}
				consumer.data <- data
			case <-ctx.Done():
				return
			}
		}
	}
}

// Consumer requests and consumes data, controlling demand.
type Consumer struct {
	request chan struct{}
	data    chan int
	id      int
}

func NewConsumer(id int) *Consumer {
	return &Consumer{
		request: make(chan struct{}),
		data:    make(chan int),
		id:      id,
	}
}

func (c *Consumer) Consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case c.request <- struct{}{}: // Signal demand
			select {
			case data := <-c.data:
				fmt.Printf("Consumer %d consumed: %d\n", c.id, data)
				time.Sleep(1 * time.Second) // Simulate consumption delay
			case <-ctx.Done():
				return
			}
		}
	}
}
