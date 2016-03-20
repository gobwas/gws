package client

import (
	"github.com/yuin/gopher-lua"
	"sync"
	"time"
)

func ExportThread(t *Thread, L *lua.LState) *lua.LTable {
	thread := L.NewTable()
	thread.RawSetString("set", L.NewClosure(func(L *lua.LState) int {
		key := L.ToString(2)
		value := L.Get(3)
		t.Set(key, value)
		return 0
	}))
	thread.RawSetString("get", L.NewClosure(func(L *lua.LState) int {
		key := L.ToString(2)
		value := t.Get(key)
		switch v := value.(type) {
		case lua.LValue:
			L.Push(v)
		case string:
			L.Push(lua.LString(v))
		case int:
			L.Push(lua.LNumber(v))
		default:
			L.Push(lua.LNil)
		}
		return 1
	}))
	thread.RawSetString("send", L.NewClosure(func(L *lua.LState) int {
		msg := L.ToString(2)
		err := t.Send([]byte(msg))
		if err != nil {
			L.Push(lua.LString(err.Error()))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))
	thread.RawSetString("receive", L.NewClosure(func(L *lua.LState) int {
		msg, err := t.Receive()
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
		} else {
			L.Push(lua.LString(msg))
			L.Push(lua.LNil)
		}
		return 2
	}))
	thread.RawSetString("sleep", L.NewClosure(func(L *lua.LState) int {
		arg := L.ToString(2)
		duration, err := time.ParseDuration(arg)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		//time.Sleep(duration)
		t.Sleep(duration)
		return 0
	}))
	thread.RawSetString("close", L.NewClosure(func(L *lua.LState) int {
		err := t.Close()
		if err != nil {
			L.Push(lua.LString(err.Error()))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))
	thread.RawSetString("kill", L.NewClosure(func(L *lua.LState) int {
		t.Kill()
		return 0
	}))
	return thread
}

type Conn interface {
	Send([]byte) error
	Receive() ([]byte, error)
	Close() error
}

type Thread struct {
	mu    sync.Mutex
	conn  Conn
	data  map[string]interface{}
	sleep chan time.Duration
	Awake chan empty
	Dead  chan struct{}
}

func NewThread() *Thread {
	t := &Thread{
		data:  make(map[string]interface{}),
		Dead:  make(chan struct{}),
		Awake: make(chan empty),
		sleep: make(chan time.Duration, 1),
	}

	t.InitEventLoop()

	return t
}

func (t *Thread) InitEventLoop() {
	go func() {
		timer := time.NewTimer(0)
		timer.Stop()

		for {
			select {
			// received signal that we should sleep for a while
			case duration := <-t.sleep:
				// flush awake chan preventing any ticks run
				select {
				case <-t.Awake:
				default:
				}

				timer.Reset(duration)
				<-timer.C

			// other way send signal that we could continue event loop
			case t.Awake <- empty{}:
				// do nothing
			}
		}
	}()
}

func (t *Thread) SetConn(c Conn) {
	t.conn = c
}

func (t *Thread) HasConn() bool {
	return t.conn != nil
}

func (t *Thread) Set(key string, value interface{}) {
	t.data[key] = value
	return
}

func (t *Thread) Get(key string) (result interface{}) {
	result = t.data[key]
	return
}

func (t *Thread) Send(d []byte) (err error) {
	return t.conn.Send(d)
}

func (t *Thread) Receive() ([]byte, error) {
	return t.conn.Receive()
}

func (t *Thread) Close() error {
	return t.conn.Close()
}

func (t *Thread) Kill() {
	close(t.Dead)
}

func (t *Thread) Sleep(duration time.Duration) {
	t.sleep <- duration
}

func (t *Thread) NextTick() bool {
	select {
	case <-t.Dead:
		return false
	case <-t.Awake:
		return true
	}
}

type empty struct{}
