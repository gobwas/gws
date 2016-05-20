package timer

import (
	"errors"
	"github.com/gobwas/gws/ev"
	"sync/atomic"
	"time"
)

type delay struct {
	timer   *time.Timer
	timeout *Timeout
	loop    *ev.Loop
	cb      ev.Callback
}

type Handler struct {
	t     ev.RequestType
	delay chan delay
	count int32
}

func NewHandler(t ev.RequestType) *Handler {
	return &Handler{
		t:     t,
		delay: make(chan delay, 1),
	}
}

func (h *Handler) Init() {
	go func() {
		for {
			for d := range h.delay {
				if d.timeout.dropped {
					d.timer.Stop()
					atomic.AddInt32(&h.count, -1)
					continue
				}

				select {
				case t := <-d.timer.C:
					d.loop.Call(func() { d.cb(nil, t) })
					if d.timeout.repeat {
						d.timer.Reset(d.timeout.delay)
						h.delay <- d
					} else {
						atomic.AddInt32(&h.count, -1)
					}
				default:
					h.delay <- d
				}
			}
		}
	}()
}

func (h *Handler) SetTimeout(loop *ev.Loop, t *Timeout, cb ev.Callback) {
	h.delay <- delay{
		cb:      cb,
		loop:    loop,
		timer:   time.NewTimer(t.delay),
		timeout: t,
	}
	atomic.AddInt32(&h.count, 1)
}

func (h *Handler) Handle(loop *ev.Loop, data interface{}, cb ev.Callback) error {
	t, ok := data.(*Timeout)
	if !ok {
		return errors.New("unexpected data")
	}
	h.SetTimeout(loop, t, cb)

	return nil
}

func (h *Handler) IsActive() bool {
	return atomic.LoadInt32(&h.count) > 0
}

type Timeout struct {
	delay   time.Duration
	dropped bool
	repeat  bool
}

func NewTimeout(t time.Duration) *Timeout {
	return &Timeout{
		delay: t,
	}
}

func NewTicker(t time.Duration) *Timeout {
	return &Timeout{
		delay:  t,
		repeat: true,
	}
}

func (t *Timeout) Stop() {
	t.dropped = true
}
