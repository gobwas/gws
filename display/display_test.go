package display_test

import (
	"fmt"
	"github.com/gobwas/gws/display"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDisplay(t *testing.T) {
	d := display.NewDisplay(os.Stderr, display.Config{
		TabSize:  4,
		Interval: time.Millisecond * 500,
	})
	r1 := d.Row()
	r1.Column(display.Column{
		Width:  20,
		Height: 2,
		Content: func() string {
			return fmt.Sprintf("left col: %s", strings.Repeat("*", rand.Intn(10)))
		},
	})
	r1.Column(display.Column{
		Width:  20,
		Height: 2,
		Content: func() string {
			return fmt.Sprintf("right col: %s", strings.Repeat("*", rand.Intn(10)))
		},
	})

	r2 := d.Row()
	r2.Column(display.Column{
		Height: 2,
		Width:  100,
		Content: func() string {
			return fmt.Sprintf("long middle single col: %s", strings.Repeat("*", rand.Intn(10)))
		},
	})

	r3 := d.Row()
	var multilineA []string
	r3.Column(display.Column{
		Width:  20,
		Height: 4,
		Content: func() string {
			multilineA = append(multilineA, fmt.Sprintf("left counter: %d", rand.Intn(100)))
			return strings.Join(multilineA, "\n")
		},
	})
	var multilineB []string
	r3.Column(display.Column{
		Width:  20,
		Height: 10,
		Content: func() string {
			multilineB = append(multilineB, fmt.Sprintf("right counter: %d", rand.Intn(100)))
			return strings.Join(multilineB, "\n")
		},
	})
	d.Begin()
	time.Sleep(time.Minute)
}
