package server

import (
	"github.com/yuin/gopher-lua"
)

const ModuleName = "server"

func Exports(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), exports)

	L.SetField(mod, "name", lua.LString(ModuleName))

	L.Push(mod)

	return 1
}

var exports = map[string]lua.LGFunction{
	"myfunc": myfunc,
}

func myfunc(L *lua.LState) int {
	return 0
}
