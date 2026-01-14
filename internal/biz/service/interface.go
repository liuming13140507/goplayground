package service

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

type Handler interface {
	Handle(ctx context.Context, c *gin.Context)
}

type Agent interface {
	Chat(ctx context.Context, msg string) (string, error)
	ChatStream(ctx context.Context, msg string) (*schema.StreamReader[*schema.Message], error)
	AddHistory(resp *schema.Message)
}
