package abs

import (
	"sync"
)

func New() *Abs {
	return &Abs{}
}

type Abs struct {
	mu    sync.Mutex
	value float64
}

func (a *Abs) Add(v float64) {
	a.mu.Lock()
	{
		a.value += v
	}
	a.mu.Unlock()
}

func (a *Abs) Reset() {
	a.mu.Lock()
	{
		a.value = 0
	}
	a.mu.Unlock()
}

func (a *Abs) Flush() (result float64) {
	a.mu.Lock()
	{
		result = a.value
	}
	a.mu.Unlock()
	return
}

func (a *Abs) Kind() string {
	return "abs"
}
