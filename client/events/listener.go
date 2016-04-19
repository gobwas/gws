package events

import "sync"

type HandlerFunc func()

func (l HandlerFunc) Fire()                { l() }
func (l HandlerFunc) IsActive() bool       { return true }
func (a HandlerFunc) Equal(b Handler) bool { return pointerEquality(a, b) }

type DefaultHandler struct {
	mu sync.Mutex

	cb     func(*DefaultHandler)
	count  uint64
	Active bool
}

func NewHandler(fn func(*DefaultHandler)) *DefaultHandler {
	return &DefaultHandler{cb: fn, Active: true}
}

func NewHandlerOnce(fn func()) *DefaultHandler {
	return NewHandler(func(h *DefaultHandler) {
		fn()
		h.Active = false
	})
}

func (l *DefaultHandler) Fire() {
	l.mu.Lock()
	l.count++
	l.cb(l)
	l.mu.Unlock()
}

func (l *DefaultHandler) Count() (c uint64) {
	l.mu.Lock()
	c = l.count
	l.mu.Unlock()
	return
}

func (l *DefaultHandler) IsActive() (active bool) {
	l.mu.Lock()
	active = l.Active
	l.mu.Unlock()
	return active
}

func (l *DefaultHandler) Equal(b Handler) bool {
	if lb, ok := b.(*DefaultHandler); ok {
		return l == lb
	}
	return false
}
