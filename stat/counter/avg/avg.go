package avg

import (
	"sync"
)

func New() *Avg {
	return &Avg{}
}

type Avg struct {
	mu    sync.Mutex
	count float64
	value float64
}

func (a *Avg) Add(v float64) {
	a.mu.Lock()
	{
		a.count++
		a.value += v
	}
	a.mu.Unlock()
}

func (a *Avg) Reset() {
	a.mu.Lock()
	{
		a.count = 0
		a.value = 0
	}
	a.mu.Unlock()
}

func (a *Avg) Flush() (result float64) {
	a.mu.Lock()
	{
		if a.count == 0 {
			result = 0
		} else {
			result = a.value / a.count
		}
	}
	a.mu.Unlock()
	return
}

func (a *Avg) Kind() string {
	return "avg"
}
