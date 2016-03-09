package cmd

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gobwas/gws/util/headers"
)

var Verbose bool
var Headers string

var ErrExitZero = errors.New("exit")

type UsageError struct {
	Err error
}

func (e UsageError) Error() string {
	return e.Err.Error()
}

func init() {
	flag.BoolVar(&Verbose, "verbose", false, "verbose output")
	flag.StringVar(&Headers, "headers", "", fmt.Sprintf("list of headers to be passed during handshake (both in client or server)\n\tformat:\n\t\t{ pair[ %q pair...] },\n\tpair:\n\t\t{ key %q value }", headers.Separator, headers.AssignmentOperator))
}
