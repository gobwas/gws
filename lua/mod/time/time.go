package time

import (
	"github.com/yuin/gopher-lua"
	"time"
)

type durationKind int

const (
	durationMicroseconds durationKind = iota
	durationMilliseconds
	durationSeconds
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
		L.SetField(mod, "us", lua.LNumber(durationMicroseconds))
		L.SetField(mod, "ms", lua.LNumber(durationMilliseconds))
		L.SetField(mod, "s", lua.LNumber(durationSeconds))
		mod.RawSetString("now", L.NewClosure(func(L *lua.LState) int {
			now := inPrecision(time.Since(m.initTime), durationKind(L.ToNumber(1)))
			L.Push(lua.LNumber(now))
			return 1
		}))
		mod.RawSetString("sleep", L.NewClosure(func(L *lua.LState) int {
			arg := L.ToString(1)
			duration, err := time.ParseDuration(arg)
			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
			time.Sleep(duration)
			return 0
		}))

		L.Push(mod)
		return 1
	}
}

func (m *Mod) Name() string {
	const moduleName = "gws.time"
	return moduleName
}

func inPrecision(dur time.Duration, p durationKind) float64 {
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
