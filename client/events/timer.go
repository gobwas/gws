package events

import (
	"sync"
	"time"
)

type timer struct {
	mu sync.Mutex

	cb func()

	timeout  time.Duration
	nextTime time.Time

	restart bool
	active  bool
}

func NewTimer(d time.Duration) Timer {
	return &timer{timeout: d}
}

func NewTicker(d time.Duration) Timer {
	return &timer{timeout: d, restart: true}
}

func (t *timer) Start(now time.Time) {
	t.mu.Lock()
	t.nextTime = now.Add(t.timeout)
	t.mu.Unlock()
}

func (t *timer) Stop() {
	t.mu.Lock()
	t.active = false
	t.mu.Unlock()
}

func (t *timer) Fire() {
	t.mu.Lock()
	{
		t.cb()
		if t.restart {
			t.nextTime.Add(t.timeout)
		}
		t.active = t.restart
	}
	t.mu.Unlock()
}

func (t *timer) Before(p time.Time) (before bool) {
	t.mu.Lock()
	{
		before = t.nextTime.Before(p)
	}
	t.mu.Unlock()
	return
}

func (t *timer) IsActive() (active bool) {
	t.mu.Lock()
	active = t.active
	t.mu.Unlock()
	return
}

func (t *timer) Equal(h Handler) bool {
	if b, ok := h.(*timer); ok {
		return b == t
	}
	return false
}
