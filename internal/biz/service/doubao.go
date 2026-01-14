package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	ds sync.Map
)

var _ Agent = (*DouBao)(nil)

type DouBao struct {
	sessionId string
	ctx       context.Context
	model     model.ChatModel
	history   []*schema.Message
	tools     map[string]tool.InvokableTool
}

func NewDouBao(sessionId string, ctx context.Context, opts *AgentOptions) (*DouBao, error) {
	if val, ok := ds.Load(sessionId); ok {
		db := val.(*DouBao)
		// Update tools if provided
		if len(opts.Tools) > 0 {
			log.Printf("[NewDouBao] Updating tools for session: %s", sessionId)
			tools := make(map[string]tool.InvokableTool)
			toolInfos := make([]*schema.ToolInfo, 0, len(opts.Tools))
			for _, t := range opts.Tools {
				info, err := t.Info(ctx)
				if err != nil {
					continue
				}
				toolInfos = append(toolInfos, info)
				tools[info.Name] = t
			}
			db.tools = tools
			// Re-bind tools to the model if possible
			if bindable, ok := db.model.(interface {
				BindTools([]*schema.ToolInfo) error
			}); ok {
				bindable.BindTools(toolInfos)
			}
		}
		return db, nil
	}

	apiKey := os.Getenv("ARK_API_KEY")
	modelID := opts.ModelID
	if modelID == "" {
		modelID = os.Getenv("ARK_MODEL_ID")
	}
	if modelID == "" {
		modelID = "doubao-seed-1-6-251015"
	}

	timeout := opts.Timeout
	m, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelID,
		Timeout: &timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ark model: %v", err)
	}

	tools := make(map[string]tool.InvokableTool)
	toolInfos := make([]*schema.ToolInfo, 0, len(opts.Tools))
	if len(opts.Tools) > 0 {
		for _, t := range opts.Tools {
			info, err := t.Info(ctx)
			if err != nil {
				continue
			}
			toolInfos = append(toolInfos, info)
			tools[info.Name] = t
		}
		err = m.BindTools(toolInfos)
		if err != nil {
			return nil, fmt.Errorf("failed to bind tools: %v", err)
		}
	}

	db := &DouBao{
		sessionId: sessionId,
		ctx:       ctx,
		model:     m,
		history:   make([]*schema.Message, 0),
		tools:     tools,
	}
	ds.Store(sessionId, db)
	return db, nil
}

func (d *DouBao) Chat(ctx context.Context, msg string) (string, error) {
	log.Printf("[Chat] received: %s", msg)
	d.history = append(d.history, schema.UserMessage(msg))

	for {
		resp, err := d.model.Generate(ctx, d.history)
		if err != nil {
			log.Printf("[Chat] Generate error: %v", err)
			return "", err
		}
		d.history = append(d.history, resp)

		log.Printf("[Chat] model response: content=%s, tool_calls=%d", resp.Content, len(resp.ToolCalls))

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		for _, tc := range resp.ToolCalls {
			log.Printf("[Chat] calling tool: %s, args: %s", tc.Function.Name, tc.Function.Arguments)
			t, ok := d.tools[tc.Function.Name]
			if !ok {
				log.Printf("[Chat] tool not found: %s", tc.Function.Name)
				continue
			}
			args := tc.Function.Arguments
			if args == "" {
				args = "{}"
			}
			res, err := t.InvokableRun(ctx, args)
			if err != nil {
				log.Printf("[Chat] tool run error: %v", err)
				d.history = append(d.history, schema.ToolMessage(fmt.Sprintf("error: %v", err), tc.ID))
				continue
			}
			log.Printf("[Chat] tool result: %s", res)
			d.history = append(d.history, schema.ToolMessage(res, tc.ID))
		}
	}
}

func (d *DouBao) ChatStream(ctx context.Context, msg string) (*schema.StreamReader[*schema.Message], error) {
	log.Printf("[ChatStream] received: %s", msg)
	d.history = append(d.history, schema.UserMessage(msg))
	return d.chatStreamInternal(ctx)
}

func (d *DouBao) chatStreamInternal(ctx context.Context) (*schema.StreamReader[*schema.Message], error) {
	log.Printf("[ChatStream] calling stream with history len: %d", len(d.history))
	reader, err := d.model.Stream(ctx, d.history)
	if err != nil {
		log.Printf("[ChatStream] Stream error: %v", err)
		return nil, err
	}

	var peekedMessages []*schema.Message
	var firstMeaningfulMsg *schema.Message

	// 持续读取直到发现内容或工具调用
	for {
		msg, err := reader.Recv()
		if err != nil {
			log.Printf("[ChatStream] Recv error or end during peek: %v", err)
			reader.Close()
			if len(peekedMessages) > 0 {
				return schema.StreamReaderFromArray(peekedMessages), nil
			}
			return nil, err
		}
		peekedMessages = append(peekedMessages, msg)
		if msg.Content != "" || len(msg.ToolCalls) > 0 {
			firstMeaningfulMsg = msg
			break
		}
	}

	log.Printf("[ChatStream] meaningful msg found: content=%s, tool_calls=%d", firstMeaningfulMsg.Content, len(firstMeaningfulMsg.ToolCalls))

	// 情况 1：是普通文本回复
	if len(firstMeaningfulMsg.ToolCalls) == 0 {
		// 此时 peekedMessages 包含了所有之前的空块和第一个有内容的块
		// 我们将已读到的块和剩余的 reader 合并返回给前端
		d.history = append(d.history, firstMeaningfulMsg)
		return schema.MergeStreamReaders([]*schema.StreamReader[*schema.Message]{
			schema.StreamReaderFromArray(peekedMessages),
			reader,
		}), nil
	}

	// 情况 2：是工具调用
	// 继续消费剩余流以聚合完整的工具参数
	fullMsg := firstMeaningfulMsg
	// 注意：我们需要处理 peekedMessages 中可能已经存在的 tool call 碎片（虽然通常第一个有意义的块就够了）
	for {
		chunk, err := reader.Recv()
		if err != nil {
			break
		}
		// 聚合 ToolCalls 参数
		for i := range chunk.ToolCalls {
			if i < len(fullMsg.ToolCalls) {
				fullMsg.ToolCalls[i].Function.Arguments += chunk.ToolCalls[i].Function.Arguments
			} else {
				fullMsg.ToolCalls = append(fullMsg.ToolCalls, chunk.ToolCalls[i])
			}
		}
	}
	reader.Close()
	d.history = append(d.history, fullMsg)

	// 执行工具逻辑
	for _, tc := range fullMsg.ToolCalls {
		log.Printf("[ChatStream] calling tool: %s, args: %s", tc.Function.Name, tc.Function.Arguments)
		t, ok := d.tools[tc.Function.Name]
		if !ok {
			log.Printf("[ChatStream] tool not found: %s", tc.Function.Name)
			continue
		}
		args := tc.Function.Arguments
		if args == "" {
			args = "{}"
		}
		res, err := t.InvokableRun(ctx, args)
		if err != nil {
			log.Printf("[ChatStream] tool run error: %v", err)
			d.history = append(d.history, schema.ToolMessage(fmt.Sprintf("工具执行失败: %v。请不要重试该工具，请直接告知用户该功能暂时不可用，并尝试用你已有的知识回答或表示歉意。", err), tc.ID))
			continue
		}
		log.Printf("[ChatStream] tool result: %s", res)
		d.history = append(d.history, schema.ToolMessage(res, tc.ID))
	}

	// 递归调用，直到 AI 给出最终的文本回答
	return d.chatStreamInternal(ctx)
}

func (d *DouBao) AddHistory(resp *schema.Message) {
	// 目前在 Chat/ChatStream 内部维护历史
}

func (d *DouBao) GetSessionId() string {
	return d.sessionId
}
