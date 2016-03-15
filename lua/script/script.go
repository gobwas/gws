package script

import (
	"github.com/gobwas/gws/lua/mod"
	"github.com/yuin/gopher-lua"
)

type Script struct {
	L *lua.LState
}

func New(f string, m ...mod.Mod) (*Script, error) {
	L := lua.NewState()
	if err := L.DoFile(f); err != nil {
		return nil, err
	}

	for _, module := range m {
		L.PreloadModule(module.Name(), module.Exports())
	}

	return &Script{L}, nil
}

func (s *Script) CallMain() error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal("main"),
		NRet:    0,
		Protect: true,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Script) CallSetup(thread *lua.LTable) error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal("setup"),
		NRet:    0,
		Protect: true,
	}, thread)
	if err != nil {
		return err
	}
	return nil
}

func (s *Script) CallTeardown(thread *lua.LTable) error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal("teardown"),
		NRet:    0,
		Protect: true,
	}, thread)
	if err != nil {
		return err
	}
	return nil
}

func (s *Script) CallTick(thread *lua.LTable) error {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal("tick"),
		NRet:    0,
		Protect: true,
	}, thread)
	if err != nil {
		return err
	}
	return nil
}

func (s *Script) CallReconnect(thread *lua.LTable) (bool, error) {
	err := s.L.CallByParam(lua.P{
		Fn:      s.L.GetGlobal("connect"),
		NRet:    1,
		Protect: true,
	}, thread)
	if err != nil {
		return false, err
	}
	ret := s.L.ToBool(-1)
	s.L.Pop(1)
	return ret, nil
}

func (s *Script) Close() {
	s.L.Close()
}
