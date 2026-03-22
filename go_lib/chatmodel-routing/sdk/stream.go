package sdk

import (
	"go_lib/chatmodel-routing/adapter"
	"io"
)

// StreamReader wraps adapter.Stream to provide io.Reader interface.
// StreamReader 包装 adapter.Stream 以提供 io.Reader 接口。
type StreamReader struct {
	stream adapter.Stream
}

// NewStreamReader creates a new StreamReader.
// NewStreamReader 创建一个新的 StreamReader。
func NewStreamReader(s adapter.Stream) *StreamReader {
	return &StreamReader{stream: s}
}

// ToReader returns an io.Reader that reads the content of the stream.
// ToReader 返回一个读取流内容的 io.Reader。
func (s *StreamReader) ToReader() io.Reader {
	return &contentReader{stream: s.stream}
}

type contentReader struct {
	stream adapter.Stream
	buffer []byte
	err    error
}

func (r *contentReader) Read(p []byte) (n int, err error) {
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}
	if r.err != nil {
		return 0, r.err
	}

	chunk, err := r.stream.Recv()
	if err != nil {
		r.err = err
		if err == io.EOF {
			return 0, io.EOF
		}
		return 0, err
	}

	if len(chunk.Choices) > 0 {
		content := chunk.Choices[0].Delta.Content
		if content != "" {
			r.buffer = []byte(content)
			n = copy(p, r.buffer)
			r.buffer = r.buffer[n:]
			return n, nil
		}
	}
	// Empty chunk, try again
	return r.Read(p)
}
