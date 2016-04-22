package ws

import (
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net/http"
)

type Kind int

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

type MessageRaw struct {
	Seq  int32
	Kind Kind
	Data []byte
}

type Message struct {
	Kind Kind
	Data []byte
	Err  error
}

type Error struct {
	Seq int32
	Err error
}

func WriteToConnFromChan(done <-chan struct{}, conn *websocket.Conn, output <-chan MessageRaw, errors chan<- Error) {
	go func() {
		select {
		case <-done:
			return

		case msg := <-output:
			err := WriteToConn(conn, msg.Kind, msg.Data)
			if err != nil {
				select {
				case <-done:
				case errors <- Error{msg.Seq, err}:
				}
				return
			}
		}
	}()
}

func ReadFromConnToChan(done <-chan struct{}, conn *websocket.Conn, ch chan<- MessageRaw, errors chan<- Error) {
	go func() {
		for {
			var msg MessageRaw
			var err error
			t, r, e := conn.NextReader()
			if e != nil {
				err = io.EOF
			} else {
				b, e := ioutil.ReadAll(r)
				if e != nil {
					err = e
				} else {
					msg = MessageRaw{Data: b, Kind: Kind(t)}
				}
			}

			if err != nil {
				select {
				case <-done:
				case errors <- Error{Seq: -1, Err: err}:
				}
				return
			}

			select {
			case <-done:
				return
			case ch <- msg:
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
	dialer := &websocket.Dialer{}
	conn, resp, err = dialer.Dial(uri, h)
	return
}
