package display

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

type ContentFn func() string

type Column struct {
	Width   int
	Height  int
	Content ContentFn
}

type Row struct {
	mu      sync.Mutex
	columns []Column
}

func (r *Row) Column(z Column) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.columns = append(r.columns, z)
}

func (r *Row) Col(width, height int, c ContentFn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.columns = append(r.columns, Column{width, height, c})
}

type Config struct {
	TabSize  int
	Interval time.Duration
}

type Display struct {
	mu     sync.Mutex
	dest   io.Writer
	index  int64
	height int
	width  int
	rows   []*Row
	done   chan struct{}
	config Config
}

func NewDisplay(w io.Writer, c Config) *Display {
	if c.TabSize == 0 {
		c.TabSize = 4
	}
	if c.Interval == 0 {
		c.Interval = time.Millisecond * 100
	}
	return &Display{
		dest:   w,
		done:   make(chan struct{}),
		config: c,
	}
}

func (d *Display) Row() (r *Row) {
	d.mu.Lock()
	defer d.mu.Unlock()
	r = &Row{}
	d.rows = append(d.rows, r)
	return
}

func (d *Display) Begin() {
	go d.renderLoop(d.config.Interval)
}

func (d *Display) Stop() {
	close(d.done)
}

var buffers = sync.Pool{}

func (d *Display) Render() {
	d.mu.Lock()
	defer d.mu.Unlock()

	var buf []byte
	if b := buffers.Get(); b != nil {
		buf = b.([]byte)
	}

	if d.height > 0 {
		buf = append(buf, fmt.Sprintf("\033[%dA", d.height)...)
	}
	if d.width > 0 {
		buf = append(buf, fmt.Sprintf("\033[%dD", d.width)...)
	}

	var height, width int
	for _, row := range d.rows {
		var rowLines [][]byte

		var currentPadding int
		for _, zone := range row.columns {
			data := []byte(zone.Content())
			data = bytes.Replace(data, []byte{'\t'}, bytes.Repeat([]byte{' '}, d.config.TabSize), -1)

			lines := fitLinesWidth(data, zone.Width)
			if zone.Height > 0 {
				if len(lines) > zone.Height {
					lines = lines[len(lines)-zone.Height:]
				} else {
					for i := 0; i < zone.Height-len(lines); i++ {
						lines = append(lines, repeat(' ', zone.Width))
					}
				}
			}

			var maxPad int
			for i, ln := range lines {
				if i == len(rowLines) {
					rowLines = append(rowLines, repeat(' ', currentPadding))
				}

				rowLines[i] = append(rowLines[i], ' ')
				rowLines[i] = append(rowLines[i], ln...)

				width = max(len(rowLines[i]), width)
				maxPad = max(len(rowLines[i]), maxPad)
			}
			currentPadding = maxPad
		}

		buf = append(buf, bytes.Join(rowLines, []byte{'\n'})...)
		buf = append(buf, '\n')
		height += len(rowLines)
	}

	d.height = height
	d.width = width
	io.Copy(d.dest, bytes.NewReader(buf))

	buffers.Put(buf[:0])
}

func (d *Display) renderLoop(duration time.Duration) {
	ticker := time.Tick(duration)
	for {
		select {
		case <-ticker:
			d.Render()
		case <-d.done:
			return
		}
	}
}

func fitLinesWidth(data []byte, width int) (lines [][]byte) {
	var pad int
	for i, w := 0, 0; i < len(data); i, w = i+1, w+1 {
		lineBreak := data[i] == '\n'
		switch {
		case lineBreak || w == width:
			source := data[pad:i]
			lines = append(lines, ensureWidth(source, width, ' '))
			w = 0

			pad += len(source)
			if lineBreak {
				pad += 1
			}
		}
	}
	lines = append(lines, ensureWidth(data[pad:], width, ' '))
	return
}

func ensureWidth(b []byte, w int, pad byte) []byte {
	if len(b) < w {
		s := make([]byte, len(b))
		copy(s, b)
		return append(s, repeat(pad, w-len(b))...)
	}
	return b
}

func repeat(b byte, n int) []byte {
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		ret[i] = b
	}
	return ret
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
