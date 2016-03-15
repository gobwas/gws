package stat

import (
	"fmt"
	"sync"
)

type counter interface {
	add(v float64)
	flush() float64
}

type avg struct {
	mu    sync.Mutex
	count float64
	value float64
}

func (a *avg) add(v float64) {
	a.mu.Lock()
	{
		a.count++
		a.value += v
	}
	a.mu.Unlock()
}

func (a *avg) flush() (result float64) {
	a.mu.Lock()
	{
		if a.count == 0 {
			result = 0
		} else {
			result = a.value / a.count
		}

		a.count = 0
		a.value = 0
	}
	a.mu.Unlock()
	return
}

type abs struct {
	mu    sync.Mutex
	value float64
}

func (c *abs) add(v float64) {
	c.mu.Lock()
	{
		c.value += v
	}
	c.mu.Unlock()
}

func (c *abs) flush() (result float64) {
	c.mu.Lock()
	{
		result = c.value
		c.value = 0
	}
	c.mu.Unlock()
	return
}

type per struct {
	mu       sync.Mutex
	value    float64
	interval float64
}

func (p *per) add(v float64) {
	p.mu.Lock()
	{
		p.value += v
	}
	p.mu.Unlock()
}

func (c *per) flush() (result float64) {
	c.mu.Lock()
	{
		if c.interval == 0 {
			result = 0
		} else {
			result = c.value / c.interval
		}

		c.value = 0
	}
	c.mu.Unlock()
	return
}

type stat struct {
	mu       sync.Mutex
	counters map[string]counter
}

func (s *stat) add(name string, c counter) (err error) {
	s.mu.Lock()
	{
		if _, ok := s.counters[name]; ok {
			err = fmt.Errorf("counter already exists: %q", name)
		} else {
			s.counters[name] = c
		}
	}
	s.mu.Unlock()
	return
}
func (s *stat) get(name string) (c counter, err error) {
	s.mu.Lock()
	{
		var ok bool
		if c, ok = s.counters[name]; !ok {
			err = fmt.Errorf("counter is not exists: %q", name)
		}
	}
	s.mu.Unlock()
	return
}
