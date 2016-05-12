package time

import (
	"github.com/gobwas/gws/ev"
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
	initTime       time.Time
	loop           *ev.Loop
	timers         map[uint32]*ev.Timer
	timeoutCounter uint32
}

func New(loop *ev.Loop) *Mod {
	return &Mod{
		initTime: time.Now(),
		loop:     loop,
		timers:   make(map[uint32]*ev.Timer),
	}
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

		mod.RawSetString("setTimeout", L.NewClosure(func(L *lua.LState) int {
			tm := L.ToNumber(1)
			cb := L.ToFunction(2)

			m.timeoutCounter++
			timeout := m.loop.Timeout(time.Duration(tm)*time.Millisecond, false, func() {
				L.CallByParam(lua.P{
					Fn:      cb,
					NRet:    0,
					Protect: false,
				})
			})
			m.timers[m.timeoutCounter] = timeout

			L.Push(lua.LNumber(m.timeoutCounter))

			return 1
		}))

		mod.RawSetString("setInterval", L.NewClosure(func(L *lua.LState) int {
			tm := L.ToNumber(1)
			cb := L.ToFunction(2)

			m.timeoutCounter++
			timeout := m.loop.Timeout(time.Duration(tm)*time.Millisecond, true, func() {
				L.CallByParam(lua.P{
					Fn:      cb,
					NRet:    0,
					Protect: false,
				})
			})
			m.timers[m.timeoutCounter] = timeout

			L.Push(lua.LNumber(m.timeoutCounter))

			return 1
		}))

		mod.RawSetString("unsetTimer", L.NewClosure(func(L *lua.LState) int {
			id := L.ToNumber(1)
			timeout, ok := m.timers[uint32(id)]
			if !ok {
				L.Push(lua.LString("unknown timeout"))
				return 1
			}

			timeout.Stop()

			return 0
		}))

		L.Push(mod)
		return 1
	}
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
