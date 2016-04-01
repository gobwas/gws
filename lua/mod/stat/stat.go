package stat

import (
	"fmt"
	"github.com/gobwas/gws/stat"
	"github.com/gobwas/gws/stat/counter/abs"
	"github.com/gobwas/gws/stat/counter/avg"
	"github.com/gobwas/gws/stat/counter/per"
	"github.com/yuin/gopher-lua"
	"time"
)

type Mod struct {
	statistics *stat.Statistics
}

func New(s *stat.Statistics) *Mod {
	return &Mod{s}
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

func registerNew(s *stat.Statistics) lua.LGFunction {
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

			var config stat.Config
			kind := int(def.RawGetString(definitionFieldKind).(lua.LNumber))
			switch kind {
			case kindAbs:
				abs, err := absFactory()
				if err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				config = stat.Config{abs, meta}

			case kindAvg:
				avg, err := avgFactory()
				if err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				config = stat.Config{avg, meta}

			case kindPer:
				interval := def.RawGetString(definitionFieldInterval).(lua.LString)
				per, err := perFactory(interval.String())
				if err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				config = stat.Config{per, meta}

			default:
				L.Push(lua.LString(fmt.Sprintf("unknown type of counter: %v", kind)))
				return 1
			}

			err := s.Setup(name, config)
			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
		}

		return 0
	}
}

func registerPretty(s *stat.Statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Push(lua.LString(s.Pretty()))
		return 1
	}
}

func registerFlush(s *stat.Statistics) lua.LGFunction {
	return func(L *lua.LState) int {
		var index int
		results := L.NewTable()

		for name, metrics := range s.Metrics {
			for _, metric := range metrics {
				tags := L.NewTable()
				for key, value := range metric.Tags {
					tags.RawSetString(key, lua.LString(value))
				}

				for _, instance := range metric.Instances {
					meta := L.NewTable()
					for key, value := range instance.Meta {
						meta.RawSetString(key, value.(lua.LValue))
					}

					sub := L.NewTable()
					sub.RawSetString("name", lua.LString(name))
					sub.RawSetString("kind", lua.LString(instance.Counter.Kind()))
					sub.RawSetString("value", lua.LNumber(instance.Counter.Flush()))
					sub.RawSetString("tags", tags)
					sub.RawSetString("meta", meta)

					results.RawSetInt(index, sub)
					index++
				}
			}
		}
		L.Push(results)
		return 1
	}
}

func registerAdd(s *stat.Statistics) lua.LGFunction {
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

		err := s.Increment(name, float64(value), tagsMap)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		return 0
	}
}
func registerAbs(s *stat.Statistics) lua.LGFunction {
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
func registerAvg(s *stat.Statistics) lua.LGFunction {
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
func registerPer(s *stat.Statistics) lua.LGFunction {
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

func absFactory() (stat.CounterFactory, error) {
	return func() stat.Counter {
		return abs.New()
	}, nil
}
func avgFactory() (stat.CounterFactory, error) {
	return func() stat.Counter {
		return avg.New()
	}, nil
}
func perFactory(dur string) (stat.CounterFactory, error) {
	duration, err := time.ParseDuration(dur)
	if err != nil {
		return nil, err
	}

	return func() stat.Counter {
		return per.New(duration)
	}, nil
}
