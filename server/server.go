package server

import (
	"errors"
	"flag"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/gobwas/glob"
	"github.com/gobwas/gws/cli/color"
	"github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/common"
	"github.com/gobwas/gws/util/headers"
	"github.com/gobwas/gws/ws"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	listen    = flag.String("server.listen", ":3000", "run ws server and listen this address")
	origin    = flag.String("server.origin", "", "use this glob pattern for server origin checks")
	heartbit  = flag.String("server.heartbit", "5s", "server statistics dump interval")
	responder = &ResponderFlag{null, []string{echo, mirror, prompt, null}}
)

func init() {
	flag.Var(responder, "server.responder", fmt.Sprintf("how should server response on message (%s)", strings.Join(responder.expect, ", ")))
}

const (
	echo   = "echo"
	mirror = "mirror"
	prompt = "prompt"
	null   = "null"
)

func Go() error {
	h, err := headers.Parse(common.Headers)
	if err != nil {
		return common.UsageError{err}
	}

	hb, err := time.ParseDuration(*heartbit)
	if err != nil {
		return common.UsageError{err}
	}

	var r Responder
	switch responder.Get() {
	case echo:
		r = EchoResponder
	case mirror:
		r = MirrorResponder
	case prompt:
		r = PromptResponder
	case null:
		r = DevNullResponder
	default:
		return errors.New("unknown responder type")
	}

	handler := newWsHandler(Config{
		Headers:  h,
		Origin:   *origin,
		Heartbit: hb,
	}, r)

	handler.Init()

	log.Println("ready to listen", *listen)
	return http.ListenAndServe(*listen, handler)
}

type wsHandler struct {
	mu sync.Mutex

	upgrader   websocket.Upgrader
	config     Config
	responder  Responder
	sig        chan os.Signal
	nextID     uint64
	connsCount uint64
	conns      map[uint64]connDescriptor

	requests uint64
}

type Config struct {
	Headers  http.Header
	Origin   string
	Heartbit time.Duration
}

type connDescriptor struct {
	conn   *websocket.Conn
	notice chan []byte
	done   <-chan struct{}
}

const headerOrigin = "Origin"

type Responder func(ws.Kind, []byte) ([]byte, error)

func newWsHandler(c Config, r Responder) *wsHandler {
	u := websocket.Upgrader{}

	if c.Origin != "" {
		originChecker := glob.MustCompile(c.Origin)
		u.CheckOrigin = func(r *http.Request) bool {
			return originChecker.Match(r.Header.Get(headerOrigin))
		}
	}

	return &wsHandler{
		upgrader:  u,
		config:    c,
		responder: r,
		sig:       make(chan os.Signal, 1),
		conns:     make(map[uint64]connDescriptor),
	}
}

func (h *wsHandler) Init() {
	signal.Notify(h.sig, os.Interrupt)
	go func() {
	listening:
		for _ = range h.sig {
			if h.connsCount == 0 {
				os.Exit(0)
			}

			h.mu.Lock()
			{
				var connId uint64
				if h.connsCount > 1 {
					var items []readline.PrefixCompleterInterface
					for id := range h.conns {
						items = append(items, readline.PcItem(string(id)))
					}
					completer := readline.NewPrefixCompleter(items...)

					r, err := input.Readline(&readline.Config{
						Prompt:       color.Green("> ") + "select connection id: ",
						AutoComplete: completer,
					})
					if err != nil {
						if err == readline.ErrInterrupt {
							os.Exit(0)
						}
						log.Println("readline error:", err)
						continue listening
					}

					connId, err = strconv.ParseUint(string(r), 10, 64)
					if err != nil {
						log.Println("readline error:", err)
						continue listening
					}
				} else {
					connId = 1
				}

				r, err := input.Readline(&readline.Config{
					Prompt:      color.Green("> ") + fmt.Sprintf("notification for the connection #%d: ", connId),
					HistoryFile: "/tmp/gws_readline_server_notice.tmp",
				})
				if err != nil {
					if err == readline.ErrInterrupt {
						os.Exit(0)
					}

					log.Println("readline error:", err)
					continue listening
				}

				h.conns[connId].notice <- r
			}
			h.mu.Unlock()
		}
	}()

	go func() {
		for range time.Tick(h.config.Heartbit) {
			v := atomic.SwapUint64(&h.requests, 0)
			log.Printf("RPS: (%d) %.2f\n", v, float64(v/uint64(h.config.Heartbit.Seconds())))
		}
	}()
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if common.Verbose {
		req, err := httputil.DumpRequest(r, false)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("new request", string(req))
	}

	// take lock on read new connection
	h.mu.Lock()
	conn, err := h.upgrader.Upgrade(w, r, h.config.Headers)
	if err != nil {
		log.Println(err)
		h.mu.Unlock()
		return
	}
	h.connsCount++
	h.nextID++
	id := h.nextID
	desc := connDescriptor{
		conn:   conn,
		notice: make(chan []byte, 1),
		done:   make(chan struct{}),
	}
	h.conns[id] = desc
	defer func() {
		conn.Close()
		h.mu.Lock()
		delete(h.conns, id)
		h.connsCount--
		h.mu.Unlock()

		if common.Verbose {
			log.Printf("connection #%d closed\n", id)
		}
	}()
	h.mu.Unlock()

	if common.Verbose {
		log.Printf("establised connection #%d from %q\n", id, r.RemoteAddr)
	}

	in := ws.ReadAsyncFromConn(desc.done, conn)

	for {
		select {
		case <-desc.done:
			return

		case notice := <-desc.notice:
			err := ws.WriteToConn(conn, ws.TextMessage, notice)
			if err != nil {
				log.Println("error reading from socket:", err)
				return
			}
			log.Printf("sent message to %d: %s\n", id, string(notice))

		case msg := <-in:
			atomic.AddUint64(&h.requests, 1)

			if msg.Err != nil {
				if msg.Err != io.EOF {
					log.Println("receive message error:", err)
				}

				return
			}
			if common.Verbose {
				log.Printf("received message from %d: %s\n", id, string(msg.Data))
			}

			h.mu.Lock()
			resp, err := h.responder(msg.Kind, msg.Data)
			h.mu.Unlock()
			if err != nil {
				log.Println("responder error:", err)
				return
			}

			if resp != nil {
				err := ws.WriteToConn(conn, msg.Kind, resp)
				if err != nil {
					log.Println(err)
					return
				}

				log.Printf("sent message to %d: %s\n", id, string(resp))
			}
		}
	}
}
