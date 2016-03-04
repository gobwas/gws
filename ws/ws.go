package ws

import (
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
)

type MsgType int

const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

func (m MsgType) String() string {
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

func writeMessageNonBlocking(ch chan<- Message, m Message) {
	select {
	case ch <- m:
	//
	default:
		//
	}
}

type MessageRaw struct {
	Kind MsgType
	Data []byte
}

type Message struct {
	Kind MsgType
	Data []byte
	Err  error
}

func WriteToConnChan(conn *websocket.Conn, done <-chan struct{}, output <-chan MessageRaw) <-chan error {
	ch := make(chan error)

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
				default:
					ch <- err
				}
				return
			}
		}
	}()

	return ch
}

func WriteToConn(conn *websocket.Conn, t MsgType, b []byte) error {
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

func ReadFromConn(conn *websocket.Conn, done <-chan struct{}) <-chan Message {
	ch := make(chan Message)

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
					msg = Message{Data: b, Kind: MsgType(t)}
				}
			}

			select {
			case <-done:
				return
			default:
				ch <- msg
				if msg.Err != nil {
					return
				}
			}
		}
	}()

	return ch
}
