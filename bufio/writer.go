// Package bufio brings tools for io.
// It extends standard bufio package with prefixed writer and ring buffer.
package bufio

import "io"

// PrefixWriter adds prefix to every write call.
type PrefixWriter struct {
	dest   io.Writer
	prefix string
}

func NewPrefixWriter(dest io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{dest, prefix}
}

// Write writes p into underlying writer with prefix.
func (w PrefixWriter) Write(p []byte) (int, error) {
	ret := make([]byte, len(p)+len(w.prefix))
	ret = append(ret, w.prefix...)
	ret = append(ret, p...)
	return w.dest.Write(ret)
}

// RingBufferWriter implements ring buffer for io operations.
type RingBufferWriter struct {
	ring *ring
	dest io.Writer
}

func NewWriter(dest io.Writer, size int) *RingBufferWriter {
	return &RingBufferWriter{
		dest: dest,
		ring: newRing(size),
	}
}

// Write writes p to the underlying circular buffer.
// It returns len(p) and optional error.
func (w *RingBufferWriter) Write(p []byte) (int, error) {
	w.ring.append(p...)
	return len(p), nil
}

// Flush dumps contents of circular buffer into the underlying io.Writer.
// Flush is the same as Dump, except the thing, that flush drops buffer contents.
func (w *RingBufferWriter) Flush() (err error) {
	_, err = w.dest.Write(w.ring.flush())
	return
}

// Dump dumps contents of circular buffer into the underlying io.Writer.
// Dump do not drops buffer contents.
func (w *RingBufferWriter) Dump() (err error) {
	_, err = w.dest.Write(w.ring.dump())
	return
}

type ring struct {
	data []byte
	size int
	pos  int
	len  int
}

func newRing(size int) *ring {
	return &ring{
		size: size,
		data: make([]byte, size),
	}
}

func (r *ring) append(b ...byte) {
	var start int
	if len(b) > r.size {
		// get bytes that could be stored
		// from the end of b
		start = len(b) - r.size
	}

	var end int
	if len(b) < r.size-r.pos {
		end = len(b)
	} else {
		end = r.size - r.pos
	}

	rc := copy(r.data[r.pos:], b[start:end])
	lc := copy(r.data[:r.pos], b[start+end:])

	r.pos = (r.pos + rc + lc) % r.size
	if r.len+rc+lc > r.size {
		r.len = r.size
	} else {
		r.len += rc + lc
	}
}

func (r *ring) flush() (ret []byte) {
	ret = r.dump()
	r.len = 0
	r.pos = 0
	return
}

func (r *ring) dump() (ret []byte) {
	ret = make([]byte, r.len)
	var start int
	if r.len == r.size {
		copy(ret, r.data[r.pos:])
		start = r.size - r.pos
	}
	copy(ret[start:], r.data[:r.pos])
	return
}
