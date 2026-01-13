package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var (
	ds sync.Map
)

type DouBao struct {
	sessionId string
	ctx       context.Context
	model     model.ChatModel
	history   []*schema.Message
}

func NewDouBao(sessionId string, ctx context.Context) *DouBao {
	if val, ok := ds.Load(sessionId); ok {
		return val.(*DouBao)
	}

	// Default to environment variables or reasonable defaults
	apiKey := os.Getenv("ARK_API_KEY")
	modelID := os.Getenv("ARK_MODEL_ID") // e.g., doubao-1.5-pro-32k-250115
	if modelID == "" {
		modelID = "doubao-1.5-pro-32k-250115"
	}

	timeout := 30 * time.Second
	m, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelID,
		Timeout: &timeout,
	})
	if err != nil {
		// In a real app, handle this better. For now, we'll log it.
		fmt.Printf("failed to create ark model: %v\n", err)
	}

	db := &DouBao{
		sessionId: sessionId,
		ctx:       ctx,
		model:     m,
		history:   make([]*schema.Message, 0),
	}
	ds.Store(sessionId, db)
	return db
}

func (d *DouBao) Chat(ctx context.Context, msg string) (string, error) {
	d.history = append(d.history, schema.UserMessage(msg))

	resp, err := d.model.Generate(ctx, d.history)
	if err != nil {
		return "", err
	}

	d.history = append(d.history, resp)
	return resp.Content, nil
}

func (d *DouBao) ChatStream(ctx context.Context, msg string) (*schema.StreamReader[*schema.Message], error) {
	d.history = append(d.history, schema.UserMessage(msg))

	reader, err := d.model.Stream(ctx, d.history)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (d *DouBao) AddHistory(resp *schema.Message) {
	d.history = append(d.history, resp)
}

func (d *DouBao) GetSessionId() string {
	return d.sessionId
}

