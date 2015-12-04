package client

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/gobwas/gws/ws"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

var (
	red     = color.New(color.FgRed).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
)

func Go(u string, h http.Header, r io.Reader, verbose bool, limit int) {
	// start to read input messages
	out := make(chan []byte)
	eof := make(chan error)
	go ioReader(r, out, eof)

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
			if attempts >= limit {
				break try
			}

			printF(blockStart, "")
			printF(info, "establishing connection..(%d)", attempts)

			attempts++

			dialer := &websocket.Dialer{}
			conn, resp, err := dialer.Dial(u, h)
			if verbose {
				req, res, _ := dumpResponse(resp)
				printF(raw, "%s", green(string(req)))
				printF(raw, "%s", cyan(string(res)))
			}
			if err != nil {
				printF(info, "%s %s", magenta(err), red("could not connect"))
				printF(blockEnd, "")
				continue try
			}

			printF(info, "connected to %s", green(u))
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
					if verbose {
						printF(incoming, "%s: %s", magenta(msg.t), cyan(string(msg.b)))
					} else {
						printF(incoming, "%s", cyan(string(msg.b)))
					}
				}
			}
		}
	}
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

		m <- message{ws.MsgType(t), msg}
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

type message struct {
	t ws.MsgType
	b []byte
}
