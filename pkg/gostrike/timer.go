// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

import (
	"github.com/corrreia/gostrike/internal/runtime"
	"github.com/corrreia/gostrike/internal/scope"
)

// TimerFlags control timer behavior
type TimerFlags int

const (
	TimerRepeat      TimerFlags = 1 << iota // Timer repeats until stopped
	TimerNoMapChange                        // Timer survives map change
)

// Timer represents a scheduled callback
type Timer struct {
	id       uint64
	interval float64
	flags    TimerFlags
	callback func()
	stopped  bool
}

// Stop cancels the timer
func (t *Timer) Stop() {
	if t == nil || t.stopped {
		return
	}
	t.stopped = true
	runtime.StopTimer(t.id)
}

// IsStopped returns true if the timer has been stopped
func (t *Timer) IsStopped() bool {
	return t == nil || t.stopped
}

// GetInterval returns the timer's interval in seconds
func (t *Timer) GetInterval() float64 {
	if t == nil {
		return 0
	}
	return t.interval
}

// IsRepeating returns true if this is a repeating timer
func (t *Timer) IsRepeating() bool {
	if t == nil {
		return false
	}
	return t.flags&TimerRepeat != 0
}

// CreateTimer schedules a callback after a delay (one-shot)
func CreateTimer(delay float64, callback func()) *Timer {
	return CreateTimerWithFlags(delay, 0, callback)
}

// CreateRepeatingTimer schedules a repeating callback
func CreateRepeatingTimer(interval float64, callback func()) *Timer {
	return CreateTimerWithFlags(interval, TimerRepeat, callback)
}

// CreateTimerWithFlags creates a timer with specific flags
func CreateTimerWithFlags(interval float64, flags TimerFlags, callback func()) *Timer {
	if callback == nil || interval <= 0 {
		return nil
	}

	timer := &Timer{
		interval: interval,
		flags:    flags,
		callback: callback,
		stopped:  false,
	}

	timer.id = runtime.CreateTimer(interval, flags&TimerRepeat != 0, func() {
		if !timer.stopped {
			callback()
		}
	})

	if s := scope.GetActive(); s != nil {
		s.TrackTimer(timer.id)
	}

	return timer
}

// After is a convenience function that runs a callback after a delay
func After(delay float64, callback func()) *Timer {
	return CreateTimer(delay, callback)
}

// Every is a convenience function that runs a callback repeatedly
func Every(interval float64, callback func()) *Timer {
	return CreateRepeatingTimer(interval, callback)
}
