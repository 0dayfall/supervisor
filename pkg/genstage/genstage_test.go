package genstage

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestGenStageFlow tests the entire flow from Producer -> Dispatcher -> Consumer
func TestGenStageFlow(t *testing.T) {
	// Define a simple data generation function for the producer
	generateFunc := func() int {
		return 42 // constant output for testing
	}

	// Define a simple processing function for the consumer
	var processedData []int
	var mu sync.Mutex
	processFunc := func(data int) {
		mu.Lock()
		defer mu.Unlock()
		processedData = append(processedData, data)
	}

	// Set up components
	producer := NewProducer[int](10, generateFunc)
	consumer := NewConsumer[int](1, processFunc)
	dispatcher := NewDispatcher(producer)

	// Create context for the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start components
	go producer.Produce(ctx)
	go dispatcher.Dispatch(ctx, consumer)
	go consumer.Consume(ctx)

	// Allow time for data to flow
	time.Sleep(500 * time.Millisecond)

	// Verify data was processed
	mu.Lock()
	defer mu.Unlock()
	if len(processedData) == 0 || processedData[0] != 42 {
		t.Errorf("expected processed data to contain 42, but got %v", processedData)
	}
}

// TestProducerState tests that the Producer transitions states correctly
func TestProducerState(t *testing.T) {
	generateFunc := func() int { return 1 }
	producer := NewProducer[int](5, generateFunc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go producer.Produce(ctx)

	// Allow a brief moment to enter "active" state
	time.Sleep(100 * time.Millisecond)
	if state := producer.GetState(); state != "active" {
		t.Errorf("expected producer state to be 'active', got %v", state)
	}

	// Cancel context and check for "stopped" state
	cancel()
	time.Sleep(100 * time.Millisecond)
	if state := producer.GetState(); state != "stopped" {
		t.Errorf("expected producer state to be 'stopped', got %v", state)
	}
}

// TestDispatcherState tests that the Dispatcher transitions states correctly
func TestDispatcherState(t *testing.T) {
	generateFunc := func() int { return 1 }
	producer := NewProducer[int](5, generateFunc)
	dispatcher := NewDispatcher(producer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Dummy consumer to satisfy dispatcher requirements
	consumer := NewConsumer[int](1, func(int) {})

	go dispatcher.Dispatch(ctx, consumer)

	// Allow time to transition state
	time.Sleep(100 * time.Millisecond)
	if state := dispatcher.GetState(); state != "active" {
		t.Errorf("expected dispatcher state to be 'active', got %v", state)
	}

	// Cancel context and check for "stopped" state
	cancel()
	time.Sleep(100 * time.Millisecond)
	if state := dispatcher.GetState(); state != "stopped" {
		t.Errorf("expected dispatcher state to be 'stopped', got %v", state)
	}
}

// TestConsumerState tests that the Consumer transitions states correctly
func TestConsumerState(t *testing.T) {
	consumer := NewConsumer[int](1, func(int) {})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Consume(ctx)

	// Allow time to transition state
	time.Sleep(100 * time.Millisecond)
	if state := consumer.GetState(); state != "active" {
		t.Errorf("expected consumer state to be 'active', got %v", state)
	}

	// Cancel context and check for "stopped" state
	cancel()
	time.Sleep(100 * time.Millisecond)
	if state := consumer.GetState(); state != "stopped" {
		t.Errorf("expected consumer state to be 'stopped', got %v", state)
	}
}

// TestConsumerProcessing verifies that the consumer processes data as expected.
func TestConsumerProcessing(t *testing.T) {
	var processed []int
	var mu sync.Mutex
	processFunc := func(data int) {
		mu.Lock()
		defer mu.Unlock()
		processed = append(processed, data)
	}

	consumer := NewConsumer[int](1, processFunc)
	consumer.request <- struct{}{} // Signal that consumer is ready

	// Send test data
	go func() {
		consumer.data <- 99
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer.Consume(ctx)

	// Allow processing time
	time.Sleep(100 * time.Millisecond)

	// Verify the processed data
	mu.Lock()
	defer mu.Unlock()
	if len(processed) == 0 || processed[0] != 99 {
		t.Errorf("expected processed data to be [99], but got %v", processed)
	}
}
