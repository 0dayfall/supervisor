package supervisor

import (
	"context"
	"log"
	"math"
	"sync"
	"time"
)

/* Supervise and restart a process */

type Message struct{}

type Supervisor struct {
	maxRestarts    int
	restartWindow  time.Duration
	backoff        time.Duration
	task           func(context.Context, chan Message) error
	mu             sync.Mutex
	activeRestarts int
	cancel         context.CancelFunc
	messageChan    chan Message
}

type SupervisorWithTimeout struct {
	*Supervisor
	timeout time.Duration
}

// NewSupervisor creates a new Supervisor instance.
func NewSupervisor(task func(context.Context, chan Message) error, maxRestarts int, restartWindow, backoff time.Duration) *Supervisor {
	return &Supervisor{
		maxRestarts:    maxRestarts,
		restartWindow:  restartWindow,
		backoff:        backoff,
		task:           task,
		activeRestarts: 0,
		messageChan:    make(chan Message),
	}
}

// NewSupervisorWithTimeout creates a SupervisorWithTimeout instance.
func NewSupervisorWithTimeout(task func(context.Context, chan Message) error, maxRestarts int, restartWindow, backoff, timeout time.Duration) *SupervisorWithTimeout {
	return &SupervisorWithTimeout{
		Supervisor: NewSupervisor(task, maxRestarts, restartWindow, backoff),
		timeout:    timeout,
	}
}

func (s *Supervisor) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	go s.supervise(ctx)
}

func (s *Supervisor) supervise(ctx context.Context) {
	restartTicker := time.NewTicker(s.restartWindow)
	defer restartTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Supervisor shutting down")
			return

		case <-restartTicker.C:
			s.mu.Lock()
			s.activeRestarts = 0
			s.mu.Unlock()

		default:
			s.runWorker(ctx)
		}
	}
}

func (s *SupervisorWithTimeout) supervise(ctx context.Context) {
	restartTicker := time.NewTicker(s.restartWindow)
	defer restartTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Supervisor with timeout shutting down")
			return

		case <-restartTicker.C:
			s.mu.Lock()
			s.activeRestarts = 0
			s.mu.Unlock()

		default:
			s.runWorkerWithTimeout(ctx)
		}
	}
}

func (s *Supervisor) runWorker(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			s.handleRestart()
		}
	}()

	err := s.task(ctx, s.messageChan)
	if err != nil {
		log.Printf("Worker returned an error: %v", err)
	}
}

func (s *SupervisorWithTimeout) runWorkerWithTimeout(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, s.timeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			s.handleRestart()
		}
	}()

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.task(ctx, s.messageChan)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log.Printf("Worker returned an error: %v", err)
			s.handleRestart()
		}
	case <-ctx.Done(): // Handle the timeout
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Worker timeout exceeded, interrupting task")
			s.handleRestart()
		}
	}
}

func (s *Supervisor) handleRestart() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeRestarts++
	if s.activeRestarts > s.maxRestarts {
		log.Println("Max restart limit reached, backing off")
		time.Sleep(s.backoff)
		s.activeRestarts = 0
	}
}

func (s *Supervisor) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	log.Println("Supervisor stopped")
}

func (s *SupervisorWithTimeout) SetTimeout(newTimeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("Timeout updated from %v to %v", s.timeout, newTimeout)
	s.timeout = newTimeout
}

func (s *Supervisor) SetBackoffStrategy(strategy string, multiplier float64) {
	switch strategy {
	case "exponential":
		s.backoff = time.Duration(float64(s.backoff) * math.Pow(2, multiplier))
	case "linear":
		s.backoff = time.Duration(float64(s.backoff) + multiplier)
	default:
		log.Println("Unknown backoff strategy, using default backoff")
	}
}
