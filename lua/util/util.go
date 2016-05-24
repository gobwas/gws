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

func HeadersFromTable(t *lua.LTable) (h http.Header) {
	h = make(http.Header, t.Len())
	t.ForEach(func(k, v lua.LValue) {
		h.Set(k.String(), v.String())
	})
	return
}
