package server

import (
	"errors"
	"flag"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/gobwas/gws/cli/color"
	"github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/config"
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
	origin    = flag.String("origin", "", "use this glob pattern for server origin checks")
	responder = &ResponderFlag{null, []string{echo, mirror, prompt, null}}
)

func init() {
	flag.Var(responder, "response", fmt.Sprintf("how should server response on message (%s)", strings.Join(responder.expect, ", ")))
}

const (
	echo   = "echo"
	mirror = "mirror"
	prompt = "prompt"
	null   = "null"
)

func Go(c config.Config) error {
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

	handler, err := newWsHandler(Config{
		Headers:  c.Headers,
		Origin:   *origin,
		StatDump: c.StatDump,
	}, r)
	if err != nil {
		return err
	}

	handler.Init()

	log.Println("ready to listen", c.Addr)
	return http.ListenAndServe(c.Addr, handler)
}

type wsHandler struct {
	mu sync.Mutex

	upgrader   ws.Upgrader
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
	StatDump time.Duration
}

type connDescriptor struct {
	conn   *websocket.Conn
	notice chan []byte
	done   <-chan struct{}
}

const headerOrigin = "Origin"

type Responder func(ws.Kind, []byte) ([]byte, error)

func newWsHandler(c Config, r Responder) (*wsHandler, error) {
	return &wsHandler{
		upgrader:  ws.GetUpgrader(ws.UpgradeConfig{c.Origin, c.Headers}),
		config:    c,
		responder: r,
		sig:       make(chan os.Signal, 1),
		conns:     make(map[uint64]connDescriptor),
	}, nil
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

					r, err := input.ReadLine(&readline.Config{
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

				r, err := input.ReadLine(&readline.Config{
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
		for range time.Tick(h.config.StatDump) {
			v := atomic.SwapUint64(&h.requests, 0)
			log.Printf("RPS: (%d) %.2f\n", v, float64(v/uint64(h.config.StatDump.Seconds())))
		}
	}()
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if config.Verbose {
		req, err := httputil.DumpRequest(r, false)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("new request", string(req))
	}

	// take lock on read new connection
	h.mu.Lock()
	conn, err := h.upgrader(w, r)
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

		if config.Verbose {
			log.Printf("connection #%d closed\n", id)
		}
	}()
	h.mu.Unlock()

	if config.Verbose {
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
			if config.Verbose {
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

				if config.Verbose {
					log.Printf("sent message to %d: %s\n", id, string(resp))
				}
			}
		}
	}
}
