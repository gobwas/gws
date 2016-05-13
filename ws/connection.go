package ws

import (
	"errors"
	"github.com/gorilla/websocket"
	"sync"
)

type Connection struct {
	once sync.Once

	conn    *websocket.Conn
	done    chan struct{}
	in      chan ReceiveRequest
	out     chan WriteRequest
	running bool
}

func NewConnection(c *websocket.Conn) *Connection {
	return &Connection{
		conn: c,
		done: make(chan struct{}),
		in:   make(chan ReceiveRequest),
		out:  make(chan WriteRequest),
	}
}

func (c *Connection) SendAsync(msg MessageRaw) chan error {
	resp := make(chan error)
	c.out <- WriteRequest{msg, resp}
	return resp
}

func (c *Connection) Send(msg MessageRaw) error {
	return <-c.SendAsync(msg)
}

func (c *Connection) ReceiveAsync() chan MessageAndError {
	resp := make(chan MessageAndError)
	c.in <- ReceiveRequest{resp}
	return resp
}

func (c *Connection) Receive() (MessageRaw, error) {
	result := <-c.ReceiveAsync()
	return result.Message, result.Error
}

func (c *Connection) Done() <-chan struct{} {
	return c.done
}

func (c *Connection) IsRunning() bool {
	return c.running
}

func (c *Connection) InitIOWorkers() {
	c.once.Do(func() {
		WriteToConnFromChan(c.done, c.conn, c.out)
		ReadFromConnToChan(c.done, c.conn, c.in)
		c.running = true
	})
}

func (c *Connection) Close() error {
	select {
	case <-c.done:
		return errors.New("already closed")
	default:
		close(c.done)
		close(c.in)
		close(c.out)
	}

	return c.conn.Close()
}
