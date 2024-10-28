package manytoone

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestProducerBasicFunctionality checks if the producer can generate data items and update its state.
func TestProducerBasicFunctionality(t *testing.T) {
	generateData := func() int { return 42 }
	producer := NewProducer[int](1, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go producer.Produce(ctx, generateData)

	// Allow some time for the producer to generate data
	time.Sleep(100 * time.Millisecond)

	// Verify the producer generated data
	select {
	case item := <-producer.data:
		if item != 42 {
			t.Errorf("expected generated data to be 42, got %d", item)
		}
	default:
		t.Error("expected producer to generate data but got none")
	}

	// Check that the producer state is set to "active"
	if state := producer.GetState(); state != "active" {
		t.Errorf("expected producer state to be 'active', got %s", state)
	}
}

// TestDispatcherRelay checks that the dispatcher relays data from a producer to a consumer.
func TestDispatcherRelay(t *testing.T) {
	// Data generation function for the producer
	generateData := func() int { return 99 }
	producer := NewProducer[int](1, 5)
	dispatcher := NewDispatcher([]*Producer[int]{producer})

	// Mutex and slice to capture processed data from the consumer
	var processedData []int
	var mu sync.Mutex

	// Stitcher and consumer setup
	stitcher := NewStitcher([]string{"part1"}, func(data map[string]int, missing []string) {})
	consumer := NewConsumer[int](1, 100*time.Millisecond, stitcher, func(data int) (string, int) {
		return "part1", data
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go producer.Produce(ctx, generateData)
	go dispatcher.Dispatch(ctx, consumer)
	go consumer.Consume(ctx)

	// Allow time for the data to flow
	time.Sleep(200 * time.Millisecond)

	// Check that data was processed by the consumer
	mu.Lock()
	defer mu.Unlock()
	if len(processedData) == 0 || processedData[0] != 99 {
		t.Errorf("expected processed data to contain 99, but got %v", processedData)
	}
}

// TestConsumerTimeout verifies that the consumer times out if no data is received.
func TestConsumerTimeout(t *testing.T) {
	var timeoutTriggered bool
	stitcher := NewStitcher([]string{"part1"}, func(data map[string]int, missing []string) {
		timeoutTriggered = true
	})

	consumer := NewConsumer[int](1, 100*time.Millisecond, stitcher, func(data int) (string, int) {
		return "part1", data
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Consume(ctx)

	// Allow time for the timeout to be triggered
	time.Sleep(200 * time.Millisecond)

	if consumer.GetState() != "timeout" {
		t.Errorf("expected consumer state to be 'timeout', got %s", consumer.GetState())
	}

	if !timeoutTriggered {
		t.Error("expected timeout to trigger, but it did not")
	}
}

// TestStitcherAggregation verifies that the Stitcher correctly aggregates parts.
func TestStitcherAggregation(t *testing.T) {
	var mu sync.Mutex
	collectedData := make(map[string]int)
	var missingParts []string

	stitcher := NewStitcher([]string{"part1", "part2"}, func(data map[string]int, missing []string) {
		mu.Lock()
		defer mu.Unlock()
		collectedData = data
		missingParts = missing
	})

	// Add part data to the stitcher
	stitcher.AddPart("part1", 100)

	mu.Lock()
	defer mu.Unlock()

	// Verify collected data and missing parts
	if len(collectedData) != 1 || collectedData["part1"] != 100 {
		t.Errorf("expected collectedData to contain part1 with value 100, got %v", collectedData)
	}

	if len(missingParts) != 1 || missingParts[0] != "part2" {
		t.Errorf("expected missingParts to contain 'part2', got %v", missingParts)
	}
}

// TestConsumerProcessesData checks that the consumer processes data correctly.
func TestConsumerProcessesData(t *testing.T) {
	var receivedData []int
	var mu sync.Mutex

	stitcher := NewStitcher([]string{"part1"}, func(data map[string]int, missing []string) {})
	consumer := NewConsumer[int](1, 100*time.Millisecond, stitcher, func(data int) (string, int) {
		return "part1", data
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Consume(ctx)

	// Send data to the consumer
	consumer.request <- struct{}{}
	consumer.data <- 7

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(receivedData) != 1 || receivedData[0] != 7 {
		t.Errorf("expected consumer to process data [7], but got %v", receivedData)
	}
}

// TestCancellation checks that all components stop correctly when context is canceled.
func TestCancellation(t *testing.T) {
	generateData := func() int { return 42 }

	producer := NewProducer[int](1, 5)
	stitcher := NewStitcher([]string{"part1"}, func(data map[string]int, missing []string) {})
	consumer := NewConsumer[int](1, 100*time.Millisecond, stitcher, func(data int) (string, int) {
		return "part1", data
	})
	dispatcher := NewDispatcher([]*Producer[int]{producer})

	ctx, cancel := context.WithCancel(context.Background())

	go producer.Produce(ctx, generateData)
	go dispatcher.Dispatch(ctx, consumer)
	go consumer.Consume(ctx)

	time.Sleep(100 * time.Millisecond)

	// Cancel the context to stop all components
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify that each component's state is "stopped"
	if state := producer.GetState(); state != "stopped" {
		t.Errorf("expected producer state to be 'stopped', got %s", state)
	}
	if state := consumer.GetState(); state != "stopped" && state != "timeout" {
		t.Errorf("expected consumer state to be 'stopped' or 'timeout', got %s", state)
	}
}
