package script

import (
	"github.com/gobwas/gws/lua/mod"
	"github.com/yuin/gopher-lua"
	"io"
)

type Script struct {
	luaState *lua.LState
}

func New() *Script {
	return &Script{
		luaState: lua.NewState(),
	}
}

func (s *Script) Preload(name string, m mod.Module) {
	s.luaState.PreloadModule(name, m.Exports())
}

func (s *Script) Do(code string) error {
	return s.luaState.DoString(code)
}

func (s *Script) Shutdown() {
	s.luaState.Close()
}

func (s *Script) HijackOutput(w io.Writer) {
	s.luaState.SetGlobal("print", s.luaState.NewFunction(func(L *lua.LState) int {
		var buf []byte
		for i := 1; ; i++ {
			def := L.Get(i)
			if _, ok := def.(*lua.LNilType); ok {
				break
			}
			buf = append(buf, []byte(def.String())...)
		}
		w.Write(append(buf, '\n'))
		return 0
	}))
}
