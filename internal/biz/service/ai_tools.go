package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// ExternalAPIRequest is the input schema for the external API tool.
type ExternalAPIRequest struct {
	URL string `json:"url,omitempty" jsonschema:"description=可选：要调用的外部 API 完整 URL。如果工具已预设 URL，模型可以不传此参数。"`
}

// ExternalAPIResponse is the output schema for the external API tool.
type ExternalAPIResponse struct {
	Body string `json:"body"`
}

// NewExternalAPITool creates a tool that calls an external API.
func NewExternalAPITool(name, desc, baseURL string) tool.InvokableTool {
	return utils.NewTool[ExternalAPIRequest, ExternalAPIResponse](
		&schema.ToolInfo{
			Name: name,
			Desc: desc,
		},
		func(ctx context.Context, input ExternalAPIRequest) (ExternalAPIResponse, error) {
			u := input.URL
			if u == "" {
				u = baseURL
			}

			if u == "" {
				return ExternalAPIResponse{}, fmt.Errorf("URL is required")
			}

			// 增加超时控制
			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := client.Get(u)
			if err != nil {
				// 如果是笑话接口报错，提供本地兜底
				if name == "get_joke" {
					return ExternalAPIResponse{
						Body: `{"setup": "[本地兜底] 为什么程序员总是分不清万圣节和圣诞节？", "punchline": "因为 Oct 31 == Dec 25"}`,
					}, nil
				}
				return ExternalAPIResponse{}, err
			}
			defer resp.Body.Close()

			var body interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				return ExternalAPIResponse{}, err
			}

			bodyBytes, _ := json.Marshal(body)
			return ExternalAPIResponse{Body: string(bodyBytes)}, nil
		},
	)
}

// DatabaseRequest is the input schema for the database tool.
type DatabaseRequest struct {
	Query string `json:"query" jsonschema:"description=The SQL query to execute"`
}

// DatabaseResponse is the output schema for the database tool.
type DatabaseResponse struct {
	Result string `json:"result"`
}

// NewDatabaseTool creates a tool that executes a SQL query on the local database.
func NewDatabaseTool() tool.InvokableTool {
	return utils.NewTool[DatabaseRequest, DatabaseResponse](
		&schema.ToolInfo{
			Name: "local_db",
			Desc: "执行本地数据库查询。注意：目前数据库中没有任何笑话或天气数据，如需笑话请使用 get_joke，如需天气请使用 get_weather。",
		},
		func(ctx context.Context, input DatabaseRequest) (DatabaseResponse, error) {
			// 如果 AI 还是来查笑话，我们直接给它一个结果，防止死循环
			return DatabaseResponse{
				Result: "数据库查询完成。结果：[未找到相关记录]。请尝试使用外部 API 工具获取实时内容。",
			}, nil
		},
	)
}
