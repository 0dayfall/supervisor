package supervisor

import "time"

type SupervisorStatus struct {
	ActiveRestarts int
	MaxRestarts    int
	RestartWindow  time.Duration
	Backoff        time.Duration
	Running        bool
}

func (s *Supervisor) Status() SupervisorStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return SupervisorStatus{
		ActiveRestarts: s.activeRestarts,
		MaxRestarts:    s.maxRestarts,
		RestartWindow:  s.restartWindow,
		Backoff:        s.backoff,
		Running:        s.cancel != nil,
	}
}
