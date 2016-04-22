package runtime

import (
	"github.com/gobwas/gws/client/ev"
	"github.com/yuin/gopher-lua"
	"sync/atomic"
	"time"
)

type Mod struct {
	exported  int32
	initTime  time.Time
	loop      *ev.Loop
	master    bool
	callbacks map[string][]callback
	data      map[string]interface{}
}

type callback struct {
	state *lua.LState
	fn    *lua.LFunction
}

func New(loop *ev.Loop, master bool) *Mod {
	return &Mod{
		initTime:  time.Now(),
		loop:      loop,
		master:    master,
		callbacks: make(map[string][]callback),
		data:      make(map[string]interface{}),
	}
}

func (m *Mod) Emit(name string) {
	if len(m.callbacks[name]) > 0 {
		m.loop.Call(func() {
			for _, callback := range m.callbacks[name] {
				callback.state.CallByParam(lua.P{
					Fn:      callback.fn,
					NRet:    0,
					Protect: false,
				})
			}
		})
	}
}

func (m *Mod) Set(key string, value interface{}) {
	m.data[key] = value
	return
}

func (m *Mod) Get(key string) (result interface{}) {
	result = m.data[key]
	return
}

func (m *Mod) Exports() lua.LGFunction {
	if atomic.LoadInt32(&m.exported) > 0 {
		panic("runtime could be exported only once")
	} else {
		atomic.AddInt32(&m.exported, 1)
	}

	return func(L *lua.LState) int {
		mod := L.NewTable()

		if m.master {
			mod.RawSetString("fork", L.NewClosure(func(L *lua.LState) int {
				return 0
			}))
		}

		mod.RawSetString("set", L.NewClosure(func(L *lua.LState) int {
			key := L.ToString(1)
			value := L.Get(3)
			m.Set(key, value)
			return 0
		}))

		mod.RawSetString("get", L.NewClosure(func(L *lua.LState) int {
			key := L.ToString(1)
			switch v := m.Get(key).(type) {
			case lua.LValue:
				L.Push(v)
			case string:
				L.Push(lua.LString(v))
			case int:
				L.Push(lua.LNumber(v))
			default:
				L.Push(lua.LNil)
			}
			return 1
		}))

		mod.RawSetString("isMaster", L.NewClosure(func(L *lua.LState) int {
			L.Push(lua.LBool(m.master))
			return 1
		}))

		mod.RawSetString("on", L.NewClosure(func(L *lua.LState) int {
			name := L.ToString(1)
			cb := L.ToFunction(2)
			m.callbacks[name] = append(m.callbacks[name], callback{L, cb})
			return 0
		}))

		mod.RawSetString("off", L.NewClosure(func(L *lua.LState) int {
			name := L.ToString(1)
			cb := L.ToFunction(2)

			for i, victim := range m.callbacks[name] {
				if victim.fn == cb {
					m.callbacks[name] = append(m.callbacks[name][:i], m.callbacks[name][i+1:]...)
					if len(m.callbacks[name]) == 0 {
						delete(m.callbacks, name)
					}
					break
				}
			}

			return 0
		}))

		L.Push(mod)
		return 1
	}
}
