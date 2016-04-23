package ws

import (
	"github.com/gobwas/gws/client/ev"
	evws "github.com/gobwas/gws/client/ev/ws"
	"github.com/gobwas/gws/lua/mod"
	"github.com/gobwas/gws/ws"
	"github.com/yuin/gopher-lua"
	"net/http"
)

type Mod struct {
	loop *ev.Loop
}

func New(loop *ev.Loop) *Mod {
	return &Mod{
		loop: loop,
	}
}

func (m *Mod) Exports() lua.LGFunction {
	return func(L *lua.LState) int {
		mod := L.NewTable()

		mod.RawSetString("connect", L.NewClosure(func(L *lua.LState) int {
			opts := L.ToTable(1)
			cb := L.ToFunction(2)

			var uri string
			if u := opts.RawGetString("url"); u.Type() == lua.LTString {
				uri = u.String()
			} else {
				m.loop.Call(func() {
					L.CallByParam(lua.P{
						Fn:      cb,
						NRet:    0,
						Protect: false,
					}, lua.LString("url is expected to be a string in options table"), lua.LNil)
				})
				return 0
			}

			var headers http.Header
			if h := opts.RawGetString("headers"); h.Type() == lua.LTTable {
				headers = make(http.Header)
				h.(*lua.LTable).ForEach(func(k, v lua.LValue) {
					if k.Type() == lua.LTString && v.Type() == lua.LTString {
						headers.Set(k.String(), v.String())
					}
				})
			}

			// todo use constant for channel id
			m.loop.Request(100, evws.Connect{uri, headers}, func(err error, data interface{}) {
				if err != nil {
					L.CallByParam(lua.P{
						Fn:      cb,
						NRet:    0,
						Protect: false,
					}, lua.LString(err.Error()), lua.LNil)
					return
				}

				if c, ok := data.(*ws.Connection); ok {
					c.InitIOWorkers()
					conn := NewConn(c, m.loop)
					L.CallByParam(lua.P{
						Fn:      cb,
						NRet:    0,
						Protect: false,
					}, lua.LNil, conn.ToTable(L))
					return
				}

				panic("ws: unexpected data in request callback")
			})

			return 0
		}))

		L.Push(mod)
		return 1
	}
}

type Conn struct {
	conn    *ws.Connection
	loop    *ev.Loop
	emitter *mod.Emitter

	receive  *evws.Receive
	listener []ln
}

func NewConn(c *ws.Connection, l *ev.Loop) *Conn {
	return &Conn{
		emitter: mod.NewEmitter(),
		conn:    c,
		loop:    l,
	}
}

func (c *Conn) Emit(name string) {
	c.loop.Call(func() {
		c.emitter.Emit(name)
	})
}

func (c *Conn) ToTable(L *lua.LState) *lua.LTable {
	table := L.NewTable()

	table.RawSetString("send", L.NewClosure(func(L *lua.LState) int {
		str := L.ToString(1)
		msg := ws.MessageRaw{ws.TextMessage, []byte(str)}

		cb := L.ToFunction(2)
		if cb == nil { // if there is no callback - act synchronous
			if err := c.conn.Send(msg); err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
			return 0
		}

		c.loop.Request(100, evws.Send{c.conn, msg}, func(err error, _ interface{}) {
			var e lua.LValue
			if err != nil {
				e = lua.LString(err.Error())
			} else {
				e = lua.LNil
			}
			L.CallByParam(lua.P{
				Fn:      cb,
				NRet:    0,
				Protect: false,
			}, e)
		})
		return 0
	}))

	table.RawSetString("listen", L.NewClosure(func(L *lua.LState) int {
		cb := L.ToFunction(1)
		c.listener = append(c.listener, ln{cb, L})

		if c.receive == nil {
			c.receive = evws.NewReceive(c.conn)
			c.loop.Request(100, c.receive, func(err error, msg interface{}) {
				for _, ln := range c.listener {
					if err != nil {
						ln.state.CallByParam(lua.P{
							Fn:      cb,
							NRet:    0,
							Protect: false,
						}, lua.LString(err.Error()), lua.LNil)
					} else {
						ln.state.CallByParam(lua.P{
							Fn:      cb,
							NRet:    0,
							Protect: false,
						}, lua.LNil, lua.LString(msg.(string)))
					}
				}
			})
		}

		return 0
	}))

	table.RawSetString("receive", L.NewClosure(func(L *lua.LState) int {
		if c.receive != nil {
			c.receive.Stop()
		}

		msg, err := c.conn.Receive()
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		} else {
			L.Push(lua.LString(msg.Data))
			L.Push(lua.LNil)
			return 2
		}
	}))

	table.RawSetString("close", L.NewClosure(func(L *lua.LState) int {
		c.receive.Stop()
		c.conn.Close()
		c.Emit("close")
		return 0
	}))

	table.RawSetString("on", c.emitter.ExportOn(L))
	table.RawSetString("off", c.emitter.ExportOff(L))

	return table
}

type ln struct {
	cb    *lua.LFunction
	state *lua.LState
}
