package manytoone

import (
	"context"
	"fmt"
	"supervisor/pkg/manytoone"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Define how to generate data in the producer (for example, GeoJSON)
	generateData := func(i int) map[string]interface{} {
		return map[string]interface{}{
			"part": fmt.Sprintf("part%d", i%2), // Simulate alternating parts
			"data": i * 10,                     // Example data
		}
	}

	producer1 := manytoone.NewProducer[map[string]interface{}](1, 10)
	producer2 := manytoone.NewProducer[map[string]interface{}](2, 10)

	// Start producers with a data generation function
	go producer1.Produce(ctx, generateData)
	go producer2.Produce(ctx, generateData)

	// Create dispatcher and add producers
	dispatcher := manytoone.NewDispatcher([]*manytoone.Producer[map[string]interface{}]{producer1, producer2})

	// Define callback for processing the stitched data
	processCallback := func(data map[string]map[string]interface{}) {
		fmt.Printf("Processed data: %v\n", data)
	}

	// Create consumer with required parts and processing callback
	consumer := manytoone.NewConsumer[map[string]interface{}](1, 2*time.Second, []string{"part0", "part1"}, processCallback)

	// Start consumer
	go consumer.Consume(ctx)

	// Start dispatcher
	go dispatcher.Dispatch(ctx, consumer)

	// Query state periodically
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fmt.Printf("Producer 1 State: %s\n", producer1.GetState())
			fmt.Printf("Producer 2 State: %s\n", producer2.GetState())
			fmt.Printf("Consumer State: %s\n", consumer.GetState())
		case <-ctx.Done():
			return
		}
	}
}
