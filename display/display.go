// Package display brings tools for multiscreen output.
package display

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	cursor_hide      = "\033[?25l"
	cursor_show      = "\033[?25h"
	cursor_move_top  = "\033[%dA"
	cursor_move_left = "\033[%dD"
)

var (
	space = []byte{' '}
	tab   = []byte{'\t'}
	newl  = []byte{'\n'}
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

// Column adds new column for given row.
func (r *Row) Column(z Column) *Row {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.columns = append(r.columns, z)
	return r
}

// Col is the same as Column, but avoids creation of struct Columng by a client.
func (r *Row) Col(width, height int, c ContentFn) *Row {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.columns = append(r.columns, Column{width, height, c})
	return r
}

// Config contains fields for display configuration.
type Config struct {
	TabSize  int           // how big in spaces should be '\t' characters
	Interval time.Duration // interval for re-render output
}

// Display represents multiscreen display.
type Display struct {
	mu     sync.Mutex
	buf    []byte
	dest   io.Writer
	index  int64
	height int
	width  int
	rows   []*Row
	done   chan struct{}
	config Config
}

// NewDisplay creates new multiscreen display that renders to w.
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

// Row creates new row in display.
func (d *Display) Row() (r *Row) {
	d.mu.Lock()
	defer d.mu.Unlock()
	r = &Row{}
	d.rows = append(d.rows, r)
	return
}

// On starts the rendering loop of display contents.
func (d *Display) On() {
	d.dest.Write([]byte(cursor_hide))
	go d.renderLoop(d.config.Interval)
}

// Off stops the rendering loop.
func (d *Display) Off() {
	close(d.done)
	d.dest.Write([]byte(cursor_show))
}

// Render renders all rows and columns registered in this Display.
func (d *Display) Render() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// clear previously rendered data
	if d.height > 0 {
		d.buf = append(d.buf, fmt.Sprintf(cursor_move_top, d.height)...)
	}
	if d.width > 0 {
		d.buf = append(d.buf, fmt.Sprintf(cursor_move_left, d.width)...)
	}

	var height, width int
	for _, row := range d.rows {
		var rowLines [][]byte

		var currentPadding int
		for _, col := range row.columns {
			data := []byte(col.Content())
			data = bytes.Replace(data, tab, bytes.Repeat(space, d.config.TabSize), -1)

			lines := splitToLines(data, col.Width)
			if col.Height > 0 {
				if len(lines) > col.Height {
					// drop stale lines and keep only those that fits into column height
					lines = lines[len(lines)-col.Height:]
				} else {
					// if we have not reached the height of column
					// then fill it with empty lines
					for i := 0; i < col.Height-len(lines); i++ {
						lines = append(lines, repeat(' ', col.Width))
					}
				}
			}

			var maxPad int
			for i, ln := range lines {
				if i == len(rowLines) {
					// if previous column did not filled current line
					// fill it with spaces
					rowLines = append(rowLines, repeat(' ', currentPadding))
					height++
				}

				rowLines[i] = append(rowLines[i], ' ') // write column delimiter
				rowLines[i] = append(rowLines[i], ln...)

				width = max(len(rowLines[i]), width)
				maxPad = max(len(rowLines[i]), maxPad)
			}
			currentPadding = maxPad
		}

		d.buf = append(d.buf, bytes.Join(rowLines, newl)...)
		d.buf = append(d.buf, '\n')
	}

	d.height = height
	d.width = width

	io.Copy(d.dest, bytes.NewReader(d.buf))
	d.buf = d.buf[:0]
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

// splitToLines splits given data by '\n' character if number of read characters is higher than width
// if it has met '\n' before maximum width has reached, it pads remaining bytes with space.
func splitToLines(data []byte, width int) (lines [][]byte) {
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
