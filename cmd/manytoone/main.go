package manytoone

import (
	"context"
	"encoding/json"
	"fmt"
	"supervisor/pkg/manytoone"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Define how to generate data in the producer (for example, GeoJSON)
	generateData := func(i int) interface{} {
		return map[string]interface{}{
			"part": "coordinates",
			"data": map[string]float64{"lat": 12.34, "lon": 56.78},
		}
	}

	producer1 := manytoone.NewProducer(1, 10)
	producer2 := manytoone.NewProducer(2, 10)

	// Start producers with a data generation function
	go producer1.Produce(ctx, generateData)
	go producer2.Produce(ctx, generateData)

	// Create dispatcher and add producers
	dispatcher := manytoone.NewDispatcher([]*manytoone.Producer{producer1, producer2})

	// Define callback for processing the stitched data
	processCallback := func(data map[string]interface{}) {
		jsonData, _ := json.Marshal(data)
		fmt.Printf("Processed JSON data: %s\n", jsonData)
	}

	// Create consumer with required parts and processing callback
	consumer := manytoone.NewConsumer(1, 2*time.Second, []string{"coordinates"}, processCallback)

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
