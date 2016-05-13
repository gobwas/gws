package client

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gobwas/gws/bufio"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/cli/color"
	cliInput "github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/common"
	"github.com/gobwas/gws/display"
	"github.com/gobwas/gws/ev"
	evWS "github.com/gobwas/gws/ev/ws"
	modRuntime "github.com/gobwas/gws/lua/mod/runtime"
	modStat "github.com/gobwas/gws/lua/mod/stat"
	modTime "github.com/gobwas/gws/lua/mod/time"
	modWS "github.com/gobwas/gws/lua/mod/ws"
	"github.com/gobwas/gws/lua/script"
	"github.com/gobwas/gws/stat"
	"github.com/gobwas/gws/util"
	headersUtil "github.com/gobwas/gws/util/headers"
	"github.com/gobwas/gws/ws"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

var (
	uri        = flag.String("u", ":3000", "websocket url to connect")
	limit      = flag.Int("retry", 1, "try to reconnect x times")
	scriptFile = flag.String("s", "", "use lua script to define client actions")
	timeout    = flag.String("t", "0", "client script run timeout")
)

const headerOrigin = "Origin"

type config struct {
	headers http.Header
	uri     *url.URL
	timeout time.Duration
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

	t, err := time.ParseDuration(*timeout)
	if err != nil {
		err = common.UsageError{err}
		return
	}

	c = config{
		uri:     uri,
		headers: fillOriginHeader(headers, uri),
		timeout: t,
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

func initRunTime(loop *ev.Loop, c config) *modRuntime.Runtime {
	rtime := modRuntime.New(loop)
	rtime.Set("url", c.uri.String())
	rtime.Set("headers", headersToMap(c.headers))
	return rtime
}

func GoLua(scriptPath string, c config) error {
	var code string
	if script, err := ioutil.ReadFile(scriptPath); err != nil {
		return err
	} else {
		code = string(script)
	}

	stats := stat.New()

	luaOutputBuffer := bytes.NewBuffer(make([]byte, 0, 1<<13))
	luaStdout := bufio.NewWriter(luaOutputBuffer, 1<<13)

	systemStdout := bytes.NewBuffer(make([]byte, 0, 1024))

	printer := display.NewDisplay(os.Stderr, display.Config{
		TabSize:  4,
		Interval: time.Millisecond * 100,
	})
	printer.Row().Col(-1, -1, func() string {
		return stats.Pretty()
	})
	printer.Row().Col(256, 9, func() (str string) {
		luaStdout.Dump()
		str = luaOutputBuffer.String()
		luaOutputBuffer.Reset()
		return
	})
	printer.Row().Col(256, 3, func() (str string) {
		str = systemStdout.String()
		return
	})
	printer.Begin()
	defer printer.Stop()
	defer printer.Render()

	cancel := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		s := <-c
		fmt.Fprintln(systemStdout, color.Cyan(cli.PrefixTheEnd), s.String())
		fmt.Fprintln(systemStdout, color.Cyan("stopping softly.."))
		close(cancel)
		s = <-c
		fmt.Fprintln(systemStdout, color.Red(cli.PrefixTheEnd), color.Yellow(s.String()+"x2"))
		fmt.Fprintln(systemStdout, color.Red("stopping hardly.."))
		printer.Stop()
		os.Exit(1)
	}()

	luaScript := script.New()
	defer luaScript.Shutdown()

	luaScript.HijackOutput(bufio.NewPrefixWriter(luaStdout, color.Green("master > ")))

	loop := ev.NewLoop()
	loop.Register(evWS.NewHandler(), 100)

	sharedStat := modStat.New(stats)

	var wg sync.WaitGroup
	var threads int
	rtime := initRunTime(loop, c)
	rtime.SetForkFn(func() error {
		go func(id int) {
			defer wg.Done()

			luaScript := script.New()
			defer luaScript.Shutdown()
			luaScript.HijackOutput(bufio.NewPrefixWriter(luaStdout, color.Green(fmt.Sprintf("thread %.2d > ", id))))

			loop := ev.NewLoop()
			loop.Register(evWS.NewHandler(), 100)

			rtime := initRunTime(loop, c)
			rtime.Set("id", id)

			luaScript.Preload("runtime", rtime)
			luaScript.Preload("stat", sharedStat)
			luaScript.Preload("time", modTime.New(loop))
			luaScript.Preload("ws", modWS.New(loop))

			err := luaScript.Do(code)
			if err != nil {
				log.Printf("run forked lua script error: %s", err)
			}

			loop.Run()
			loop.Teardown(func() {
				rtime.Emit("exit")
			})

			waitLoop(cancel, loop)
		}(threads)

		wg.Add(1)
		threads++

		return nil
	})

	luaScript.Preload("runtime", rtime)
	luaScript.Preload("stat", sharedStat)
	luaScript.Preload("time", modTime.New(loop))
	luaScript.Preload("ws", modWS.New(loop))

	err := luaScript.Do(code)
	if err != nil {
		log.Printf("run lua script error: %s", err)
		return err
	}

	loop.Run()
	loop.Teardown(func() {
		wg.Wait()
		rtime.Emit("exit")
	})

	waitLoop(cancel, loop)
	wg.Wait()

	return nil
}

func waitLoop(cancel chan struct{}, loop *ev.Loop) {
	select {
	case <-loop.Done():
	case <-cancel:
		loop.Stop()
		shutdown := time.NewTimer(time.Second * 4)
		select {
		case <-loop.Done():
		case <-shutdown.C:
			loop.Shutdown()
		}
	}
}

func headersToMap(h http.Header) map[string]string {
	m := make(map[string]string)
	for key := range h {
		m[key] = h.Get(key)
	}
	return m
}

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
