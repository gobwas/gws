package util

import (
	"github.com/yuin/gopher-lua"
	"net/http"
)

func HeadersToMap(h http.Header) map[string]string {
	m := make(map[string]string)
	for key := range h {
		m[key] = h.Get(key)
	}
	return m
}

func MapOfStringToTable(L *lua.LState, m map[string]string) *lua.LTable {
	table := L.NewTable()
	for key, value := range m {
		table.RawSetString(key, lua.LString(value))
	}
	return table
}

func MapOfInterfaceToTable(L *lua.LState, m map[string]interface{}) *lua.LTable {
	table := L.NewTable()
	for key, value := range m {
		switch v := value.(type) {
		case string:
			table.RawSetString(key, lua.LString(v))
		case int:
			table.RawSetString(key, lua.LNumber(v))
		case uint:
			table.RawSetString(key, lua.LNumber(v))
		case float64:
			table.RawSetString(key, lua.LNumber(v))
		}
	}
	return table
}

func HeadersFromTable(t *lua.LTable) (h http.Header) {
	h = make(http.Header, t.Len())
	t.ForEach(func(k, v lua.LValue) {
		h.Set(k.String(), v.String())
	})
	return
}
