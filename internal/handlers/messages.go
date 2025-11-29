package handlers

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    "sider2api/internal/converter"
    "sider2api/pkg/types"
)

// PostMessages handles /v1/messages
func (h *Handler) PostMessages(c *gin.Context) {
    authToken, ok := c.Get("authToken")
    if !ok {
        c.JSON(http.StatusUnauthorized, types.AnthropicError{Type: "error", Error: types.AnthropicErrorDetails{Type: "authentication_error", Message: "Authentication required"}})
        return
    }
    tokenStr, _ := authToken.(string)

    var req types.AnthropicRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, types.AnthropicError{Type: "error", Error: types.AnthropicErrorDetails{Type: "invalid_request_error", Message: err.Error()}})
        return
    }

    conversationID := c.Query("cid")
    if conversationID == "" {
        conversationID = c.GetHeader("X-Conversation-ID")
    }

    if conversationID == "" && hasAssistantHistory(req.Messages) {
        conversationID = h.Config.ContinuousCID
    }

    parentMessageID := c.GetHeader("X-Parent-Message-ID")
    if parentMessageID == "" && conversationID != "" {
        parentMessageID = h.Sessions.NextParentMessageID(conversationID)
    }

    siderReq, err := converter.ConvertAnthropicToSider(req, converter.ConvertOptions{
        ConversationID:  conversationID,
        ParentMessageID: parentMessageID,
        ContinuousCID:   h.Config.ContinuousCID,
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, types.AnthropicError{Type: "error", Error: types.AnthropicErrorDetails{Type: "invalid_request_error", Message: err.Error()}})
        return
    }

    ctx := c.Request.Context()
    siderResp, err := h.Client.Chat(ctx, siderReq, tokenStr)
    if err != nil {
        c.JSON(http.StatusInternalServerError, types.AnthropicError{Type: "error", Error: types.AnthropicErrorDetails{Type: "api_error", Message: err.Error()}})
        return
    }

    anthResp := converter.ConvertSiderToAnthropic(siderResp, req.Model)
    headers := converter.SessionHeadersFromSider(siderResp)

    if req.Stream {
        h.writeAnthropicStream(c, anthResp, headers)
        return
    }

    for k, v := range headers {
        c.Header(k, v)
    }
    c.JSON(http.StatusOK, anthResp)
}

// CountTokens handles /v1/messages/count_tokens (rough estimate)
func (h *Handler) CountTokens(c *gin.Context) {
    var body map[string]any
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
        return
    }
    b, _ := json.Marshal(body)
    est := len(b) / 4
    c.JSON(http.StatusOK, gin.H{"input_tokens": est})
}

func hasAssistantHistory(messages []types.AnthropicMessage) bool {
    for _, m := range messages {
        if m.Role == "assistant" {
            return true
        }
    }
    return false
}

// writeAnthropicStream emits SSE following Anthropic-like structure.
func (h *Handler) writeAnthropicStream(c *gin.Context, resp types.AnthropicResponse, headers map[string]string) {
    w := c.Writer
    for k, v := range headers {
        w.Header().Set(k, v)
    }
    w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    flusher, ok := w.(http.Flusher)
    if !ok {
        c.AbortWithStatus(http.StatusInternalServerError)
        return
    }

    send := func(payload any) {
        data, _ := json.Marshal(payload)
        w.Write([]byte("data: "))
        w.Write(data)
        w.Write([]byte("\n\n"))
        flusher.Flush()
    }

    send(gin.H{"type": "message_start", "message": gin.H{"id": resp.ID, "type": "message", "role": "assistant", "content": []any{}, "model": resp.Model, "stop_reason": nil, "usage": gin.H{"input_tokens": resp.Usage.InputTokens, "output_tokens": 0}}})

    text := ""
    if len(resp.Content) > 0 {
        text = resp.Content[0].Text
    }

    send(gin.H{"type": "content_block_start", "index": 0, "content_block": gin.H{"type": "text", "text": ""}})

    parts := splitWordsWithSpaces(text)
    for _, p := range parts {
        send(gin.H{"type": "content_block_delta", "index": 0, "delta": gin.H{"type": "text_delta", "text": p}})
    }

    send(gin.H{"type": "content_block_stop", "index": 0})
    send(gin.H{"type": "message_delta", "delta": gin.H{"stop_reason": "end_turn"}, "usage": gin.H{"output_tokens": resp.Usage.OutputTokens}})
    send(gin.H{"type": "message_stop"})
}

func splitWordsWithSpaces(text string) []string {
    if text == "" {
        return []string{}
    }
    fields := strings.Fields(text)
    out := make([]string, 0, len(fields))
    for i, f := range fields {
        if i == 0 {
            out = append(out, f)
        } else {
            out = append(out, " "+f)
        }
    }
    return out
}
