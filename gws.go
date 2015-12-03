package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/gobwas/gws/server"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

type prefix string

const (
	empty      = " "
	input      = ">"
	incoming   = "<"
	info       = "–"
	theEnd     = "\u2020"
	blockStart = "("
	blockEnd   = ")"
	raw        = "r"
)

const headersSeparator = ";"
const headerAssignmentOperator = ":"

const (
	echo   = "echo"
	mirror = "mirror"
	prompt = "prompt"
)

var (
	url       = flag.String("u", "", "websocket server url")
	headers   = flag.String("H", "", fmt.Sprintf("list of headers to be passed during handshake\n\tformat:\n\t\t{ pair[ %q pair...] },\n\tpair:\n\t\t{ key %q value }", headersSeparator, headerAssignmentOperator))
	verbose   = flag.Bool("v", false, "show additional debugging info")
	limit     = flag.Int("x", 1, "try to reconnect x times")
	listen    = flag.String("l", "", "run ws server and listen this address")
	responder = &ResponderFlag{mirror, []string{echo, mirror, prompt}}
)

func init() {
	flag.Var(responder, "resp", fmt.Sprintf("how should server response on message (%s)", strings.Join(responder.e, ", ")))
}

var (
	red     = color.New(color.FgRed).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
)

type message struct {
	t MsgType
	b []byte
}

type MsgType int

const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

func (m MsgType) String() string {
	switch m {
	case TextMessage:
		return "TextMessage"
	case BinaryMessage:
		return "BinaryMessage"
	case CloseMessage:
		return "CloseMessage"
	case PingMessage:
		return "PingMessage"
	case PongMessage:
		return "PongMessage"
	default:
		return "UnknownMessage"
	}
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

func printF(p prefix, format string, c ...interface{}) {
	var (
		prefix, end string
	)

	prefix = strings.Repeat(" ", 2)
	end = fmt.Sprintf(" \n%s%s ", prefix, input)

	switch p {
	case blockStart, blockEnd:
		prefix = ""
		end = fmt.Sprintf(" \n")
	case input:
		prefix = strings.Repeat(" ", 2)
		end = ""
	case raw:
		fmt.Printf("\r%s\n", strings.Repeat(" ", 16))
		for _, l := range strings.Split(fmt.Sprintf(format, c...), "\n") {
			fmt.Printf("%s%s\n", strings.Repeat(" ", 4), l)
		}
		fmt.Print(magenta("\n"))

		return
	}

	fmt.Printf("\r%s%s %s%s", prefix, p, fmt.Sprintf(format, c...), end)
}

func wsWriter(conn *websocket.Conn, m <-chan []byte, e chan<- error) {
	for b := range m {
		writer, err := conn.NextWriter(websocket.TextMessage)
		if err != nil {
			e <- io.EOF
			return
		}

		_, err = writer.Write(b)
		if err != nil {
			e <- err
			return
		}

		err = writer.Close()
		if err != nil {
			e <- err
			return
		}

		if *verbose {
			printF(info, yellow("%s"), string(b))
		}
	}
}

func wsReader(conn *websocket.Conn, m chan<- message, e chan<- error) {
	for {
		t, r, err := conn.NextReader()
		if err != nil {
			e <- io.EOF
			return
		}

		msg, err := ioutil.ReadAll(r)
		if err != nil {
			e <- err
			return
		}

		m <- message{MsgType(t), msg}
	}
}

func ioReader(r io.Reader, to chan<- []byte, eof chan<- error) {
	var buf []byte
	reader := bufio.NewReader(r)

	for {
		b, err := reader.ReadByte()
		if err != nil {
			eof <- err
			return
		}

		if b == '\n' {
			to <- buf
			buf = nil
			printF(input, "")
			continue
		}

		buf = append(buf, b)
	}
}

func connect(u string, h http.Header) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{}
	conn, response, wsErr := dialer.Dial(*url, h)
	if *verbose {
		req, res, _ := dumpResponse(response)
		printF(raw, "%s", green(string(req)))
		printF(raw, "%s", cyan(string(res)))
	}
	if wsErr != nil {
		return nil, fmt.Errorf("dial websocket error: %s", wsErr)
	}

	return conn, nil
}

func main() {
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

		fmt.Println(server.Listen(*listen, h, r))
		return
	}

	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}

	// start to read input messages
	out := make(chan []byte)
	eof := make(chan error)
	go ioReader(os.Stdin, out, eof)

	var (
		inputClosed bool
		attempts    int
	)

try:
	for !inputClosed {
		select {
		case <-eof:
			close(out)
			inputClosed = true

		default:
			if attempts >= *limit {
				break try
			}

			printF(blockStart, "")
			printF(info, "establishing connection..(%d)", attempts)

			attempts++

			conn, err := connect(*url, h)
			if err != nil {
				printF(info, "%s %s", magenta(err), red("could not connect"))
				printF(blockEnd, "")
				continue try
			}

			printF(info, "connected to %s", green(*url))
			printF(empty, "")

			errors := make(chan error)
			in := make(chan message)

			go wsWriter(conn, out, errors)
			go wsReader(conn, in, errors)

			for {
				select {
				case err := <-errors:
					if err == io.EOF {
						printF(theEnd, "%s %s", magenta(err), red("server has closed connection"))
					} else {
						printF(info, "%s %s", magenta(err), red("unknown error"))
					}

					printF(blockEnd, "")

					continue try

				case msg := <-in:
					if *verbose {
						printF(incoming, "%s: %s", magenta(msg.t), cyan(string(msg.b)))
					} else {
						printF(incoming, "%s", cyan(string(msg.b)))
					}
				}
			}
		}
	}

	fmt.Printf("\r")
	os.Exit(0)
}

func dumpResponse(resp *http.Response) ([]byte, []byte, error) {
	if resp == nil {
		return nil, nil, fmt.Errorf("nil response")
	}

	if resp.Request == nil {
		return nil, nil, fmt.Errorf("nil request")
	}

	req, err := httputil.DumpRequest(resp.Request, false)
	if err != nil {
		return nil, nil, err
	}

	res, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return req, nil, err
	}

	return req, res, nil
}
