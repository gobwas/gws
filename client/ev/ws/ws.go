package ws

import (
	"errors"
	"fmt"
	"github.com/gobwas/gws/client/ev"
	"github.com/gobwas/gws/ws"
	"net/http"
	"sync/atomic"
)

type Handler struct {
	pending int32
}

func NewHandler() *Handler {
	return &Handler{}
}

type Connect struct {
	Url     string
	Headers http.Header
}

type Send struct {
	Conn    *ws.Connection
	Message ws.MessageRaw
}

type Receive struct {
	done chan struct{}
	conn *ws.Connection
}

func NewReceive(c *ws.Connection) *Receive {
	return &Receive{
		done: make(chan struct{}),
		conn: c,
	}
}

func (r *Receive) Stop() {
	close(r.done)
}

func (h *Handler) Handle(loop *ev.Loop, data interface{}, cb ev.Callback) error {
	switch v := data.(type) {
	case Connect:
		h.doConnect(loop, v, cb)

	case Send:
		h.doSend(loop, v, cb)

	case *Receive:
		h.doReceive(loop, v, cb)

	default:
		return fmt.Errorf("unknown request format to ws handler: %s", data)
	}

	return nil
}

func (h *Handler) doConnect(loop *ev.Loop, req Connect, cb ev.Callback) {
	atomic.AddInt32(&h.pending, 1)
	// todo use here pool of workers
	go func() {
		conn, _, err := ws.GetConn(req.Url, req.Headers)
		if err != nil {
			loop.Call(func() {
				cb(err, nil)
			})
		} else {
			loop.Call(func() {
				cb(nil, ws.NewConnection(conn))
			})
		}
		atomic.AddInt32(&h.pending, -1)
	}()
}

func (h *Handler) doSend(loop *ev.Loop, req Send, cb ev.Callback) {
	atomic.AddInt32(&h.pending, 1)
	// todo use here pool of workers
	go func() {
		err := req.Conn.Send(req.Message)
		if err != nil {
			loop.Call(func() {
				cb(err, nil)
			})
		} else {
			loop.Call(func() {
				cb(nil, nil)
			})
		}
		atomic.AddInt32(&h.pending, -1)
	}()
}

func (h *Handler) doReceive(loop *ev.Loop, req *Receive, cb ev.Callback) {
	atomic.AddInt32(&h.pending, 1)
	// todo use here pool of workers
	go func() {
		defer atomic.AddInt32(&h.pending, -1)

		for {
			select {
			case <-req.conn.Done():
				return

			case <-req.done:
				fmt.Println("Closed receive")
				loop.Call(func() {
					cb(errors.New("listen was interrupted"), nil)
				})
				return

			default:
				msg, err := req.conn.Receive()
				if err != nil {
					loop.Call(func() {
						cb(err, nil)
					})
					return
				}

				loop.Call(func() {
					cb(nil, string(msg.Data))
				})
			}
		}
	}()
}

func (h *Handler) IsActive() bool {
	return atomic.LoadInt32(&h.pending) > 0
}
