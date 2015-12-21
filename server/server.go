package server

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/gobwas/glob"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
)

type wsHandler struct {
	respLock  sync.Mutex
	upgrader  websocket.Upgrader
	config    Config
	responder Responder
}

type Config struct {
	Headers http.Header
	Origin  string
	Verbose bool
}

const headerOrigin = "Origin"

type Responder func(int, []byte) ([]byte, bool, error)

func newWsHandler(c Config, r Responder) *wsHandler {
	u := websocket.Upgrader{}

	if c.Origin != "" {
		originChecker := glob.New(c.Origin)
		u.CheckOrigin = func(r *http.Request) bool {
			return originChecker.Match(r.Header.Get(headerOrigin))
		}
	}

	return &wsHandler{
		upgrader:  u,
		config:    c,
		responder: r,
	}
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.config.Verbose {
		req, err := httputil.DumpRequest(r, false)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("new request", string(req))
	}

	// take lock on read new connection
	h.respLock.Lock()
	conn, err := h.upgrader.Upgrade(w, r, h.config.Headers)
	h.respLock.Unlock()
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	log.Println("establised new connection from", r.RemoteAddr)

	for {
		t, r, err := conn.NextReader()
		if err != nil {
			log.Println(err)
			return
		}

		msg, err := ioutil.ReadAll(r)
		if err != nil {
			log.Println(err)
			return
		}

		h.respLock.Lock()
		log.Println("received message:", string(msg))
		resp, should, err := h.responder(t, msg)
		h.respLock.Unlock()
		if err != nil {
			log.Println("responder error:", err)
			return
		}

		if should {
			writer, err := conn.NextWriter(t)
			if err != nil {
				log.Println(err)
				return
			}

			_, err = writer.Write(resp)
			if err != nil {
				log.Println(err)
				return
			}

			err = writer.Close()
			if err != nil {
				log.Println(err)
				return
			}

			log.Println("sent message:", string(resp))
		}
	}
}

func Listen(addr string, c Config, r Responder) error {
	log.Println("ready to listen", addr)
	return http.ListenAndServe(addr, newWsHandler(c, r))
}

func EchoResponder(t int, msg []byte) ([]byte, bool, error) {
	return msg, true, nil
}

func MirrorResponder(t int, msg []byte) (r []byte, ok bool, err error) {
	if t != websocket.TextMessage {
		return
	}

	resp := []rune(string(msg))
	for i, l := 0, len(resp)-1; i < len(resp)/2; i, l = i+1, l-1 {
		resp[i], resp[l] = resp[l], resp[i]
	}

	return []byte(string(resp)), true, nil
}

var green = color.New(color.FgGreen).SprintFunc()

func PromptResponder(t int, msg []byte) (r []byte, ok bool, err error) {
	fmt.Print(green("> "))
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadSlice('\n')
	fmt.Printf("\033[F")

	return resp[:len(resp)-1], true, nil
}
