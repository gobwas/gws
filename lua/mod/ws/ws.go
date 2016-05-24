package ws

import (
	"github.com/gobwas/gws/ev"
	evws "github.com/gobwas/gws/ev/ws"
	luautil "github.com/gobwas/gws/lua/util"
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

		mod.RawSetString("createServer", L.NewClosure(func(L *lua.LState) int {
			var cfg ws.ServerConfig
			if opts := L.ToTable(1); opts != nil {
				opts.ForEach(func(key lua.LValue, value lua.LValue) {
					if key.Type() == lua.LTString {
						switch key.String() {
						case "cert":
							cfg.Cert = value.String()
						case "key":
							cfg.Key = value.String()
						case "origin":
							cfg.Origin = value.String()
						case "headers":
							t, ok := value.(*lua.LTable)
							if ok {
								cfg.Headers = luautil.HeadersFromTable(t)
							}
						}
					}
				})
			}

			server := NewServer(m.loop, cfg)
			L.Push(server.ToTable(L))

			return 1
		}))

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
