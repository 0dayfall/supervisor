package supervisor

import (
	"context"
	"edbtest/supervisor/message"
	"log"
	"sync"
	"time"
)

type Supervisor struct {
	maxRestarts    int
	restartWindow  time.Duration
	backoff        time.Duration
	task           func(context.Context, chan message.Message) error
	mu             sync.Mutex
	activeRestarts int
	cancel         context.CancelFunc
	messageChan    chan message.Message
}

func NewSupervisor(task func(context.Context, chan message.Message) error, maxRestarts int, restartWindow, backoff time.Duration) *Supervisor {
	return &Supervisor{
		maxRestarts:    maxRestarts,
		restartWindow:  restartWindow,
		backoff:        backoff,
		task:           task,
		activeRestarts: 0,
		messageChan:    make(chan message.Message),
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

func (s *Supervisor) runWorker(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			s.handleRestart()
		}
	}()

	err := s.task(ctx, s.messageChan) // Pass the message channel to the worker
	if err != nil {
		log.Printf("Worker returned an error: %v", err)
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
}
