package ws

import (
	"github.com/gobwas/gws/client/ev"
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
			m.loop.Request(100, connRequest{uri, headers}, func(err error, data interface{}) {
				if err != nil {
					L.CallByParam(lua.P{
						Fn:      cb,
						NRet:    0,
						Protect: false,
					}, lua.LString(err.Error()), lua.LNil)
					return
				}

				if c, ok := data.(*ws.Connection); !ok {
					L.CallByParam(lua.P{
						Fn:      cb,
						NRet:    0,
						Protect: false,
					}, lua.LString("internal error"), lua.LNil)
				} else {
					conn := &Conn{c}
					L.CallByParam(lua.P{
						Fn:      cb,
						NRet:    0,
						Protect: false,
					}, lua.LNil, conn.ToTable(L))
				}
			})

			return 0
		}))

		L.Push(mod)
		return 1
	}
}

type connRequest struct {
	u string
	h http.Header
}

func (c connRequest) Url() string          { return c.u }
func (c connRequest) Headers() http.Header { return c.h }

type Conn struct {
	conn *ws.Connection
}

func (c *Conn) ToTable(L *lua.LState) *lua.LTable {
	table := L.NewTable()

	table.RawSetString("send", L.NewClosure(func(L *lua.LState) int {
		return 2
	}))

	table.RawSetString("onmessage", L.NewClosure(func(L *lua.LState) int {
		return 2
	}))

	table.RawSetString("close", L.NewClosure(func(L *lua.LState) int {
		return 2
	}))

	return table
}
