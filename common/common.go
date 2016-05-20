// Package common brings common configuration flags and utils.
package common

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
	// BoolVar and StringVar are used here just for reading them
	// from other packages with pure common.{Verbose|Headers} (without *)
	flag.BoolVar(&Verbose, "v", false, "verbose output")
	flag.StringVar(&Headers, "h", "", fmt.Sprintf("list of headers to be passed during handshake (both in client or server)\n\tformat:\n\t\t{ pair[ %q pair...] },\n\tpair:\n\t\t{ key %q value }", headers.Separator, headers.AssignmentOperator))
}
