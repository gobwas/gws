package server

import (
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"bufio"
	"os"
	"github.com/fatih/color"
	"fmt"
	"sync"
)

type wsHandler struct {
	respLock sync.Mutex
	upgrader  websocket.Upgrader
	headers   http.Header
	responder Responder
}

type Responder func(int, []byte) ([]byte, bool, error)

func newWsHandler(h http.Header, r Responder) *wsHandler {
	return &wsHandler{
		upgrader:  websocket.Upgrader{},
		headers:   h,
		responder: r,
	}
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// take lock on read new connection
	h.respLock.Lock()
	conn, err := h.upgrader.Upgrade(w, r, h.headers)
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

func Listen(addr string, h http.Header, r Responder) error {
	log.Println("ready to listen", addr)
	return http.ListenAndServe(addr, newWsHandler(h, r))
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
func PromptResponder(t int, msg[]byte) (r []byte, ok bool, err error) {
	fmt.Print(green("> "))
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadSlice('\n')
	fmt.Printf("\033[F")

	return resp[:len(resp)-1], true, nil
}
