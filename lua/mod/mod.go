package mod

import "github.com/yuin/gopher-lua"

type Mod interface {
	Exports() lua.LGFunction
	Name() string
}
