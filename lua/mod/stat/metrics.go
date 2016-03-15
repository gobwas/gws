package stat

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"
)

type counter interface {
	add(v float64)
	flush() float64
	name() string
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

func (a *avg) name() string {
	return "avg"
}

type abs struct {
	mu    sync.Mutex
	value float64
}

func (a *abs) add(v float64) {
	a.mu.Lock()
	{
		a.value += v
	}
	a.mu.Unlock()
}

func (a *abs) flush() (result float64) {
	a.mu.Lock()
	{
		result = a.value
		a.value = 0
	}
	a.mu.Unlock()
	return
}

func (a *abs) name() string {
	return "abs"
}

type per struct {
	mu       sync.Mutex
	value    float64
	interval time.Duration
	stamp    time.Time
}

func (p *per) add(v float64) {
	p.mu.Lock()
	{
		p.value += v
	}
	p.mu.Unlock()
}

func (p *per) flush() (result float64) {
	p.mu.Lock()
	{
		k := float64(time.Since(p.stamp)) / float64(p.interval)
		if k == 0 {
			result = 0
		} else {
			result = p.value / k
		}

		p.value = 0
	}
	p.mu.Unlock()
	return
}

func (p *per) name() string {
	return "per " + p.interval.String()
}

type statistics struct {
	mu       sync.Mutex
	counters map[string]counter
}

func newStatistics() *statistics {
	return &statistics{
		counters: make(map[string]counter),
	}
}

const (
	titleKey   = "counter"
	titleValue = "value"
	titleKind  = "kind"
	tab        = "  "
)

func (s *statistics) pretty() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	buf := &bytes.Buffer{}

	maxKeyLen := len(titleKey)
	maxValueLen := len(titleValue)
	maxKindLen := len(titleKind)

	var pairs [][]string
	for id, counter := range s.counters {
		key := fmt.Sprintf("%s", id)
		value := fmt.Sprintf("%.3f", counter.flush())
		kind := fmt.Sprintf("%s", counter.name())

		pairs = append(pairs, []string{key, value, kind})

		if l := len(key); l > maxKeyLen {
			maxKeyLen = l
		}
		if l := len(value); l > maxValueLen {
			maxValueLen = l
		}
		if l := len(kind); l > maxKindLen {
			maxKindLen = l
		}
	}

	fmt.Fprint(buf, strings.Join(
		[]string{
			fmt.Sprintf("%-*s", maxKeyLen, titleKey),
			fmt.Sprintf("%-*s", maxValueLen, titleValue),
			fmt.Sprintf("%-*s", maxKindLen, titleKind),
		},
		tab,
	), "\n")

	buf.WriteString(strings.Repeat("-", maxKeyLen+maxValueLen+maxKindLen+len(tab)*2) + "\n")

	for _, p := range pairs {
		fmt.Fprint(buf, strings.Join(
			[]string{
				fmt.Sprintf("%-*s", maxKeyLen, p[0]),
				fmt.Sprintf("%-*s", maxValueLen, p[1]),
				fmt.Sprintf("%-*s", maxKindLen, p[2]),
			},
			tab,
		), "\n")
	}
	buf.WriteByte('\n')

	return buf.String()
}

func (s *statistics) add(name string, c counter) (err error) {
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
func (s *statistics) get(name string) (c counter, err error) {
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
