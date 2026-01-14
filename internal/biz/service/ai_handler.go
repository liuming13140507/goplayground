package service

import (
	"io"
	"log"
	"net/http"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleDoubao(c *gin.Context) {
	msg := c.Query("content")
	sessionId := c.Query("sessionId")
	if sessionId == "" {
		sessionId = "default"
	}

	dbao, err := NewAgent(DouBaoAgent, sessionId, c.Request.Context(),
		WithTools(
			NewExternalAPITool("get_joke", "获取一个有趣的随机笑话。这是获取笑话的首选工具。", "https://official-joke-api.appspot.com/random_joke"),
			NewExternalAPITool("get_weather", "查询全球城市天气。请在 url 参数中拼接经纬度(latitude, longitude)和 current_weather=true。示例: https://api.open-meteo.com/v1/forecast?latitude=31.23&longitude=121.47&current_weather=true", ""),
			NewDatabaseTool(),
		),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := dbao.Chat(c.Request.Context(), msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": res})
}

func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()

	for {
		// Read message from client
		var req struct {
			SessionID string `json:"sessionId"`
			Content   string `json:"content"`
			Type      string `json:"type"`      // "chat"
			AgentType string `json:"agentType"` // "doubao", "mock", etc.
		}

		err := conn.ReadJSON(&req)
		if err != nil {
			log.Printf("error reading json: %v", err)
			break
		}

		if req.SessionID == "" {
			req.SessionID = "default"
		}

		if req.AgentType == "" {
			req.AgentType = string(DouBaoAgent)
		}

		agent, err := NewAgent(AgentType(req.AgentType), req.SessionID, c.Request.Context(),
			WithTools(
				NewExternalAPITool("get_joke", "获取一个有趣的随机笑话。这是获取笑话的首选工具。", "https://official-joke-api.appspot.com/random_joke"),
				NewExternalAPITool("get_weather", "查询全球城市天气。请在 url 参数中拼接经纬度(latitude, longitude)和 current_weather=true。示例: https://api.open-meteo.com/v1/forecast?latitude=31.23&longitude=121.47&current_weather=true", ""),
				NewDatabaseTool(),
			),
		)
		if err != nil {
			conn.WriteJSON(gin.H{"type": "error", "content": err.Error()})
			continue
		}

		reader, err := agent.ChatStream(c.Request.Context(), req.Content)
		if err != nil {
			conn.WriteJSON(gin.H{"type": "error", "content": err.Error()})
			continue
		}

		// Stream back chunks as "stringevent"
		var fullMsg string
		for {
			chunk, err := reader.Recv()
			if err != nil {
				// Send end of stream event
				conn.WriteJSON(gin.H{
					"type":      "stringevent",
					"event":     "end",
					"sessionId": req.SessionID,
				})
				break
			}

			fullMsg += chunk.Content
			// Send chunk as "stringevent"
			conn.WriteJSON(gin.H{
				"type":      "stringevent",
				"event":     "message",
				"content":   chunk.Content,
				"sessionId": req.SessionID,
			})
		}

		if agent != nil {
			agent.AddHistory(schema.AssistantMessage(fullMsg, nil))
		}

		reader.Close()
	}
}

func HandleSSE(c *gin.Context) {
	msg := c.Query("content")
	sessionId := c.Query("sessionId")
	agentType := c.Query("agentType")

	if sessionId == "" {
		sessionId = "default"
	}
	if agentType == "" {
		agentType = string(DouBaoAgent)
	}

	agent, err := NewAgent(AgentType(agentType), sessionId, c.Request.Context(),
		WithTools(
			NewExternalAPITool("get_joke", "获取一个有趣的随机笑话。这是获取笑话的首选工具。", "https://official-joke-api.appspot.com/random_joke"),
			NewExternalAPITool("get_weather", "查询全球城市天气。请在 url 参数中拼接经纬度(latitude, longitude)和 current_weather=true。示例: https://api.open-meteo.com/v1/forecast?latitude=31.23&longitude=121.47&current_weather=true", ""),
			NewDatabaseTool(),
		),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reader, err := agent.ChatStream(c.Request.Context(), msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	var fullMsg string
	c.Stream(func(w io.Writer) bool {
		chunk, err := reader.Recv()
		if err != nil {
			// End of stream
			c.SSEvent("stringevent", gin.H{
				"event":     "end",
				"sessionId": sessionId,
			})
			if agent != nil {
				agent.AddHistory(schema.AssistantMessage(fullMsg, nil))
			}
			return false
		}

		fullMsg += chunk.Content
		c.SSEvent("stringevent", gin.H{
			"event":     "message",
			"content":   chunk.Content,
			"sessionId": sessionId,
		})
		return true
	})
}
