package main

import (
	"flag"
	"fmt"
	"github.com/gobwas/gws/cli/color"
	"github.com/gobwas/gws/client"
	"github.com/gobwas/gws/common"
	"github.com/gobwas/gws/server"
	"os"
	"strings"
)

const (
	modeServer = "server"
	modeClient = "client"
)

var modes = []string{modeServer, modeClient}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s %s|%s [options]\n", os.Args[0], modeClient, modeServer)
		fmt.Fprintf(os.Stderr, "options:\n")
		flag.PrintDefaults()
	}
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}
	flag.CommandLine.Parse(os.Args[2:])

	var err error
	switch os.Args[1] {
	case modeServer:
		err = server.Go()
	case modeClient:
		err = client.Go()
	default:
		err = fmt.Errorf("mode is required to be a one of `%s`; but `%s` given", color.Cyan(strings.Join(modes, "`, `")), color.Yellow(flag.Arg(0)))
	}

	if err != nil && err != common.ErrExitZero {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s %s\n\n", color.Red("error:"), err))
		if _, ok := err.(common.UsageError); ok {
			flag.Usage()
		}
		os.Exit(1)
	}

	fmt.Printf("\r")
	os.Exit(0)
}
