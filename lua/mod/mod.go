package mod

import "github.com/yuin/gopher-lua"

type Module interface {
	Exports() lua.LGFunction
}
