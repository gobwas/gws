package ws

import (
	"crypto/tls"
	"flag"
	"github.com/gobwas/glob"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

var insecure = flag.Bool("insecure", false, "do not check tls certificate during dialing")
var keepalive = flag.Duration("keepalive", time.Minute*60, "how long to ws connection should be alive")

type Kind int

const HeaderOrigin = "Origin"

const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

func (m Kind) String() string {
	switch m {
	case TextMessage:
		return "TextMessage"
	case BinaryMessage:
		return "BinaryMessage"
	case CloseMessage:
		return "CloseMessage"
	case PingMessage:
		return "PingMessage"
	case PongMessage:
		return "PongMessage"
	default:
		return "UnknownMessage"
	}
}

type WriteRequest struct {
	Message MessageRaw
	Result  chan error
}

type ReceiveRequest struct {
	Result chan MessageAndError
}

type MessageRaw struct {
	Kind Kind
	Data []byte
}

type MessageAndError struct {
	Message MessageRaw
	Error   error
}

type Message struct {
	Kind Kind
	Data []byte
	Err  error
}

func WriteToConnFromChan(done <-chan struct{}, conn *websocket.Conn, output <-chan WriteRequest) {
	go func() {
		for {
			select {
			case <-done:
				return

			case req := <-output:
				err := WriteToConn(conn, req.Message.Kind, req.Message.Data)
				select {
				case <-done:
				case req.Result <- err:
				}
			}
		}
	}()
}

func ReadFromConnToChan(done <-chan struct{}, conn *websocket.Conn, ch <-chan ReceiveRequest) {
	go func() {
		for {
			select {
			case <-done:
				return

			case req := <-ch:
				m, err := ReadFromConn(conn)
				select {
				case <-done:
				case req.Result <- MessageAndError{m, err}:
				}
			}
		}
	}()
}

func WriteToConn(conn *websocket.Conn, t Kind, b []byte) error {
	writer, err := conn.NextWriter(int(t))
	if err != nil {
		return err
	}

	_, err = writer.Write(b)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	return nil
}

func ReadFromConn(conn *websocket.Conn) (msg MessageRaw, err error) {
	t, r, err := conn.NextReader()
	if err != nil {
		return
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}

	msg = MessageRaw{Data: b, Kind: Kind(t)}
	return
}

//todo refactor this to dedup
func ReadFromConnInto(done <-chan struct{}, conn *websocket.Conn, ch chan<- Message) {
	go func() {
		for {
			var msg Message
			t, r, err := conn.NextReader()
			if err != nil {
				msg = Message{Err: io.EOF}
			} else {
				b, err := ioutil.ReadAll(r)
				if err != nil {
					msg = Message{Err: err}
				} else {
					msg = Message{Data: b, Kind: Kind(t)}
				}
			}

			select {
			case <-done:
				return
			case ch <- msg:
				if msg.Err != nil {
					return
				}
			}
		}
	}()
}

func ReadAsyncFromConn(done <-chan struct{}, conn *websocket.Conn) <-chan Message {
	ch := make(chan Message)
	ReadFromConnInto(done, conn, ch)
	return ch
}

func GetConn(uri string, h http.Header) (conn *websocket.Conn, resp *http.Response, err error) {
	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			netDialer := &net.Dialer{
				KeepAlive: *keepalive,
			}
			return netDialer.Dial(network, addr)
		},
	}
	if *insecure {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	conn, resp, err = dialer.Dial(uri, h)
	return
}

type UpgradeConfig struct {
	Origin  string
	Headers http.Header
}

type Upgrader func(http.ResponseWriter, *http.Request) (*websocket.Conn, error)

func GetUpgrader(config UpgradeConfig) Upgrader {
	u := &websocket.Upgrader{}
	if config.Origin != "" {
		originChecker := glob.MustCompile(config.Origin)
		u.CheckOrigin = func(r *http.Request) bool {
			return originChecker.Match(r.Header.Get(HeaderOrigin))
		}
	}

	return func(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
		return u.Upgrade(w, r, config.Headers)
	}
}
