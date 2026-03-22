package shepherd

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type stubChatModel struct {
	generateResp *schema.Message
	generateErr  error
	called       bool // 记录 Generate 是否被调用
}

func (m *stubChatModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	m.called = true
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	if m.generateResp != nil {
		return m.generateResp, nil
	}
	return &schema.Message{}, nil
}

func (m *stubChatModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented in tests")
}

func (m *stubChatModel) BindTools(_ []*schema.ToolInfo) error {
	return nil
}
