package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"goplayground/internal/biz/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Run() {
	r := gin.New()
	r.Use(Logger(), gin.Recovery())

	// Static files for the chat UI
	r.StaticFile("/", "./static/index.html")

	api := r.Group("/ai")
	{
		api.GET("/doubao", HandleDoubao)
		api.GET("/ws", HandleWebSocket)
		api.GET("/sse", HandleSSE)
	}

	fmt.Println("Server starting on :8080")
	r.Run(":8080")
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		latency := time.Since(startTime)
		log.Printf("[%s] %s %s %s %d %s", c.ClientIP(), c.Request.Method, c.Request.URL.Path, c.Request.Proto, c.Writer.Status(), latency)
	}
}

func HandleDoubao(c *gin.Context) {
	msg := c.Query("content")
	sessionId := c.Query("sessionId")
	if sessionId == "" {
		sessionId = "default"
	}

	dbao := service.NewDouBao(sessionId, c.Request.Context())
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
			req.AgentType = "doubao"
		}

		var reader *schema.StreamReader[*schema.Message]
		var dbao *service.DouBao

		switch req.AgentType {
		case "doubao":
			dbao = service.NewDouBao(req.SessionID, c.Request.Context())
			reader, err = dbao.ChatStream(c.Request.Context(), req.Content)
		default:
			conn.WriteJSON(gin.H{"type": "error", "content": "unknown agent type: " + req.AgentType})
			continue
		}

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
					"type": "stringevent",
					"event": "end",
					"sessionId": req.SessionID,
				})
				break
			}

			fullMsg += chunk.Content
			// Send chunk as "stringevent"
			conn.WriteJSON(gin.H{
				"type": "stringevent",
				"event": "message",
				"content": chunk.Content,
				"sessionId": req.SessionID,
			})
		}

		if dbao != nil {
			dbao.AddHistory(schema.AssistantMessage(fullMsg, nil))
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
		agentType = "doubao"
	}

	var reader *schema.StreamReader[*schema.Message]
	var dbao *service.DouBao
	var err error

	switch agentType {
	case "doubao":
		dbao = service.NewDouBao(sessionId, c.Request.Context())
		reader, err = dbao.ChatStream(c.Request.Context(), msg)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown agent type"})
		return
	}

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
			if dbao != nil {
				dbao.AddHistory(schema.AssistantMessage(fullMsg, nil))
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
