package session

import (
	"sync"
	"time"
)

// Status represents the current state of a session.
type Status string

const (
	StatusIdle     Status = "idle"      // initial, triggers ignored
	StatusArmed    Status = "armed"     // waiting for first trigger
	StatusRunning  Status = "running"   // clock ticking, triggers recorded
	StatusPaused   Status = "paused"    // clock frozen, triggers ignored
	StatusReArmed  Status = "re-armed"  // waiting for car to pass before resuming
	StatusStopped  Status = "stopped"   // finished, results saved
)

type State struct {
	Status     Status    `json:"status"`
	LapNumber  int       `json:"lapNumber"`
	LastLap    float64   `json:"lastLap"`
	BestLap    float64   `json:"bestLap"`
	TotalTime  float64   `json:"totalTime"`
	LapHistory []float64 `json:"lapHistory"`
}

type Session struct {
	mu          sync.Mutex
	status      Status
	laps        []time.Duration
	startTime   time.Time
	lastTrigger time.Time
	stopTime    time.Time
	pauseStart  time.Time
}

func New() *Session {
	return &Session{status: StatusIdle}
}

// Arm enables trigger detection. Works from idle or paused.
func (s *Session) Arm() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.status {
	case StatusIdle:
		s.status = StatusArmed
	case StatusPaused:
		// Wait for the car to pass before resuming — keep timers frozen
		s.status = StatusReArmed
	}
	return s.buildState(time.Now())
}

// Pause freezes the clock and stops accepting triggers. Only from running.
func (s *Session) Pause() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == StatusRunning {
		s.pauseStart = time.Now()
		s.status = StatusPaused
	}
	return s.buildState(time.Now())
}

// ResetTime removes the incomplete current lap from the race timer while keeping
// completed laps. Session stays paused; both current lap and race timer show 0 / sum(laps).
// Only works when paused.
func (s *Session) ResetTime() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == StatusPaused {
		// Sum of completed laps
		var completed time.Duration
		for _, d := range s.laps {
			completed += d
		}
		// Anchor startTime so that totalTime = completed (incomplete lap removed)
		s.startTime = s.pauseStart.Add(-completed)
		// Anchor lastTrigger to pauseStart so next lap starts fresh on resume
		s.lastTrigger = s.pauseStart
	}
	return s.buildState(time.Now())
}

// Stop freezes and marks for saving. Works from running or paused.
func (s *Session) Stop() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == StatusRunning || s.status == StatusPaused || s.status == StatusReArmed {
		if s.status == StatusPaused || s.status == StatusReArmed {
			// Resolve paused time before stopping
			paused := time.Since(s.pauseStart)
			s.startTime = s.startTime.Add(paused)
			s.lastTrigger = s.lastTrigger.Add(paused)
			s.pauseStart = time.Time{}
		}
		s.stopTime = time.Now()
		s.status = StatusStopped
	}
	return s.buildState(time.Now())
}

// RecordTrigger records a lap. Only accepted when armed or running.
func (s *Session) RecordTrigger() State {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	switch s.status {
	case StatusArmed:
		// First trigger: start the clock
		s.startTime = now
		s.lastTrigger = now
		s.status = StatusRunning
	case StatusReArmed:
		// Car passed: shift race timer by pause duration, start a fresh lap from now
		paused := now.Sub(s.pauseStart)
		s.startTime = s.startTime.Add(paused)
		s.lastTrigger = now
		s.pauseStart = time.Time{}
		s.status = StatusRunning
	case StatusRunning:
		lap := now.Sub(s.lastTrigger)
		s.laps = append(s.laps, lap)
		s.lastTrigger = now
	default:
		// idle, paused, stopped — ignore
	}

	return s.buildState(now)
}

func (s *Session) CurrentState() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buildState(time.Now())
}

func (s *Session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = StatusIdle
	s.laps = nil
	s.startTime = time.Time{}
	s.lastTrigger = time.Time{}
	s.stopTime = time.Time{}
	s.pauseStart = time.Time{}
}

// Snapshot returns raw data for persistence. ok=false if nothing to save.
func (s *Session) Snapshot() (startedAt, stoppedAt time.Time, laps []time.Duration, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startTime.IsZero() || len(s.laps) == 0 {
		return
	}
	end := s.stopTime
	if end.IsZero() {
		end = time.Now()
	}
	return s.startTime, end, append([]time.Duration(nil), s.laps...), true
}

// buildState must be called with mu held.
func (s *Session) buildState(now time.Time) State {
	st := State{
		Status:     s.status,
		LapHistory: []float64{},
	}

	if s.startTime.IsZero() {
		return st
	}

	// Compute total elapsed time (excluding current pause if any)
	var elapsed time.Time
	switch s.status {
	case StatusStopped:
		elapsed = s.stopTime
	case StatusPaused, StatusReArmed:
		elapsed = s.pauseStart
	default:
		elapsed = now
	}
	st.TotalTime = elapsed.Sub(s.startTime).Seconds()

	st.LapNumber = len(s.laps)
	st.LapHistory = make([]float64, len(s.laps))

	var best time.Duration
	for i, d := range s.laps {
		st.LapHistory[i] = d.Seconds()
		if best == 0 || d < best {
			best = d
		}
	}
	if len(s.laps) > 0 {
		st.LastLap = s.laps[len(s.laps)-1].Seconds()
		st.BestLap = best.Seconds()
	}

	return st
}
