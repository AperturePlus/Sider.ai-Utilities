package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"

    "sider2api/pkg/types"
)

func (h *Handler) Health(c *gin.Context) {
    c.JSON(http.StatusOK, types.HealthResponse{
        Status:    "ok",
        Service:   "sider2api",
        Version:   "1.0.0-go",
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        TechStack: "gin + go",
    })
}

func (h *Handler) Root(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "name":        "Sider2API",
        "description": "Convert Sider AI API to Anthropic/OpenAI compatible API",
        "version":     "1.0.0-go",
        "tech_stack":  "gin + go",
        "endpoints": gin.H{
            "health":           "/health",
            "messages":         "/v1/messages",
            "count_tokens":     "/v1/messages/count_tokens",
            "chat_completions": "/v1/chat/completions",
            "playground":       "/ui",
        },
    })
}

func (h *Handler) Test(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Route structure is working", "timestamp": time.Now().Format(time.RFC3339)})
}
