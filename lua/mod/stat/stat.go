package stat

import (
	"fmt"
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
		mod.RawSetString("new", L.NewClosure(registerNew(m.statistics)))
		mod.RawSetString("abs", L.NewClosure(registerAbs(m.statistics)))
		mod.RawSetString("avg", L.NewClosure(registerAvg(m.statistics)))
		mod.RawSetString("per", L.NewClosure(registerPer(m.statistics)))
		mod.RawSetString("add", L.NewClosure(registerAdd(m.statistics)))
		mod.RawSetString("flush", L.NewClosure(registerFlush(m.statistics)))
		mod.RawSetString("pretty", L.NewClosure(registerPretty(m.statistics)))

		L.Push(mod)
		return 1
	}
}

func (m *Mod) Name() string {
	const moduleName = "gws.stat"
	return moduleName
}

const (
	definitionFieldKind     = "kind"
	definitionFieldMeta     = "meta"
	definitionFieldInterval = "interval"
)

const (
	kindAbs int = iota
	kindAvg
	kindPer
)

func registerNew(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		name := L.ToString(1)
		for i := 2; ; i++ {
			def := L.ToTable(i)
			if def == nil {
				break
			}

			var meta map[string]interface{}
			m := def.RawGetString(definitionFieldMeta)
			if table, ok := m.(*lua.LTable); ok {
				meta = make(map[string]interface{})
				table.ForEach(func(key lua.LValue, value lua.LValue) {
					if key.Type() == lua.LTString {
						meta[key.String()] = value
					}
				})
			}

			var counter *counterSetup
			kind := int(def.RawGetString(definitionFieldKind).(lua.LNumber))
			switch kind {
			case kindAbs:
				abs, err := createAbs()
				if err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				counter = &counterSetup{abs, meta}

			case kindAvg:
				avg, err := createAvg()
				if err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				counter = &counterSetup{avg, meta}

			case kindPer:
				interval := def.RawGetString(definitionFieldInterval).(lua.LString)
				per, err := createPer(interval.String())
				if err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				counter = &counterSetup{per, meta}

			default:
				L.Push(lua.LString(fmt.Sprintf("unknown type of counter: %v", kind)))
				return 1
			}

			err := s.add(name, counter)
			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
		}

		return 0
	}
}

func registerPretty(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Push(lua.LString(s.pretty()))
		return 1
	}
}

func registerFlush(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		results := L.NewTable()
		for key, counters := range s.instances {
			sub := L.NewTable()
			for _, instance := range counters {
				tags := L.NewTable()
				for key, value := range instance.tags {
					tags.RawSetString(key, lua.LString(value))
				}
				meta := L.NewTable()
				for key, value := range instance.setup.meta {
					meta.RawSetString(key, value.(lua.LValue))
				}
				sub.RawSetString("value", lua.LNumber(instance.counter.flush()))
				sub.RawSetString("tags", tags)
				sub.RawSetString("meta", meta)
			}
			results.RawSetString(key, sub)
		}
		L.Push(results)
		return 1
	}
}

func registerAdd(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		name := L.ToString(1)
		value := L.ToNumber(2)
		tags := L.Get(3)

		var tagsMap map[string]string
		if table, ok := tags.(*lua.LTable); ok {
			tagsMap = make(map[string]string)
			table.ForEach(func(key lua.LValue, value lua.LValue) {
				if key.Type() == lua.LTString {
					tagsMap[key.String()] = value.String()
				}
			})
		}

		err := s.inc(name, float64(value), tagsMap)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		return 0
	}
}
func registerAbs(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		def := L.NewTable()
		def.RawSetString(definitionFieldKind, lua.LNumber(kindAbs))
		if meta := L.ToTable(1); meta != nil {
			def.RawSetString(definitionFieldMeta, meta)
		}
		L.Push(def)
		return 1
	}
}
func registerAvg(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		def := L.NewTable()
		def.RawSetString(definitionFieldKind, lua.LNumber(kindAvg))
		if meta := L.ToTable(1); meta != nil {
			def.RawSetString(definitionFieldMeta, meta)
		}
		L.Push(def)
		return 1
	}
}
func registerPer(s *statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		def := L.NewTable()
		def.RawSetString(definitionFieldKind, lua.LNumber(kindPer))
		def.RawSetString(definitionFieldInterval, L.Get(1))
		if meta := L.ToTable(2); meta != nil {
			def.RawSetString(definitionFieldMeta, meta)
		}
		L.Push(def)
		return 1
	}
}

func createAbs() (factory, error) {
	return func() counter {
		return &abs{}
	}, nil
}
func createAvg() (factory, error) {
	return func() counter {
		return &avg{}
	}, nil
}
func createPer(dur string) (factory, error) {
	duration, err := time.ParseDuration(dur)
	if err != nil {
		return nil, err
	}

	return func() counter {
		return &per{
			interval: duration,
			stamp:    time.Now(),
		}
	}, nil
}
