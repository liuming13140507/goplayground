package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
)

type AgentType string

const (
	DouBaoAgent AgentType = "doubao"
)

// AgentOptions holds the configuration for creating an agent.
type AgentOptions struct {
	ModelID string
	Timeout time.Duration
	Tools   []tool.InvokableTool
}

// Option is a functional option for configuring an Agent.
type Option func(*AgentOptions)

func WithModelID(modelID string) Option {
	return func(o *AgentOptions) {
		o.ModelID = modelID
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *AgentOptions) {
		o.Timeout = timeout
	}
}

func WithTools(tools ...tool.InvokableTool) Option {
	return func(o *AgentOptions) {
		o.Tools = append(o.Tools, tools...)
	}
}

func NewAgent(agentType AgentType, sessionId string, ctx context.Context, opts ...Option) (Agent, error) {
	options := &AgentOptions{
		Timeout: 30 * time.Second, // default timeout
	}
	for _, opt := range opts {
		opt(options)
	}

	switch agentType {
	case DouBaoAgent:
		return NewDouBao(sessionId, ctx, options)
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}
