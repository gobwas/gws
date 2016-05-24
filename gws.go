package main

import (
	"flag"
	"fmt"
	"github.com/gobwas/gws/cli/color"
	"github.com/gobwas/gws/client"
	"github.com/gobwas/gws/config"
	"github.com/gobwas/gws/lua"
	"github.com/gobwas/gws/server"
	"io"
	"os"
	"strings"
)

const (
	modeServer = "server"
	modeClient = "client"
	modeScript = "script"
)

var modes = []string{modeServer, modeClient, modeScript}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s %s|%s|%s [options]\n", os.Args[0], modeClient, modeServer, modeScript)
		fmt.Fprintf(os.Stderr, "options:\n")
		flag.PrintDefaults()
	}
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}
	flag.CommandLine.Parse(os.Args[2:])

	cfg, err := config.Parse()
	if err != nil {
		flag.Usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case modeServer:
		err = server.Go(cfg)
	case modeClient:
		err = client.Go(cfg)
	case modeScript:
		err = lua.Go(cfg)
	default:
		err = fmt.Errorf("mode is required to be a one of `%s`; but `%s` given", color.Cyan(strings.Join(modes, "`, `")), color.Yellow(os.Args[1]))
	}

	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s %s\n\n", color.Red("error:"), err))
		os.Exit(1)
	}

	fmt.Printf("\r")
	os.Exit(0)
}
