package ws

import (
	"fmt"
	"github.com/gobwas/gws/ev"
	"github.com/gobwas/gws/ws"
	"github.com/gorilla/websocket"
	"reflect"
	"sync"
	"sync/atomic"
)

type serverDesc struct {
	cfg    ws.ServerConfig
	server *ws.Server
	loops  []*ev.Loop
}

type ServerHandler struct {
	mu      sync.Mutex
	pending map[*ev.Loop]int32
	servers map[string]serverDesc
	stop    chan struct{}
	stopped int32
}

func NewServerHandler() *ServerHandler {
	return &ServerHandler{
		pending: make(map[*ev.Loop]int32),
		servers: make(map[string]serverDesc),
		stop:    make(chan struct{}),
	}
}

func (h *ServerHandler) Init(*ev.Loop) error {
	return nil
}

func (h *ServerHandler) Handle(loop *ev.Loop, data interface{}, cb ev.Callback) error {
	switch v := data.(type) {
	case ws.ServerConfig:
		h.doListen(loop, v, cb)
	default:
		return fmt.Errorf("unknown request format to ws handler: %s", data)
	}
	return nil
}

func (h *ServerHandler) Stop() {
	if atomic.CompareAndSwapInt32(&h.stopped, 0, 1) {
		close(h.stop)
	}
}

func (h *ServerHandler) IsActive(loop *ev.Loop) bool {
	return h.getPending(loop) > 0
}

func (h *ServerHandler) doListen(loop *ev.Loop, cfg ws.ServerConfig, cb ev.Callback) {
	h.mu.Lock()
	defer h.mu.Unlock()

	desc, ok := h.servers[cfg.Addr]
	if ok {
		if !reflect.DeepEqual(desc.cfg, cfg) {
			cb(fmt.Errorf("already listening on %s with different configuration", cfg.Addr), nil)
			return
		}
		for _, l := range desc.loops {
			if l == loop {
				cb(fmt.Errorf("already listening on %s in current thread", cfg.Addr), nil)
				return
			}
		}
	} else {
		s := ws.NewServer(ws.ServerConfig{
			Key:     cfg.Key,
			Cert:    cfg.Cert,
			Addr:    cfg.Addr,
			Headers: cfg.Headers,
			Origin:  cfg.Origin,
		})
		defer s.Listen(h.stop)
		desc = serverDesc{server: s, cfg: cfg}
		h.servers[cfg.Addr] = desc
	}

	desc.loops = append(desc.loops, loop)
	desc.server.Handle(ws.HandlerFunc(func(conn *websocket.Conn, err error) {
		if err != nil {
			loop.Call(func() { cb(err, nil) })
		} else {
			loop.Call(func() { cb(nil, ws.NewConnection(conn)) })
		}
	}))

	h.pending[loop] += 1
	desc.server.Defer(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.pending[loop] -= 1
	})
}

func (h *ServerHandler) getPending(loop *ev.Loop) int32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pending[loop]
}
