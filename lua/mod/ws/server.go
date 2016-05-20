package ws

import (
	"github.com/gobwas/gws/ev"
	evws "github.com/gobwas/gws/ev/ws"
	"github.com/gobwas/gws/lua/mod"
	"github.com/gobwas/gws/ws"
	"github.com/yuin/gopher-lua"
)

type Server struct {
	loop      *ev.Loop
	emitter   *mod.Emitter
	config    ServerConfig
	listening bool
}

type ServerConfig struct {
	TLS  bool
	Key  string
	Cert string
}

func NewServer(l *ev.Loop, c ServerConfig) *Conn {
	return &Server{
		loop:    l,
		emitter: mod.NewEmitter(),
		config:  c,
	}
}

func (s *Server) Emit(name string, args ...interface{}) {
	s.loop.Call(func() {
		s.emitter.Emit(name, args...)
	})
}

func (s *Server) ToTable(L *lua.LState) *lua.LTable {
	table := L.NewTable()

	table.RawSetString("listen", L.NewClosure(func(L *lua.LState) int {
		if s.listening {
			L.Push(lua.LString("already listening"))
			return 1
		}
		s.listening = true

		addr := L.ToString(1)
		cb := L.ToFunction(2)

		req := evws.Listen{
			TLS:  s.config.TLS,
			Key:  s.config.Key,
			Cert: s.config.Cert,
			Addr: addr,
		}
		var e error // hack to catch sync error
		s.loop.Request(100, req, func(err error, msg interface{}) {
			if err != nil {
				e = err
				s.Emit("error", err.Error())
				return
			}

			if c, ok := msg.(*ws.Connection); ok {
				c.InitIOWorkers()
				conn := NewConn(c, s.loop)
				L.CallByParam(lua.P{
					Fn:      cb,
					NRet:    0,
					Protect: false,
				}, conn.ToTable(L))
				return
			}

			panic("ws: unexpected data in request callback")
		})

		if e != nil { // sync error
			L.Push(lua.LString(e.Error()))
			return 1
		}

		return 0
	}))

	return table
}
