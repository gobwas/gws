package client

import (
	"flag"
	"fmt"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/cli/color"
	cliInput "github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/common"
	luaClient "github.com/gobwas/gws/lua/client"
	luaServer "github.com/gobwas/gws/lua/server"
	"github.com/gobwas/gws/util"
	"github.com/gobwas/gws/util/headers"
	"github.com/gobwas/gws/ws"
	"github.com/gorilla/websocket"
	"github.com/yuin/gopher-lua"
	"io"
	"net/http"
	"net/url"
	"stash.mail.ru/scm/ego/easygo.git/log"
	"strings"
	"sync"
	"time"
)

var (
	uri     = flag.String("client.url", ":3000", "websocket url to connect")
	limit   = flag.Int("client.try", 1, "try to reconnect x times")
	script  = flag.String("client.script", "", "use lua script to perform action")
	threads = flag.Int("client.threads", 1, "how many threads (clients) start to initialize with script")
)

const headerOrigin = "Origin"

//func getConn(uri url.URL, headers http.Header) (resp http.Response, conn websocket.Conn, err error) {
//	dialer := &websocket.Dialer{}
//	conn, resp, err = dialer.Dial(uri.String(), headers)
//	return
//}

type config struct {
	headers http.Header
	uri     *url.URL
}

func Go() error {
	u := *uri

	h, err := headers.Parse(common.Headers)
	if err != nil {
		return common.UsageError{err}
	}

	// prevent false error on parsing url
	if strings.Index(u, "://") == -1 {
		u = fmt.Sprintf("ws://%s", u)
	}
	uri, err := url.Parse(u)
	if err != nil {
		return err
	}
	if uri.Scheme == "" {
		uri.Scheme = "ws"
	}

	// by default, set the same origin
	// to avoid same origin policy check on connections
	if orig := h.Get(headerOrigin); orig == "" {
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
		h.Set(headerOrigin, orig.String())
	}

	c := config{
		uri:     uri,
		headers: h,
	}

	if *script != "" {
		return GoLua(c)
	}

	return GoIO(c)
}

func GoLua(c config) error {
	wg := sync.WaitGroup{}
	for i := 0; i < *threads; i++ {
		wg.Add(1)

		go func() {
			l := lua.NewState()
			l.PreloadModule(luaServer.ModuleName, luaServer.Exports)
			l.PreloadModule(luaClient.ModuleName, luaClient.Exports)
			if err := l.DoFile(*script); err != nil {
				log.Printf("load #%d lua script error: %s\n", i, err)
				l.Close()
				wg.Done()
				return
			}

			// create connection
			dialer := &websocket.Dialer{}
			conn, _, err := dialer.Dial(c.uri.String(), c.headers)
			if err != nil {
				log.Printf("connection for #%d lua script error: %s\n", i, err)
				l.Close()
				wg.Done()
				return
			}

			done := make(chan struct{})
			input := ws.ReadFromConn(conn, done)

			thread := l.NewTable()
			l.SetFuncs(thread, map[string]lua.LGFunction{
				"send": func(L *lua.LState) int {
					msg := L.ToString(1)
					err := ws.WriteToConn(conn, ws.TextMessage, []byte(msg))
					if err != nil {
						L.Push(lua.LString(err.Error()))
					} else {
						L.Push(lua.LString(""))
					}
					return 1
				},
				"receive": func(L *lua.LState) int {
					msg := <-input
					if msg.Err != nil {
						L.Push(lua.LString(""))
						L.Push(lua.LString(msg.Err.Error()))
					} else {
						L.Push(lua.LString(string(msg.Data)))
						L.Push(lua.LString(""))
					}
					return 2
				},
				"die": func(L *lua.LState) int {
					close(done)
					return 0
				},
				"set": func(L *lua.LState) int {
					//					log.Println("calling set with:", L.ToInt(3))
					return 0
				},
			})

			err = l.CallByParam(lua.P{
				Fn:      l.GetGlobal("setup"),
				NRet:    0,
				Protect: true,
			}, thread)
			if err != nil {
				log.Printf("setup #%d error: %s\n", i, err)
			}

			wg.Done()
			l.Close()
			conn.Close()
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
	input := ws.ReadFromConn(conn, done)

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
