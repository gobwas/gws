package ws

import (
	"fmt"
	"github.com/gobwas/gws/ev"
	"github.com/gobwas/gws/ws"
	"net/http"
	"sync/atomic"
)

type Handler struct {
	pending int32
	stop    chan struct{}
}

func NewHandler() *Handler {
	return &Handler{
		stop: make(chan struct{}),
	}
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
	conn *ws.Connection
}

type Listen struct {
	TLS     bool
	Key     string
	Cert    string
	Addr    string
	Origin  string
	Headers http.Header
}

func NewReceive(c *ws.Connection) *Receive {
	return &Receive{
		conn: c,
	}
}

func (h *Handler) Handle(loop *ev.Loop, data interface{}, cb ev.Callback) error {
	switch v := data.(type) {
	case Connect:
		h.doConnect(loop, v, cb)

	case Send:
		h.doSend(loop, v, cb)

	case Listen:
		h.doListen(loop, v, cb)

	case *Receive:
		h.doReceive(loop, v, cb)

	default:
		return fmt.Errorf("unknown request format to ws handler: %s", data)
	}

	return nil
}

func (h *Handler) Stop() {
	close(h.stop)
}

func (h *Handler) IsActive() bool {
	return atomic.LoadInt32(&h.pending) > 0
}

func (h *Handler) doListen(loop *ev.Loop, req Listen, cb ev.Callback) {
	atomic.AddInt32(&h.pending, 1)

	go func() {
		upgrade := ws.GetUpgrader(ws.UpgradeConfig{
			Origin:  req.Origin,
			Headers: req.Headers,
		})

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if conn, err := upgrade(w, r); err != nil {
				loop.Call(func() {
					cb(err, nil)
				})
			} else {
				loop.Call(func() {
					cb(nil, ws.NewConnection(conn))
				})
			}
		})

		var err error
		if req.TLS {
			err = http.ListenAndServeTLS(req.Addr, req.Cert, req.Key, handler)
		} else {
			err = http.ListenAndServe(req.Addr, handler)
		}

		cb(err, nil)
		atomic.AddInt32(&h.pending, -1)
	}()
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
			case <-h.stop:
				return

			case <-req.conn.Done():
				return

			case envelope := <-req.conn.ReceiveAsync():
				msg, err := envelope.Message, envelope.Error
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
