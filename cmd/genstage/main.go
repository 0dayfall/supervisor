package main

import (
	"context"
	"supervisor/pkg/genstage"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Define a data generation function for the producer
	generateData := func(i int) int {
		return i * 2 // Example data: doubles the index
	}

	// Specify the type when creating producer, dispatcher, and consumers
	producer := genstage.NewProducer[int](10, generateData, 500*time.Millisecond) // Production rate added
	dispatcher := genstage.NewDispatcher[int](producer)
	consumer1 := genstage.NewConsumer[int](1, func(data int) {
		// Define what each consumer does with the data
		println("Consumer 1 processed:", data)
	}, 1*time.Second) // Consumption rate added
	consumer2 := genstage.NewConsumer[int](2, func(data int) {
		println("Consumer 2 processed:", data)
	}, 1*time.Second) // Consumption rate added

	// Start producer
	go producer.Produce(ctx)

	// Start consumers
	go consumer1.Consume(ctx)
	go consumer2.Consume(ctx)

	// Start dispatcher
	go dispatcher.Dispatch(ctx, consumer1)
	go dispatcher.Dispatch(ctx, consumer2)

	// Run for a few seconds
	time.Sleep(10 * time.Second)
}
