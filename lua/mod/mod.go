package mod

import "github.com/yuin/gopher-lua"

type Module interface {
	Exports() lua.LGFunction
}

type Storage struct {
	data map[string]interface{}
}

func NewStorage() *Storage {
	return &Storage{
		data: make(map[string]interface{}),
	}
}

func (s *Storage) Set(key string, value interface{}) {
	s.data[key] = value
	return
}

func (s *Storage) Get(key string) (result interface{}) {
	result = s.data[key]
	return
}

func (s *Storage) ExportSet(L *lua.LState) *lua.LFunction {
	return L.NewClosure(func(L *lua.LState) int {
		key := L.ToString(1)
		value := L.Get(3)
		s.Set(key, value)
		return 0
	})
}

func (s *Storage) ExportGet(L *lua.LState) *lua.LFunction {
	return L.NewClosure(func(L *lua.LState) int {
		key := L.ToString(1)
		switch v := s.Get(key).(type) {
		case lua.LValue:
			L.Push(v)
		case string:
			L.Push(lua.LString(v))
		case int:
			L.Push(lua.LNumber(v))
		default:
			L.Push(lua.LNil)
		}
		return 1
	})
}

type Emitter struct {
	seq       uint32
	callbacks map[string][]*callback
	registry  map[desc]uint32
}

func NewEmitter() *Emitter {
	return &Emitter{
		callbacks: make(map[string][]*callback),
		registry:  make(map[desc]uint32),
	}
}

func (e *Emitter) Emit(name string, args ...interface{}) {
	if len(e.callbacks[name]) == 0 {
		return
	}
	for _, callback := range e.callbacks[name] {
		callback.cb(args...)
	}
}

func (e *Emitter) On(event string, cb func(...interface{})) uint32 {
	e.seq++
	e.callbacks[event] = append(e.callbacks[event], &callback{e.seq, cb})
	return e.seq
}

func (e *Emitter) Off(id uint32) {
	for evt, callbacks := range e.callbacks {
		for i, cb := range callbacks {
			if cb.id == id {
				copy(callbacks[i:], callbacks[i+1:])
				last := len(callbacks) + 1
				callbacks[last] = nil
				e.callbacks[evt] = callbacks[:last]
				return
			}
		}
	}
}

func (e *Emitter) ExportOn(L *lua.LState) *lua.LFunction {
	return L.NewClosure(func(L *lua.LState) int {
		name := L.ToString(1)
		cb := L.ToFunction(2)
		e.registry[desc{name, cb}] = e.On(name, func(args ...interface{}) {
			var callArgs []lua.LValue
			for _, arg := range args {
				switch v := arg.(type) {
				case string:
					callArgs = append(callArgs, lua.LString(v))
				case int, uint, float64:
					callArgs = append(callArgs, lua.LNumber(v))
				default:
					//
				}
			}
			L.CallByParam(lua.P{
				Fn:      cb,
				NRet:    0,
				Protect: false,
			}, callArgs...)
		})
		return 0
	})
}

func (e *Emitter) ExportOff(L *lua.LState) *lua.LFunction {
	return L.NewClosure(func(L *lua.LState) int {
		name := L.ToString(1)
		cb := L.ToFunction(2)
		if id, ok := e.registry[desc{name, cb}]; ok {
			e.Off(id)
		}
		return 0
	})
}

type callback struct {
	id uint32
	cb func(...interface{})
}

type desc struct {
	event string
	fn    *lua.LFunction
}
