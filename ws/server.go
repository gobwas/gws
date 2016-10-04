package ws

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"sync"
	"time"
)

type Handler interface {
	Handle(*websocket.Conn, error)
}

type HandlerFunc func(*websocket.Conn, error)

func (h HandlerFunc) Handle(c *websocket.Conn, e error) {
	h(c, e)
}

type ServerConfig struct {
	Addr    string
	Key     string
	Cert    string
	Origin  string
	Headers http.Header
}

type Server struct {
	mu        sync.Mutex
	listening bool
	config    ServerConfig
	deferreds []func()
	handlers  []Handler
	conns     chan conn
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		config: cfg,
		conns:  make(chan conn),
	}
}

func (s *Server) Handle(h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers = append(s.handlers, h)
	go func() {
		for w := range s.conns {
			h.Handle(w.conn, w.err)
		}
	}()
}

func (s *Server) Defer(dfd func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deferreds = append(s.deferreds, dfd)
}

func (s *Server) Listen(done chan struct{}) {
	go func() {
		upgrade := GetUpgrader(UpgradeConfig{
			Origin:  s.config.Origin,
			Headers: s.config.Headers,
		})

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := upgrade(w, r)
			s.conns <- conn{c, err}
		})

		var ln net.Listener
		var err error
		if s.config.Cert != "" || s.config.Key != "" {
			ln, err = getTLSListener(done, s.config.Addr, s.config.Cert, s.config.Key)
		} else {
			ln, err = getListener(done, s.config.Addr)
		}

		if err == nil {
			err = http.Serve(ln, handler)
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		for _, dfd := range s.deferreds {
			dfd()
		}
		close(s.conns)
		for _, h := range s.handlers {
			h.Handle(nil, err)
		}
	}()
}

type conn struct {
	conn *websocket.Conn
	err  error
}

type tcpStoppableListener struct {
	*net.TCPListener
	stop chan struct{}
}

func (ln *tcpStoppableListener) Accept() (c net.Conn, err error) {
	for {
		ln.SetDeadline(time.Now().Add(time.Second))
		select {
		case <-ln.stop:
			ln.Close()
			return nil, fmt.Errorf("listener has been stopped!")

		default:
			c, err = ln.AcceptTCP()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Temporary() && ne.Timeout() {
					continue
				}
			}
			return
		}
	}
}

func getListener(done chan struct{}, addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &tcpStoppableListener{ln.(*net.TCPListener), done}, nil
}

func getTLSListener(done chan struct{}, addr, cert, key string) (net.Listener, error) {
	config := &tls.Config{
		NextProtos:   []string{"http/1.1"},
		Certificates: make([]tls.Certificate, 1),
	}
	var err error
	config.Certificates[0], err = tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return tls.NewListener(&tcpStoppableListener{ln.(*net.TCPListener), done}, config), nil
}
