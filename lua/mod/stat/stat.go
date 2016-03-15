package stat

import (
	"github.com/yuin/gopher-lua"
	"time"
)

const moduleName = "gws.stat"

type Mod struct {
	statistics *stat
}

func New() *Mod {
	return &Mod{&stat{}}
}

func (m *Mod) Exports() lua.LGFunction {
	return func(L *lua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "name", lua.LString(moduleName))
		mod.RawSetString("abs", L.NewClosure(registerAbs(m.statistics)))
		mod.RawSetString("avg", L.NewClosure(registerAvg(m.statistics)))
		mod.RawSetString("per", L.NewClosure(registerPer(m.statistics)))
		mod.RawSetString("add", L.NewClosure(increment(m.statistics)))

		L.Push(mod)
		return 1
	}
}

func (m *Mod) Name() string {
	return moduleName
}

func increment(s *stat) lua.LGFunction {
	return func(L *lua.LState) int {
		name := L.ToString(1)
		c, err := s.get(name)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}

		val := L.ToNumber(2)
		c.add(float64(val))

		return 0
	}
}

func registerAbs(s *stat) lua.LGFunction {
	return func(L *lua.LState) int {
		name := L.ToString(1)
		err := s.add(name, &abs{})
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}

		return 0
	}
}
func registerAvg(s *stat) lua.LGFunction {
	return func(L *lua.LState) int {
		name := L.ToString(1)
		err := s.add(name, &avg{})
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}

		return 0
	}
}
func registerPer(s *stat) lua.LGFunction {
	return func(L *lua.LState) int {
		dur := L.ToString(2)
		duration, err := time.ParseDuration(dur)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}

		name := L.ToString(1)
		err = s.add(name, &per{interval: float64(duration)})
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}

		return 0
	}
}
