package ws

import (
	"errors"
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

type Request interface {
	Url() string
	Headers() http.Header
}

func (h *Handler) Handle(loop *ev.Loop, data interface{}, cb ev.Callback) error {
	req, ok := data.(Request)
	if !ok {
		return errors.New("unknown request format")
	}

	atomic.AddInt32(&h.pending, 1)

	// todo use here pool of workers
	go func() {
		conn, _, err := ws.GetConn(req.Url(), req.Headers())
		if err != nil {
			loop.Call(func() {
				cb(err, nil)
			})
			return
		}

		loop.Call(func() {
			cb(nil, ws.NewConnection(conn))
		})

		atomic.AddInt32(&h.pending, -1)
	}()

	return nil
}

func (h *Handler) IsActive() bool {
	return atomic.LoadInt32(&h.pending) > 0
}
