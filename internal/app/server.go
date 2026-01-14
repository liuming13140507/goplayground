package app

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"goplayground/internal/biz/service"
)

func Run() {
	r := gin.New()
	r.Use(Logger(), gin.Recovery())

	// Static files for the chat UI
	r.StaticFile("/", "./static/index.html")

	api := r.Group("/ai")
	{
		api.GET("/doubao", service.HandleDoubao)
		api.GET("/ws", service.HandleWebSocket)
		api.GET("/sse", service.HandleSSE)
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
