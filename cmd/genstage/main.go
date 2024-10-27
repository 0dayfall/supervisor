package genstage

import (
	"context"
	"supervisor/pkg/genstage"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := genstage.NewProducer(10)
	dispatcher := genstage.NewDispatcher(producer)
	consumer1 := genstage.NewConsumer(1)
	consumer2 := genstage.NewConsumer(2)

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
