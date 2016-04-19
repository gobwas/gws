package events

import (
	"reflect"
	"sync"
	"time"
)

type EventKind int

const (
	EventCustom EventKind = iota
)

type Header struct {
	Kind EventKind
	Name string
}

type Event struct {
	Header Header
	Data   interface{}
}

type Pinger interface {
	IsActive() bool
}

type Handler interface {
	Fire()
	Equal(Handler) bool
}

type Listener interface {
	Pinger
	Handler
}

type Timer interface {
	Pinger
	Handler
	Start(time.Time)
	Stop()
	Before(time.Time) bool
}

type Loop struct {
	mu sync.RWMutex

	now       time.Time
	listeners map[Header][]Listener
	timers    []Timer
	ticks     []Handler
	events    []Event
	done      chan struct{}
	error     error
}

func New() *Loop {
	return &Loop{
		listeners: make(map[Header][]Listener),
		done:      make(chan struct{}),
	}
}

func (loop *Loop) AddListener(header Header, listener Listener) {
	loop.mu.Lock()
	{
		loop.listeners[header] = append(loop.listeners[header], listener)
	}
	loop.mu.Unlock()
}

func (loop *Loop) RemoveListener(header Header, listener Listener) {
	loop.mu.Lock()
	{
		listeners := loop.listeners[header]
		for i, ln := range listeners {
			if ln.Equal(listener) {
				copy(listeners[i:], listeners[i+1:])
				listeners[len(listeners)-1] = nil // remove link to the Listener, avoid potential memory leak
				listeners = listeners[:len(listeners)-1]
			}
		}
		loop.listeners[header] = listeners
	}
	loop.mu.Unlock()
}

func (loop *Loop) RemoveListeners(header Header) {
	loop.mu.Lock()
	{
		for i := range loop.listeners[header] {
			loop.listeners[header][i] = nil
		}
		loop.listeners[header] = loop.listeners[header][:0]
	}
	loop.mu.Unlock()
}

func (loop *Loop) AddTimer(t Timer) {
	loop.mu.Lock()
	{
		t.Start(loop.now)
		loop.timers = append(loop.timers, t)
	}
	loop.mu.Unlock()
}

func (loop *Loop) RemoveTimer(victim Timer) {
	loop.mu.Lock()
	{
		for i, t := range loop.timers {
			if t.Equal(victim) {
				copy(loop.timers[i:], loop.timers[i+1:])
				loop.timers[len(loop.timers)-1] = nil // remove link to the Listener, avoid potential memory leak
				loop.timers = loop.timers[:len(loop.timers)-1]
			}
		}
	}
	loop.mu.Unlock()
}

func (loop *Loop) NextTick(fn Handler) {
	loop.mu.Lock()
	loop.ticks = append(loop.ticks, fn)
	loop.mu.Unlock()
}

func (loop *Loop) Trigger(evt Event) {
	loop.mu.Lock()
	loop.events = append(loop.events, evt)
	loop.mu.Unlock()
}

func (loop *Loop) Start() error {
	go func() {
		for loop.IsAlive() {
			loop.setNow(time.Now())
			loop.checkTimers()
			loop.drainNextEvent()
			loop.drainTicks()
		}

		close(loop.done) // loop is done now
	}()
	return nil
}

func (loop *Loop) checkTimers() {
	loop.mu.RLock()
	{
		for _, timer := range loop.timers {
			if timer.IsActive() && timer.Before(loop.now) {
				timer.Fire()
			}
		}
	}
	loop.mu.RUnlock()
}

func (loop *Loop) setNow(now time.Time) {
	loop.mu.Lock()
	loop.now = now
	loop.mu.Unlock()
}

func (loop *Loop) drainTicks() {
	loop.mu.Lock()
	{
		for _, tick := range loop.ticks {
			tick.Fire()
		}
		loop.ticks = loop.ticks[:0]
	}
	loop.mu.Unlock()
}

func (loop *Loop) drainNextEvent() {
	var (
		evt Event
		ok  bool
	)
	loop.mu.Lock()
	{
		if len(loop.events) > 0 {
			evt = loop.events[0]
			ok = true
			loop.events = loop.events[1:]
		}
	}
	loop.mu.Unlock()

	if ok {
		loop.handleEvent(evt)
	}
}

func (loop *Loop) IsAlive() (alive bool) {
	loop.mu.RLock()
	{
		switch {
		case len(loop.ticks) > 0:
			alive = true
		case loop.hasActiveListener():
			alive = true
		case loop.hasActiveTimer():
			alive = true
		}
	}
	loop.mu.RUnlock()
	return
}

func (loop *Loop) Done() <-chan struct{} {
	return loop.done
}

func (loop *Loop) Error() error {
	return loop.error
}

func (loop *Loop) hasActiveListener() bool {
	for _, listeners := range loop.listeners {
		for _, ln := range listeners {
			if ln.IsActive() {
				return true
			}
		}
	}
	return false
}

func (loop *Loop) hasActiveTimer() bool {
	for _, t := range loop.timers {
		if t.IsActive() {
			return true
		}
	}
	return false
}

func (loop *Loop) handleEvent(evt Event) {
	loop.mu.RLock()
	{
		for _, listener := range loop.listeners[evt.Header] {
			listener.Fire()
		}
	}
	loop.mu.RUnlock()
}

func pointerEquality(a, b interface{}) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}
