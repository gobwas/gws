package time

import (
	"github.com/yuin/gopher-lua"
	"time"
)

const (
	durationMicroseconds = "us"
	durationMilliseconds = "ms"
	durationSeconds      = "s"
)

type Mod struct {
	initTime time.Time
}

func New() *Mod {
	return &Mod{time.Now()}
}

func (m *Mod) Exports() lua.LGFunction {
	return func(L *lua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "name", lua.LString(m.Name()))
		L.SetField(mod, "microseconds", lua.LString(durationMicroseconds))
		L.SetField(mod, "milliseconds", lua.LString(durationMilliseconds))
		L.SetField(mod, "seconds", lua.LString(durationSeconds))
		mod.RawSetString("now", L.NewClosure(func(L *lua.LState) int {
			now := inPrecision(time.Since(m.initTime), L.ToString(1))
			L.Push(lua.LNumber(now))
			return 1
		}))

		L.Push(mod)
		return 1
	}
}

func (m *Mod) Name() string {
	const moduleName = "gws.time"
	return moduleName
}

func inPrecision(dur time.Duration, p string) float64 {
	switch p {
	case durationMicroseconds:
		return dur.Seconds() * 1000000
	case durationMilliseconds:
		return dur.Seconds() * 1000
	case durationSeconds:
		return dur.Seconds()

	default:
		return float64(dur)
	}
}
