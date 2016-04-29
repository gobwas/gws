package runtime

import (
	"github.com/gobwas/gws/client/ev"
	"github.com/gobwas/gws/lua/mod"
	"github.com/yuin/gopher-lua"
	"sync/atomic"
	"time"
)

type Runtime struct {
	exported int32
	initTime time.Time
	loop     *ev.Loop
	emitter  *mod.Emitter
	storage  *mod.Storage
	fork     forkFn
}

type callback struct {
	event string
	fn    *lua.LFunction
}

type forkFn func() error

func New(loop *ev.Loop) *Runtime {
	return &Runtime{
		emitter:  mod.NewEmitter(),
		storage:  mod.NewStorage(),
		initTime: time.Now(),
		loop:     loop,
	}
}

func (m *Runtime) SetForkFn(f forkFn) {
	if atomic.LoadInt32(&m.exported) > 0 {
		panic("could not set fork function after runtime is exported")
	}
	m.fork = f
}

func (m *Runtime) Emit(name string) {
	m.loop.Call(func() {
		m.emitter.Emit(name)
	})
}

func (m *Runtime) Set(key string, value interface{}) {
	m.storage.Set(key, value)
}

func (m *Runtime) Get(key string) (result interface{}) {
	return m.storage.Get(key)
}

func (m *Runtime) Exports() lua.LGFunction {
	if atomic.LoadInt32(&m.exported) > 0 {
		panic("runtime could be exported only once")
	} else {
		atomic.AddInt32(&m.exported, 1)
	}

	return func(L *lua.LState) int {
		mod := L.NewTable()

		if m.fork != nil {
			mod.RawSetString("fork", L.NewClosure(func(L *lua.LState) int {
				if err := m.fork(); err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				return 0
			}))
		}

		mod.RawSetString("isMaster", L.NewClosure(func(L *lua.LState) int {
			L.Push(lua.LBool(m.fork != nil))
			return 1
		}))

		mod.RawSetString("set", m.storage.ExportSet(L))
		mod.RawSetString("get", m.storage.ExportGet(L))

		mod.RawSetString("on", m.emitter.ExportOn(L))
		mod.RawSetString("off", m.emitter.ExportOff(L))

		L.Push(mod)
		return 1
	}
}
