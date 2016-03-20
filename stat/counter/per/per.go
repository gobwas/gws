package per

import (
	"sync"
	"time"
)

func New(d time.Duration) *Per {
	return &Per{
		interval: d,
		stamp:    time.Now(),
	}
}

type Per struct {
	mu       sync.Mutex
	value    float64
	interval time.Duration
	stamp    time.Time
}

func (p *Per) Add(v float64) {
	p.mu.Lock()
	{
		p.value += v
	}
	p.mu.Unlock()
}

func (p *Per) Flush() (result float64) {
	p.mu.Lock()
	{
		k := float64(time.Since(p.stamp)) / float64(p.interval)
		if k == 0 {
			result = 0
		} else {
			result = p.value / k
		}

		p.value = 0
		p.stamp = time.Now()
	}
	p.mu.Unlock()
	return
}

func (p *Per) Kind() string {
	return "per " + p.interval.String()
}
