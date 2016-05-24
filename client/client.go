package client

import (
	"flag"
	"github.com/chzyer/readline"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/cli/color"
	cliInput "github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/config"
	"github.com/gobwas/gws/util"
	"github.com/gobwas/gws/ws"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"time"
)

var limit = flag.Int("retry", 1, "try to reconnect x times")

const readLineTemp = "/tmp/gws_readline_client.tmp"

func Go(c config.Config) error {
	var conn *websocket.Conn
	var err error

	for i := 0; i < *limit; i++ {
		conn, err = getConn(c.URI, c.Headers)
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond * time.Duration(100*i))
	}
	if err != nil {
		cli.Printf(cli.PrefixTheEnd, "could not connect: %s", color.Red(err))
		return err
	}

	done := make(chan struct{})
	output, err := cliInput.ReadLineAsync(done, &readline.Config{
		Prompt:      cli.PaddingLeft + "> ",
		HistoryFile: readLineTemp,
	})
	if err != nil {
		return err
	}
	input := ws.ReadAsyncFromConn(done, conn)

	for {
		select {
		case in := <-input:
			if in.Err != nil {
				if in.Err == io.EOF {
					cli.Printf(cli.PrefixTheEnd, "%s %s", color.Magenta(in.Err), color.Red("server has closed connection"))
				} else {
					cli.Printf(cli.PrefixInfo, "%s %s", color.Magenta(in.Err), color.Red("unknown error"))
				}

				cli.Printf(cli.PrefixBlockEnd, "")
				return in.Err
			}

			cli.Printf(cli.PrefixIncoming, "%s: %s", color.Magenta(in.Kind), color.Cyan(string(in.Data)))

		case out := <-output:
			if out.Err != nil {
				cli.Printf(cli.PrefixTheEnd, "%s %s", color.Magenta(out.Err), color.Red("input closed"))
				return out.Err
			}

			err := ws.WriteToConn(conn, ws.TextMessage, out.Data)
			if err != nil {
				cli.Printf(cli.PrefixInfo, "%s", color.Red(err))
			}
		}
	}
}

func getConn(uri string, h http.Header) (*websocket.Conn, error) {
	conn, resp, err := ws.GetConn(uri, h)
	if config.Verbose {
		req, res, _ := util.DumpRequestResponse(resp)
		cli.Printf(cli.PrefixRaw, "%s", color.Green(string(req)))
		cli.Printf(cli.PrefixRaw, "%s", color.Cyan(string(res)))
	}
	if err != nil {
		cli.Printf(cli.PrefixInfo, "%s %s", color.Magenta(err), color.Red("could not connect"))
		return nil, err
	}

	cli.Printf(cli.PrefixInfo, "connected to %s", color.Green(uri))
	cli.Printf(cli.PrefixEmpty, "")

	return conn, nil
}
