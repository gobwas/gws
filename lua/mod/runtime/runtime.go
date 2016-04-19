package runtime

import (
	"github.com/gobwas/gws/client/events"
	"github.com/yuin/gopher-lua"
	"time"
)

type Mod struct {
	initTime  time.Time
	loop      *events.Loop
	master    bool
	callbacks map[lua.LFunction]events.Listener
}

func New(loop *events.Loop, master bool) *Mod {
	return &Mod{
		initTime:  time.Now(),
		loop:      loop,
		master:    master,
		callbacks: make(map[lua.LFunction]events.Listener),
	}
}

func (m *Mod) Exports() lua.LGFunction {
	return func(L *lua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "name", "runtime")

		mod.RawSetString("isMaster", L.NewClosure(func(L *lua.LState) int {
			L.Push(lua.LBool(m.master))
			return 1
		}))

		mod.RawSetString("on", L.NewClosure(func(L *lua.LState) int {
			name := L.ToString(1)
			cb := L.ToFunction(2)
			listener := events.HandlerFunc(func() {
				L.CallByParam(lua.P{
					Fn:      cb,
					NRet:    0,
					Protect: false,
				})
			})
			m.callbacks[listener] = cb
			m.loop.AddListener(events.Header{events.EventCustom, name}, listener)
			return 0
		}))

		mod.RawSetString("off", L.NewClosure(func(L *lua.LState) int {
			name := L.ToString(1)
			cb := L.ToFunction(2)
			listener, ok := m.callbacks[cb]
			if !ok {
				L.Push(lua.LString("unknown callback"))
				return 1
			}

			m.loop.RemoveListener(events.Header{events.EventCustom, name}, listener)
			delete(m.callbacks, cb)

			return 0
		}))

		L.Push(mod)
		return 1
	}
}

func (m *Mod) Name() string {
	return "gws.runtime"
}
