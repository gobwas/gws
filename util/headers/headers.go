package headers

import (
	"errors"
	"net/http"
	"strings"
)

var ErrorMalformedHeaderString = errors.New("malformed headers")

const Separator = ";"
const AssignmentOperator = ':'

func ParseOne(s string) (key, value string, err error) {
	for i := 0; i < len(s); i++ {
		if s[i] == AssignmentOperator {
			key, value = strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
			return
		}
	}
	return s, "", ErrorMalformedHeaderString
}

func Parse(s string) (h http.Header, err error) {
	h = make(http.Header)
	if s == "" {
		return
	}
	for _, pair := range strings.Split(s, Separator) {
		i := strings.Index(pair, string(AssignmentOperator))
		if i == -1 {
			err = ErrorMalformedHeaderString
			return
		}

		h.Add(strings.TrimSpace(pair[:i]), strings.TrimSpace(pair[i+1:]))
	}
	return
}
