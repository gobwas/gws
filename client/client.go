package client

import (
	"flag"
	"fmt"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/cli/color"
	cliInput "github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/common"
	modStat "github.com/gobwas/gws/lua/mod/stat"
	modTime "github.com/gobwas/gws/lua/mod/time"
	"github.com/gobwas/gws/lua/script"
	"github.com/gobwas/gws/util"
	headersUtil "github.com/gobwas/gws/util/headers"
	"github.com/gobwas/gws/ws"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"stash.mail.ru/scm/ego/easygo.git/log"
	"strings"
	"sync"
	"time"
)

var (
	uri        = flag.String("client.url", ":3000", "websocket url to connect")
	limit      = flag.Int("client.try", 1, "try to reconnect x times")
	scriptFile = flag.String("client.script", "", "use lua script to perform action")
	threads    = flag.Int("client.threads", 1, "how many threads (clients) start to initialize with script")
)

const headerOrigin = "Origin"

type config struct {
	headers http.Header
	uri     *url.URL
}

func parseURL(rawURL string) (*url.URL, error) {
	// prevent false error on parsing url
	if strings.Index(rawURL, "://") == -1 {
		rawURL = fmt.Sprintf("ws://%s", rawURL)
	}
	uri, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if uri.Scheme == "" {
		uri.Scheme = "ws"
	}

	return uri, nil
}

func fillOriginHeader(headers http.Header, uri *url.URL) http.Header {
	// by default, set the same origin
	// to avoid same origin policy check on connections
	if headers.Get(headerOrigin) == "" {
		var s string
		switch uri.Scheme {
		case "wss":
			s = "https"
		default:
			s = "http"
		}
		orig := url.URL{
			Scheme: s,
			Host:   uri.Host,
		}
		headers.Set(headerOrigin, orig.String())
	}

	return headers
}

func getConfig() (c config, err error) {
	headers, err := headersUtil.Parse(common.Headers)
	if err != nil {
		err = common.UsageError{err}
		return
	}

	uri, err := parseURL(*uri)
	if err != nil {
		err = common.UsageError{err}
		return
	}

	c = config{
		uri:     uri,
		headers: fillOriginHeader(headers, uri),
	}

	return
}

func Go() error {
	c, err := getConfig()
	if err != nil {
		return err
	}

	if *scriptFile != "" {
		return GoLua(*scriptFile, c)
	} else {
		return GoIO(c)
	}
}

func safeClose(ch chan struct{}) {
	select {
	case <-ch:
	// done has already closed
	default:
		close(ch)
	}
}

type threadConn struct {
	conn *websocket.Conn
	kind ws.Kind
	err  error
}

func (w *threadConn) Send(b []byte) error {
	err := ws.WriteToConn(w.conn, w.kind, b)
	if err != nil {
		w.err = err
		return err
	}
	return nil
}
func (w *threadConn) Receive() ([]byte, error) {
	for {
		msg, err := ws.ReadFromConn(w.conn)
		if err != nil {
			w.err = err
			return nil, err
		}
		if msg.Kind == w.kind {
			return msg.Data, nil
		}
	}
	panic("unexpected loop leave")
}
func (w *threadConn) Close() error {
	return w.conn.Close()
}
func (w *threadConn) Error() error {
	return w.err
}

func getConnRaw(c config) (conn *websocket.Conn, err error) {
	dialer := &websocket.Dialer{}
	conn, _, err = dialer.Dial(c.uri.String(), c.headers)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
func getThreadConn(c config) (*threadConn, error) {
	conn, err := getConnRaw(c)
	if err != nil {
		return nil, err
	}

	return &threadConn{conn, ws.TextMessage, nil}, nil
}

func GoLua(scriptPath string, c config) error {
	wg := sync.WaitGroup{}
	wg.Add(*threads)
	for i := 0; i < *threads; i++ {
		go func() {
			luaScript, err := script.New(scriptPath, modStat.New(), modTime.New())
			if err != nil {
				log.Printf("create #%d lua script error: %s\n", i, err)
				wg.Done()
				return
			}
			defer func() {
				luaScript.Close()
				wg.Done()
			}()

			if i == 0 {
				luaScript.CallMain()
			}

			thread := NewThread()
			luaThread := ExportThread(thread, luaScript.L)

			// call setup on new thread
			if err := luaScript.CallSetup(luaThread); err != nil {
				log.Printf("setup #%d error: %s\n", i, err)
				return
			}

			for thread.NextTick() {
				if !thread.HasConn() {
					reconnect, err := luaScript.CallReconnect(luaThread)
					if err != nil {
						log.Printf("reconnect #%d error: %s\n", i, err)
						return
					}

					if !reconnect {
						if err := luaScript.CallTeardown(luaThread); err != nil {
							log.Printf("teardown #%d error: %s\n", i, err)
						}
						return
					}

					tc, err := getThreadConn(c)
					if err != nil {
						log.Println("thread connect error:", err)
						return
					}
					thread.SetConn(tc)

					continue
				}

				if thread.conn.(*threadConn).Error() != nil {
					thread.conn.Close()
					continue
				}

				if err := luaScript.CallTick(luaThread); err != nil {
					log.Printf("tick #%d error: %s\n", i, err)
					return
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

//func GoIO(u string, h http.Header, r io.Reader, verbose bool, limit int, l *lua.LState) error {
func GoIO(c config) error {
	var conn *websocket.Conn
	var err error
	for i := 0; i < *limit; i++ {
		conn, err = getConn(c.uri, c.headers)
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
	output, err := cliInput.ReadFromStdReadline(done)
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

func getConn(uri *url.URL, h http.Header) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{}
	conn, resp, err := dialer.Dial(uri.String(), h)
	if common.Verbose {
		req, res, _ := util.DumpRequestResponse(resp)
		cli.Printf(cli.PrefixRaw, "%s", color.Green(string(req)))
		cli.Printf(cli.PrefixRaw, "%s", color.Cyan(string(res)))
	}
	if err != nil {
		cli.Printf(cli.PrefixInfo, "%s %s", color.Magenta(err), color.Red("could not connect"))
		return nil, err
	}

	cli.Printf(cli.PrefixInfo, "connected to %s", color.Green(uri.String()))
	cli.Printf(cli.PrefixEmpty, "")

	return conn, nil
}
