package ws

import (
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
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
	Kind Kind
	Data []byte
}

type Message struct {
	Kind Kind
	Data []byte
	Err  error
}

func WriteToConnChan(conn *websocket.Conn, done <-chan struct{}, output <-chan MessageRaw, errors chan<- error) {
	go func() {
		select {
		case <-done:
			return

		case msg := <-output:
			err := WriteToConn(conn, msg.Kind, msg.Data)
			if err != nil {
				select {
				case <-done:
					return
				case errors <- err:
					return
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
