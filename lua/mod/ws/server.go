package ws

import (
	"github.com/gobwas/gws/ev"
	"github.com/gobwas/gws/lua/mod"
	"github.com/gobwas/gws/ws"
	"github.com/yuin/gopher-lua"
)

type Server struct {
	loop    *ev.Loop
	emitter *mod.Emitter
	config  ws.ServerConfig
}

func NewServer(l *ev.Loop, c ws.ServerConfig) *Server {
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
		addr := L.ToString(1)
		cb := L.ToFunction(2)

		req := s.config
		req.Addr = addr

		var e error // hack to catch sync error
		s.loop.Request(101, req, func(err error, msg interface{}) {
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
		} else {
			s.Emit("listening")
		}

		return 0
	}))

	table.RawSetString("on", s.emitter.ExportOn(L))
	table.RawSetString("off", s.emitter.ExportOff(L))

	return table
}
