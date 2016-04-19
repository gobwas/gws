package events

import (
	"fmt"
	"testing"
)

func TestLoopCallback(t *testing.T) {
	loop := New()
	loop.AddListener(Header{EventCustom, "hello"}, NewHandler(func(d *DefaultHandler) {
		fmt.Println("EVENT")
		for i := 0; i < 1024; i++ {
			func(i int) {
				loop.NextTick(HandlerFunc(func() {
					fmt.Println("TICK", i)
				}))
			}(i)
		}
		fmt.Println("DEACTIVATED")
		d.Active = false
	}))
	loop.Trigger(Event{Header{EventCustom, "hello"}, "ok"})
	loop.Start()
	<-loop.Done()
}
