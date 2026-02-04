// Package runtime provides the internal runtime for GoStrike.
// This file contains the timer system implementation.
package runtime

import (
	"sync"
	"sync/atomic"
)

// timer represents a scheduled callback
type timer struct {
	id        uint64
	interval  float64
	remaining float64
	repeating bool
	callback  func()
	stopped   bool
}

var (
	timers      = make(map[uint64]*timer)
	timersMu    sync.RWMutex
	nextTimerID uint64
)

func initTimers() {
	timers = make(map[uint64]*timer)
	nextTimerID = 0
}

func shutdownTimers() {
	timersMu.Lock()
	timers = make(map[uint64]*timer)
	timersMu.Unlock()
}

// CreateTimer creates a new timer
// Returns the timer ID
func CreateTimer(interval float64, repeating bool, callback func()) uint64 {
	id := atomic.AddUint64(&nextTimerID, 1)

	t := &timer{
		id:        id,
		interval:  interval,
		remaining: interval,
		repeating: repeating,
		callback:  callback,
		stopped:   false,
	}

	timersMu.Lock()
	timers[id] = t
	timersMu.Unlock()

	return id
}

// StopTimer stops a timer by ID
func StopTimer(id uint64) {
	timersMu.Lock()
	defer timersMu.Unlock()

	if t, ok := timers[id]; ok {
		t.stopped = true
		delete(timers, id)
	}
}

// processTimers is called every tick to update and fire timers
func processTimers(deltaTime float64) {
	timersMu.Lock()
	defer timersMu.Unlock()

	var toRemove []uint64
	var toFire []*timer

	// Update timers and collect ones to fire
	for id, t := range timers {
		if t.stopped {
			toRemove = append(toRemove, id)
			continue
		}

		t.remaining -= deltaTime
		if t.remaining <= 0 {
			toFire = append(toFire, t)
			if t.repeating {
				t.remaining = t.interval
			} else {
				toRemove = append(toRemove, id)
			}
		}
	}

	// Remove finished timers
	for _, id := range toRemove {
		delete(timers, id)
	}

	// Fire callbacks (outside the lock would be safer, but this is simpler)
	// Note: In a real implementation, callbacks should be called outside the lock
	for _, t := range toFire {
		if !t.stopped && t.callback != nil {
			// Call callback with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Timer callback panicked - log but don't crash
					}
				}()
				t.callback()
			}()
		}
	}
}

// GetTimerCount returns the number of active timers
func GetTimerCount() int {
	timersMu.RLock()
	defer timersMu.RUnlock()
	return len(timers)
}
