package session

import (
	"testing"
	"time"
)

func TestSessionFlow(t *testing.T) {
	s := New()

	// Initial state: idle
	st := s.CurrentState()
	if st.Status != StatusIdle {
		t.Fatalf("expected idle, got %s", st.Status)
	}

	// Arm: move to armed, triggers now accepted
	st = s.Arm()
	if st.Status != StatusArmed {
		t.Fatalf("expected armed, got %s", st.Status)
	}

	// First trigger starts the clock
	st = s.RecordTrigger()
	if st.Status != StatusRunning {
		t.Fatalf("expected running, got %s", st.Status)
	}
	if st.LapNumber != 0 {
		t.Fatalf("expected lap 0, got %d", st.LapNumber)
	}

	time.Sleep(10 * time.Millisecond)
	st = s.RecordTrigger() // lap 1
	if st.LapNumber != 1 {
		t.Fatalf("expected lap 1, got %d", st.LapNumber)
	}

	// Pause freezes the clock
	st = s.Pause()
	if st.Status != StatusPaused {
		t.Fatalf("expected paused, got %s", st.Status)
	}
	frozen := st.TotalTime

	time.Sleep(20 * time.Millisecond)

	// Triggers ignored while paused
	s.RecordTrigger()
	st = s.CurrentState()
	if st.LapNumber != 1 {
		t.Fatal("trigger should be ignored while paused")
	}
	if st.TotalTime != frozen {
		t.Fatal("total time should be frozen while paused")
	}

	// Resume
	st = s.Arm()
	if st.Status != StatusRunning {
		t.Fatalf("expected running after resume, got %s", st.Status)
	}

	time.Sleep(10 * time.Millisecond)
	st = s.RecordTrigger() // lap 2
	if st.LapNumber != 2 {
		t.Fatalf("expected lap 2, got %d", st.LapNumber)
	}

	// Stop
	st = s.Stop()
	if st.Status != StatusStopped {
		t.Fatalf("expected stopped, got %s", st.Status)
	}

	// Reset clears everything
	s.Reset()
	st = s.CurrentState()
	if st.Status != StatusIdle {
		t.Fatalf("expected idle after reset, got %s", st.Status)
	}
}
