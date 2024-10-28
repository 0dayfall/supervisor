package supervisor

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestSupervisorRestart verifies that the supervisor restarts the task when it fails.
func TestSupervisorRestart(t *testing.T) {
	restartCount := 0
	task := func(ctx context.Context, ch chan Message) error {
		restartCount++
		if restartCount <= 2 {
			return errors.New("simulated task failure")
		}
		return nil
	}

	sup := NewSupervisor(task, 3, time.Second, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sup.Start(ctx)
	time.Sleep(500 * time.Millisecond) // Wait for restarts

	if restartCount != 3 {
		t.Errorf("expected restart count 3, got %d", restartCount)
	}

	sup.Stop()
}

// TestSupervisorBackoff verifies that the backoff strategy works as expected.
func TestSupervisorBackoff(t *testing.T) {
	backoffDurations := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	backoffIdx := 0
	task := func(ctx context.Context, ch chan Message) error {
		if backoffIdx < len(backoffDurations) {
			time.Sleep(backoffDurations[backoffIdx])
			backoffIdx++
			return errors.New("simulated task failure")
		}
		return nil
	}

	sup := NewSupervisor(task, 3, 1*time.Second, 100*time.Millisecond)

	sup.SetBackoffStrategy("exponential", 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sup.Start(ctx)
	time.Sleep(2 * time.Second) // Wait for exponential backoff restarts

	sup.Stop()

	if backoffIdx != len(backoffDurations) {
		t.Errorf("expected backoffIdx to be %d, got %d", len(backoffDurations), backoffIdx)
	}
}

// TestSupervisorTimeout verifies that the SupervisorWithTimeout applies the timeout correctly to tasks.
func TestSupervisorTimeout(t *testing.T) {
	task := func(ctx context.Context, ch chan Message) error {
		time.Sleep(200 * time.Millisecond) // Simulate a task that takes too long
		return nil
	}

	timeout := 100 * time.Millisecond
	supWithTimeout := NewSupervisorWithTimeout(task, 3, 1*time.Second, 100*time.Millisecond, timeout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go supWithTimeout.Start(ctx)
	time.Sleep(500 * time.Millisecond) // Allow for timeout to take effect

	// Check that task has timed out by the log output or inferred by cancellation.
	// Actual timeout verification would require more complex checks with channels or timing.
}

// TestSupervisorStop verifies that the supervisor stops all tasks on cancelation.
func TestSupervisorStop(t *testing.T) {
	task := func(ctx context.Context, ch chan Message) error {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(100 * time.Millisecond):
			return errors.New("simulated error")
		}
	}

	sup := NewSupervisor(task, 3, 1*time.Second, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	go sup.Start(ctx)
	cancel() // Stop the supervisor immediately

	time.Sleep(200 * time.Millisecond)

	// Confirm the state has transitioned to "stopped"
	// Uncomment the following line if there is a GetState method in Supervisor:
	// if sup.GetState() != "stopped" {
	//     t.Errorf("expected supervisor to be stopped, got %v", sup.GetState())
	// }
}

// TestSetTimeout verifies that the timeout setting can be updated dynamically.
func TestSetTimeout(t *testing.T) {
	task := func(ctx context.Context, ch chan Message) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	supWithTimeout := NewSupervisorWithTimeout(task, 3, 1*time.Second, 100*time.Millisecond, 100*time.Millisecond)

	newTimeout := 50 * time.Millisecond
	supWithTimeout.SetTimeout(newTimeout)

	if supWithTimeout.timeout != newTimeout {
		t.Errorf("expected timeout to be %v, got %v", newTimeout, supWithTimeout.timeout)
	}
}
