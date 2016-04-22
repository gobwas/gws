package client

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/cli/color"
	cliInput "github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/client/ev"
	evWS "github.com/gobwas/gws/client/ev/ws"
	"github.com/gobwas/gws/common"
	"github.com/gobwas/gws/lua/mod/runtime"
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
	"strings"
	"sync"
	"time"
)

var (
	uri        = flag.String("client.url", ":3000", "websocket url to connect")
	limit      = flag.Int("client.try", 1, "try to reconnect x times")
	scriptFile = flag.String("client.script", "", "use lua script to perform action")
	threads    = flag.Int("client.threads", 1, "how many threads (clients) start to initialize with script")
	timeout    = flag.String("client.script.timeout", "0", "client script run timeout")
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
func getThreadConn(c config) (*threadConn, error) {
	conn, err := getConnRaw(c)
	if err != nil {
		return nil, err
	}

	return &threadConn{conn, ws.TextMessage, nil}, nil
}

var (
	statStarted   = "gws_thread_started"
	statCompleted = "gws_thread_completed"
	statError     = "gws_thread_error"
)

func GoLua(scriptPath string, c config) error {
	var code string
	if script, err := ioutil.ReadFile(scriptPath); err != nil {
		return err
	} else {
		code = string(script)
	}

	luaScript := script.New()
	defer luaScript.Shutdown()
	luaScript.HijackOutput(color.Green("master > "), os.Stderr)

	loop := ev.NewLoop()
	loop.Register(evWS.NewHandler(), 100)

	master := runtime.New(loop, true)
	master.Set("url", c.uri.String())
	master.Set("headers", headersToMap(c.headers))

	luaScript.Preload("runtime", master)
	luaScript.Preload("stat", modStat.New(stat.New()))
	luaScript.Preload("time", modTime.New(loop))
	luaScript.Preload("ws", modWS.New(loop))

	err := luaScript.Do(code)
	if err != nil {
		log.Printf("run lua script error: %s", err)
		return err
	}

	loop.Run()
	loop.Teardown(func() {
		master.Emit("exit")
	})
	<-loop.Done()

	os.Exit(0)
	return nil
}

func headersToMap(h http.Header) map[string]string {
	m := make(map[string]string)
	for key := range h {
		m[key] = h.Get(key)
	}
	return m
}

//
//func GoLua(scriptPath string, c config) error {
//	statistics := stat.New()
//	statistics.Setup(statStarted, stat.Config{
//		Factory: func() stat.Counter {
//			return abs.New()
//		},
//	})
//	statistics.Setup(statCompleted, stat.Config{
//		Factory: func() stat.Counter {
//			return abs.New()
//		},
//	})
//	statistics.Setup(statError, stat.Config{
//		Factory: func() stat.Counter {
//			return abs.New()
//		},
//	})
//
//	moduleStat := modStat.New(statistics)
//	moduleTime := modTime.New()
//
//	// read file once to prevent max open files unnecessary error
//	scriptBytes, err := ioutil.ReadFile(scriptPath)
//	if err != nil {
//		return err
//	}
//	scriptCode := string(scriptBytes)
//
//	luaScript, err := script.New(scriptCode, moduleStat, moduleTime)
//	if err != nil {
//		log.Printf("create global lua script error: %s\n", err)
//		return err
//	}
//
//	var ctx context.Context
//	var cancel context.CancelFunc
//	if c.timeout == 0 {
//		ctx, cancel = context.WithCancel(context.Background())
//	} else {
//		ctx, cancel = context.WithTimeout(context.Background(), c.timeout)
//	}
//
//	go func() {
//		c := make(chan os.Signal, 1)
//		signal.Notify(c, os.Interrupt)
//		s := <-c
//		// todo printer clear ?
//		fmt.Print("\r")
//		fmt.Println(color.Cyan(cli.PrefixTheEnd), s.String())
//		cancel()
//		s = <-c
//		fmt.Print("\r")
//		fmt.Println(color.Red(cli.PrefixTheEnd), color.Yellow(s.String()+"x2"))
//		os.Exit(1)
//	}()
//
//	if err := luaScript.CallMain(ctx); err != nil {
//		return err
//	}
//
//	printer := display{
//		lines: []lineFn{
//			getStatLineFn(statistics),
//		},
//		done: make(chan struct{}),
//	}
//	printer.begin()
//
//	results := make(chan error, *threads)
//	for i := 0; i < *threads; i++ {
//		go func(i int) {
//			time.Sleep(time.Duration(rand.Int63n(int64(*threads))) * time.Millisecond * 2)
//			statistics.Increment(statStarted, 1, nil)
//
//			luaScript, err := script.New(scriptCode, moduleStat, moduleTime)
//			if err != nil {
//				results <- err
//				return
//			}
//			defer func() {
//				luaScript.Close()
//			}()
//
//			thread := NewThread()
//			luaThread := ExportThread(thread, luaScript.l)
//
//			// call setup on new thread
//			if err := luaScript.CallSetup(ctx, luaThread, i); err != nil {
//				results <- err
//				return
//			}
//
//			for thread.NextTick() {
//				select {
//				case <-ctx.Done():
//					results <- ctx.Err()
//					return
//				default:
//					//
//				}
//
//				if !thread.HasConn() {
//					reconnect, err := luaScript.CallReconnect(ctx, luaThread)
//					if err != nil {
//						results <- err
//						return
//					}
//
//					if !reconnect {
//						if err := luaScript.CallTeardown(ctx, luaThread); err != nil {
//							results <- err
//						} else {
//							results <- nil
//						}
//						return
//					}
//
//					tc, err := getThreadConn(c)
//					if err != nil {
//						results <- err
//						return
//					}
//					thread.SetConn(tc)
//
//					continue
//				}
//
//				if thread.conn.(*threadConn).Error() != nil {
//					thread.conn.Close()
//					thread.conn = nil
//					continue
//				}
//
//				if err := luaScript.CallTick(ctx, luaThread); err != nil {
//					results <- err
//					return
//				}
//			}
//			results <- nil
//		}(i)
//	}
//
//	var stop bool
//	for i := 0; i < *threads && !stop; i++ {
//		select {
//		case <-ctx.Done():
//			stop = true
//		case err := <-results:
//			if err != nil {
//				statistics.Increment(statError, 1, nil)
//				if common.Verbose {
//					log.Println(err)
//				}
//			} else {
//				statistics.Increment(statCompleted, 1, nil)
//			}
//		}
//	}
//
//	printer.stop()
//
//	doneCtx, _ := context.WithTimeout(context.Background(), time.Second)
//	if err := luaScript.CallDone(doneCtx); err != nil {
//		log.Println(err)
//		return err
//	}
//
//	printer.update()
//
//	return nil
//}

func getStatLineFn(statistics *stat.Statistics) lineFn {
	return func() string {
		return statistics.Pretty()
	}
}

type lineFn func() string

type display struct {
	index  int64
	height int
	lines  []lineFn
	done   chan struct{}
}

func (d *display) begin() {
	ticker := time.Tick(time.Millisecond * 100)
	go func() {
		for {
			select {
			case <-ticker:
				d.update()
			case <-d.done:
				return
			}
		}
	}()
}

func (d *display) update() {
	buf := &bytes.Buffer{}
	for _, line := range d.lines {
		fmt.Fprintln(buf, line())
	}

	if d.height > 0 {
		fmt.Printf("\033[%dA", d.height)
	}
	d.height = bytes.Count(buf.Bytes(), []byte{'\n'})
	io.Copy(os.Stderr, buf)
}

func (d *display) stop() {
	close(d.done)
}

var threadStat = sync.Mutex{}

func updateThreadStat(started, completed, failed int32) {
	if !common.Verbose {
		threadStat.Lock()
		fmt.Printf("\rstarted: %d;\tcompleted: %d;\tfailed: %d;\t", started, completed, failed)
		threadStat.Unlock()
	}
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
