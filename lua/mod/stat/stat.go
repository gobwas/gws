package stat

import (
	"github.com/yuin/gopher-lua"
	"time"
)

type Mod struct {
	statistics *statistics
}

func New() *Mod {
	return &Mod{newStatistics()}
}

func (m *Mod) Exports() lua.LGFunction {
	return func(L *lua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "name", lua.LString(m.Name()))
		mod.RawSetString("abs", L.NewClosure(registerAbs(m.statistics)))
		mod.RawSetString("avg", L.NewClosure(registerAvg(m.statistics)))
		mod.RawSetString("per", L.NewClosure(registerPer(m.statistics)))
		mod.RawSetString("add", L.NewClosure(increment(m.statistics)))
		mod.RawSetString("flush", L.NewClosure(flush(m.statistics)))
		mod.RawSetString("pretty", L.NewClosure(pretty(m.statistics)))

		L.Push(mod)
		return 1
	}
}

func (m *Mod) Name() string {
	const moduleName = "gws.stat"
	return moduleName
}

func pretty(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Push(lua.LString(s.pretty()))
		return 1
	}
}

func flush(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		results := L.NewTable()
		for key, c := range s.counters {
			results.RawSetString(key, lua.LNumber(c.flush()))
		}
		L.Push(results)
		return 1
	}
}

func increment(s *statistics) lua.LGFunction {
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

func registerAbs(s *statistics) lua.LGFunction {
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
func registerAvg(s *statistics) lua.LGFunction {
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
func registerPer(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		dur := L.ToString(2)
		duration, err := time.ParseDuration(dur)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}

		name := L.ToString(1)
		err = s.add(name, &per{
			interval: duration,
			stamp:    time.Now(),
		})
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}

		return 0
	}
}
