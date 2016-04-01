package events

import "sync"

type EventKind int

const (
	Timer EventKind = iota
	Custom
)

type Header struct {
	Kind EventKind
	Name string
}

type Event struct {
	Header Header
	Data   map[string]interface{}
}

type TickFn func()
type ListenerFn func(Event)

type Loop struct {
	mu        sync.Mutex
	listeners map[Header][]ListenerFn
	ticks     chan TickFn
	events    chan Event
	done      chan struct{}
	error     error
}

func New() *Loop {
	return &Loop{}
}

func (loop *Loop) Start() error {
	loop.done = make(chan struct{})
	go func() {
		for loop.IsAlive() {
			var ticksAreDrained bool
			for !ticksAreDrained {
				select {
				case tick := <-loop.ticks:
					tick()
				default:
					ticksAreDrained = true
				}
			}

			select {
			case event := <-loop.events:
				loop.handleEvent(event)
			default:
				//
			}
		}

		close(loop.done)
	}()

	return nil
}

func (loop *Loop) Done() <-chan struct{} {
	return loop.done
}

func (loop *Loop) Error() error {
	return loop.error
}

func (loop *Loop) IsAlive() bool {
	dead := true
	dead = dead && len(loop.listeners) == 0
	dead = dead && len(loop.ticks) == 0
	return !dead
}

func (loop *Loop) Listen(header Header, listener ListenerFn) {
	loop.mu.Lock()
	{
		loop.listeners[header] = append(loop.listeners[header], listener)
	}
	loop.mu.Unlock()
}

func (loop *Loop) Trigger(evt Event) {
	loop.events <- evt
}

func (loop *Loop) handleEvent(evt Event) {
	loop.mu.Lock()
	{
		for _, listener := range loop.listeners[evt.Header] {
			listener(evt)
		}
	}
	loop.mu.Unlock()
}

func (loop *Loop) NextTick(fn TickFn) {
	loop.ticks <- fn
}
