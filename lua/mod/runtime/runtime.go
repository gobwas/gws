package runtime

import (
	"github.com/gobwas/gws/client/ev"
	"github.com/gobwas/gws/lua/mod"
	"github.com/yuin/gopher-lua"
	"sync/atomic"
	"time"
)

type Mod struct {
	exported int32
	initTime time.Time
	loop     *ev.Loop
	emitter  *mod.Emitter
	storage  *mod.Storage
	master   bool
}

type callback struct {
	event string
	fn    *lua.LFunction
}

func New(loop *ev.Loop, master bool) *Mod {
	return &Mod{
		emitter:  mod.NewEmitter(),
		storage:  mod.NewStorage(),
		initTime: time.Now(),
		loop:     loop,
		master:   master,
	}
}

func (m *Mod) Emit(name string) {
	m.loop.Call(func() {
		m.emitter.Emit(name)
	})
}

func (m *Mod) Set(key string, value interface{}) {
	m.storage.Set(key, value)
}

func (m *Mod) Get(key string) (result interface{}) {
	return m.storage.Get(key)
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

		mod.RawSetString("isMaster", L.NewClosure(func(L *lua.LState) int {
			L.Push(lua.LBool(m.master))
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
