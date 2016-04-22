package ws

import (
	"github.com/gorilla/websocket"
)

type Connection struct {
	conn   *websocket.Conn
	done   chan struct{}
	in     chan MessageRaw
	out    chan MessageRaw
	errors chan Error
}

func NewConnection(c *websocket.Conn) *Connection {
	return &Connection{
		conn:   c,
		done:   make(chan struct{}),
		in:     make(chan MessageRaw),
		out:    make(chan MessageRaw),
		errors: make(chan Error),
	}
}

func (c *Connection) Init() {
	ReadFromConnToChan(c.done, c.conn, c.in, c.errors)
	WriteToConnFromChan(c.done, c.conn, c.out, c.errors)
}

func (c *Connection) Close() error {
	close(c.done)
	return c.conn.Close()
}
