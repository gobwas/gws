package script

import (
	"fmt"
	"github.com/gobwas/gws/lua/mod"
	"github.com/yuin/gopher-lua"
)

const (
	functionMain      = "main"
	functionDone      = "done"
	functionSetup     = "setup"
	functionTeardown  = "teardown"
	functionTick      = "tick"
	functionReconnect = "reconnect"
)

type Script struct {
	L *lua.LState
}

func New(f string, m ...mod.Mod) (*Script, error) {
	L := lua.NewState()
	for _, module := range m {
		L.PreloadModule(module.Name(), module.Exports())
	}
	if err := L.DoFile(f); err != nil {
		return nil, err
	}

	return &Script{L}, nil
}

func (s *Script) CallMain() error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal(functionMain),
		NRet:    0,
		Protect: true,
	})
	if err != nil {
		return fmt.Errorf("call lua %q function error: %s", functionMain, err)
	}
	return nil
}

func (s *Script) CallDone() error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal(functionDone),
		NRet:    0,
		Protect: true,
	})
	if err != nil {
		return fmt.Errorf("call lua %q function error: %s", functionDone, err)
	}
	return nil
}

func (s *Script) CallSetup(thread *lua.LTable, index int) error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal(functionSetup),
		NRet:    0,
		Protect: true,
	}, thread, lua.LNumber(index))
	if err != nil {
		return fmt.Errorf("call lua %q function error: %s", functionSetup, err)
	}
	return nil
}

func (s *Script) CallTeardown(thread *lua.LTable) error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal(functionTeardown),
		NRet:    0,
		Protect: true,
	}, thread)
	if err != nil {
		return fmt.Errorf("call lua %q function error: %s", functionTeardown, err)
	}
	return nil
}

func (s *Script) CallTick(thread *lua.LTable) error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal(functionTick),
		NRet:    0,
		Protect: true,
	}, thread)
	if err != nil {
		return fmt.Errorf("call lua %q function error: %s", functionTick, err)
	}
	return nil
}

func (s *Script) CallReconnect(thread *lua.LTable) (bool, error) {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal(functionReconnect),
		NRet:    1,
		Protect: true,
	}, thread)
	if err != nil {
		return false, fmt.Errorf("call lua %q function error: %s", functionReconnect, err)
	}
	ret := s.L.ToBool(-1)
	s.L.Pop(1)
	return ret, nil
}

func (s *Script) Close() {
	s.L.Close()
}
