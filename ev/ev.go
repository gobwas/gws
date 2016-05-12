package ev

import (
	"sync"
	"time"
)

type RequestType int

type Callback func(error, interface{})

type Handler interface {
	Handle(*Loop, interface{}, Callback) error
	IsActive() bool
	Stop()
}

type Loop struct {
	mu sync.Mutex

	handlers  map[RequestType][]Handler
	requests  []*request
	events    []event
	teardowns []event
	timers    []*Timer
	now       time.Time
	done      chan struct{}
	shutdown  chan struct{}
	stop      chan struct{}
	locked    bool
	//	idles    []Idle
}

func NewLoop() *Loop {
	return &Loop{
		done:     make(chan struct{}),
		shutdown: make(chan struct{}),
		stop:     make(chan struct{}),
		handlers: make(map[RequestType][]Handler),
		now:      time.Now(),
	}
}

// todo use priority
func (l *Loop) Register(h Handler, t RequestType) {
	l.handlers[t] = append(l.handlers[t], h)
}

func (l *Loop) Request(t RequestType, data interface{}, cb Callback) error {
	l.mu.Lock()
	{
		if !l.locked {
			l.requests = append(l.requests, &request{
				t:    t,
				cb:   cb,
				data: data,
			})
		}
	}
	l.mu.Unlock()
	return nil
}

func (l *Loop) Call(cb event) {
	l.mu.Lock()
	{
		if !l.locked {
			l.events = append(l.events, cb)
		}
	}
	l.mu.Unlock()
}

func (l *Loop) Timeout(delay time.Duration, repeat bool, cb event) *Timer {
	timer := &Timer{
		delay:  delay,
		repeat: repeat,
		cb:     cb,
	}

	l.mu.Lock()
	{
		if !l.locked {
			timer.next = l.now.Add(timer.delay)
			l.timers = append(l.timers, timer)
		}
	}
	l.mu.Unlock()

	return timer
}

func (l *Loop) Teardown(cb event) {
	l.mu.Lock()
	{
		l.teardowns = append(l.teardowns, cb)
	}
	l.mu.Unlock()
}

func (l *Loop) Done() chan struct{} {
	return l.done
}

func (l *Loop) Shutdown() {
	close(l.shutdown)
}

func (l *Loop) Stop() {
	l.stop <- struct{}{}
}

func (l *Loop) lock() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.locked = true
}

func (l *Loop) Run() error {
	//	l.mu.Lock()
	//	defer l.mu.Unlock()
	// TODO(s.kamardin): what we should do if we run twice?

	go func() {
		for {
			select {
			case <-l.shutdown:
				close(l.done)
				return

			case <-l.stop:
				l.stopTimers()
				l.drainTeardown()
				l.stopHandlers()
				l.lock()

			default:
				if !l.IsAlive() {
					if !l.drainTeardown() {
						close(l.done)
						return
					}
				} else {
					l.updateNow()
					l.drainTimers()
					l.drainTicks()

					l.nextEvent()
					l.nextRequest()
				}
			}
		}
	}()

	return nil
}

func (l *Loop) IsAlive() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.requests) > 0 {
		return true
	}
	if len(l.events) > 0 {
		return true
	}
	if len(l.timers) > 0 {
		return true
	}
	//	if l.locked {
	//		return false
	//	}

	for _, handlers := range l.handlers {
		for _, handler := range handlers {
			if handler.IsActive() {
				return true
			}
		}
	}

	return false
}

func (l *Loop) updateNow() {
	l.now = time.Now()
}

func (l *Loop) stopHandlers() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, handlers := range l.handlers {
		for _, h := range handlers {
			h.Stop()
		}
	}
}

func (l *Loop) stopTimers() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, timer := range l.timers {
		timer.Stop()
	}
}

func (l *Loop) drainTimers() {
	var callbacks []event

	l.mu.Lock()
	{
		for i := 0; i < len(l.timers); {
			t := l.timers[i]

			var remove bool
			switch {
			case t.dropped:
				remove = true

			case t.next.Before(l.now):
				callbacks = append(callbacks, t.cb)

				if t.repeat {
					t.next = l.now.Add(t.delay)
				} else {
					remove = true
				}
			}

			if remove {
				l.deleteTimer(i)
			} else {
				i++
			}
		}
	}
	l.mu.Unlock()

	for _, cb := range callbacks {
		cb()
	}
}

func (l *Loop) deleteTimer(i int) {
	copy(l.timers, l.timers[i+1:])
	last := len(l.timers) - 1
	l.timers[last] = nil
	l.timers = l.timers[:last]
}

func (l *Loop) drainTicks() {
	// TODO(s.kamardin)
}

func (l *Loop) drainTeardown() bool {
	var callbacks []event

	l.mu.Lock()
	{
		callbacks = l.teardowns
		l.teardowns = nil
	}
	l.mu.Unlock()

	for _, cb := range callbacks {
		cb()
	}

	return len(callbacks) > 0
}

func (l *Loop) nextTeardown() bool {
	l.mu.Lock()

	if len(l.teardowns) == 0 {
		l.mu.Unlock()
		return false
	}

	teardown := l.teardowns[0]
	l.teardowns = append(l.teardowns[:0], l.teardowns[1:]...)

	l.mu.Unlock()

	teardown()
	return true
}

func (l *Loop) nextEvent() bool {
	l.mu.Lock()

	if len(l.events) == 0 {
		l.mu.Unlock()
		return false
	}

	evt := l.events[0]
	l.events = append(l.events[:0], l.events[1:]...)

	l.mu.Unlock()

	evt()

	return true
}

func (l *Loop) nextRequest() {
	l.mu.Lock()

	if len(l.requests) == 0 {
		l.mu.Unlock()
		return
	}

	evt := l.requests[0]
	l.requests = append(l.requests[:0], l.requests[1:]...)

	l.mu.Unlock()

	// unlock event here allows handler to act in synchronous way,
	// e.g. it could call evt.Callback() immediately.
	// after loop is done, eve.lock() guarantee that possible result of request
	// will be called only through loop.Request()
	//	evt.unlock()
	{
		for _, handler := range l.handlers[evt.t] {
			err := handler.Handle(l, evt.data, evt.cb)
			if err != nil {
				panic(err)
			}
		}
	}
	//	evt.lock()
}

type event func()

type request struct {
	t    RequestType
	cb   Callback
	data interface{}
}

func (e *request) Data() interface{} {
	return e.data
}

func (e *request) Callback() Callback {
	return e.cb
}

type Timer struct {
	delay   time.Duration
	repeat  bool
	cb      event
	dropped bool
	next    time.Time
}

func (t *Timer) Stop() {
	t.dropped = true
}
