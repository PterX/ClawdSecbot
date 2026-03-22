package sdk

import (
	"bytes"
	"io"
	"testing"

	"go_lib/chatmodel-routing/adapter"

	"github.com/openai/openai-go"
)

type mockStream struct {
	chunks []*openai.ChatCompletionChunk
	index  int
}

var _ adapter.Stream = (*mockStream)(nil)

func (m *mockStream) Recv() (*openai.ChatCompletionChunk, error) {
	if m.index >= len(m.chunks) {
		return nil, io.EOF
	}
	chunk := m.chunks[m.index]
	m.index++
	return chunk, nil
}

func (m *mockStream) Close() error {
	return nil
}

func TestStreamReader(t *testing.T) {
	chunks := []*openai.ChatCompletionChunk{
		{
			Choices: []openai.ChatCompletionChunkChoice{
				{Delta: openai.ChatCompletionChunkChoiceDelta{Content: "Hello"}},
			},
		},
		{
			Choices: []openai.ChatCompletionChunkChoice{
				{Delta: openai.ChatCompletionChunkChoiceDelta{Content: " World"}},
			},
		},
	}

	stream := &mockStream{chunks: chunks}
	reader := NewStreamReader(stream).ToReader()

	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, reader)
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}

	if buf.String() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", buf.String())
	}
}
