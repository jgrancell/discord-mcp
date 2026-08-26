package discord_test

import (
	"sync"
	"time"
)

// fakeClock is an injected clock: Now() is fixed unless advanced, and Sleep
// records the requested duration instead of blocking — per
// engineering:test-patterns resources/determinism.md, no real sleeping.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) Sleep(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sleeps = append(f.sleeps, d)
	f.now = f.now.Add(d)
}

func (f *fakeClock) Sleeps() []time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]time.Duration, len(f.sleeps))
	copy(out, f.sleeps)
	return out
}

// zeroJitter removes jitter from a test so sleep durations are asserted
// exactly rather than within a range.
func zeroJitter() time.Duration { return 0 }
