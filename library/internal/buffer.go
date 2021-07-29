package internal

import (
	"bytes"
	"io"
)

//MultiplexingBuffer is a type that will allow storing, replaying and streaming data from it's internal buffer
//It implements io.Writer storing the written bytes to an unbound buffer internally
//The Reader function will return an io.Reader that will stream data starting with the buffer, until the Close method of the underlying MultiplexingBuffer is called
type MultiplexingBuffer struct {
	closed bool
	buffer []byte
}

type streamingReader struct {
	multiplexingBuffer *MultiplexingBuffer
	idx                int
	drained            bool
}

func (s *streamingReader) Read(p []byte) (n int, err error) {

	toRead := len(p)

	if s.drained {
		return 0, io.EOF
	}

	for i := 0; i < toRead; i++ {
		if i+s.idx >= len(s.multiplexingBuffer.buffer) {
			if s.multiplexingBuffer.closed {
				s.drained = true
			}
			s.idx += i

			return i, nil
		}
		p[i] = s.multiplexingBuffer.buffer[s.idx+i]
	}

	s.idx += toRead
	return toRead, nil
}

func (m *MultiplexingBuffer) Reader() io.Reader {
	if m.closed {
		return bytes.NewReader(m.buffer)
	}

	return &streamingReader{idx: 0, multiplexingBuffer: m}
}

func (m *MultiplexingBuffer) Close() error {
	m.closed = true

	return nil
}

func (m *MultiplexingBuffer) Write(p []byte) (n int, err error) {
	m.buffer = append(m.buffer, p...)

	return len(p), nil
}
