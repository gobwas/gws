package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/gobwas/gws/client"
	"github.com/gobwas/gws/server"
	"net/http"
	"os"
	"strings"
)

const headersSeparator = ";"
const headerAssignmentOperator = ":"

const (
	echo   = "echo"
	mirror = "mirror"
	prompt = "prompt"
)

var (
	url       = flag.String("u", "", "websocket url to connect")
	headers   = flag.String("H", "", fmt.Sprintf("list of headers to be passed during handshake\n\tformat:\n\t\t{ pair[ %q pair...] },\n\tpair:\n\t\t{ key %q value }", headersSeparator, headerAssignmentOperator))
	listen    = flag.String("l", "", "run ws server and listen this address")
	verbose   = flag.Bool("v", false, "show additional debugging info")
	limit     = flag.Int("x", 1, "try to reconnect x times")
	script    = flag.String("s", "", "use lua script to perform action")
	origin    = flag.String("o", "", "use this glob pattern for server origin checks")
	responder = &ResponderFlag{mirror, []string{echo, mirror, prompt}}
	red       = color.New(color.FgRed).SprintFunc()
)

func main() {
	flag.Var(responder, "resp", fmt.Sprintf("how should server response on message (%s)", strings.Join(responder.e, ", ")))
	flag.Parse()

	var h http.Header
	if *headers != "" {
		h = make(http.Header)
		for _, pair := range strings.Split(*headers, headersSeparator) {
			i := strings.Index(pair, headerAssignmentOperator)
			if i == -1 {
				fmt.Println(red("malformed headers value"))
			}

			h.Add(strings.TrimSpace(pair[:i]), strings.TrimSpace(pair[i+1:]))
		}
	}

	if *listen != "" {
		var r server.Responder
		switch responder.Get() {
		case echo:
			r = server.EchoResponder
		case mirror:
			r = server.MirrorResponder
		case prompt:
			r = server.PromptResponder
		default:
			fmt.Println(red("unknown responder type"))
			os.Exit(1)
		}

		fmt.Println(server.Listen(*listen, server.Config{h, *origin}, r))

		os.Exit(0)
		return
	}

	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}

	client.Go(*url, h, os.Stdin, *verbose, *limit)

	fmt.Printf("\r")
	os.Exit(0)
}

type ResponderFlag struct {
	v string
	e []string
}

func (r *ResponderFlag) Set(s string) error {
	for _, e := range r.e {
		if e == s {
			r.v = s
			return nil
		}
	}

	return fmt.Errorf("expecting one of %s", r.e)
}
func (r ResponderFlag) String() string {
	return r.v
}
func (r ResponderFlag) Get() interface{} {
	return r.v
}
