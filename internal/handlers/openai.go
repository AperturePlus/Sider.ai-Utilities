package handlers

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"

    "sider2api/internal/converter"
    "sider2api/pkg/types"
)

// PostChatCompletions handles /v1/chat/completions (OpenAI-compatible)
func (h *Handler) PostChatCompletions(c *gin.Context) {
    authToken, ok := c.Get("authToken")
    if !ok {
        c.JSON(http.StatusUnauthorized, converter.CreateOpenAIErrorResponse("Authentication required", "authentication_error"))
        return
    }
    tokenStr, _ := authToken.(string)

    var req types.OpenAIChatCompletionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, converter.CreateOpenAIErrorResponse(err.Error(), "invalid_request_error"))
        return
    }

    anthropicReq := converter.OpenAIToAnthropic(req)

    conversationID := c.Query("cid")
    if conversationID == "" {
        conversationID = c.GetHeader("X-Conversation-ID")
    }
    if conversationID == "" && hasAssistantHistory(anthropicReq.Messages) {
        conversationID = h.Config.ContinuousCID
    }

    parentMessageID := c.GetHeader("X-Parent-Message-ID")
    if parentMessageID == "" && conversationID != "" {
        parentMessageID = h.Sessions.NextParentMessageID(conversationID)
    }

    siderReq, err := converter.ConvertAnthropicToSider(anthropicReq, converter.ConvertOptions{
        ConversationID:  conversationID,
        ParentMessageID: parentMessageID,
        ContinuousCID:   h.Config.ContinuousCID,
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, converter.CreateOpenAIErrorResponse(err.Error(), "invalid_request_error"))
        return
    }

    ctx := c.Request.Context()
    siderResp, err := h.Client.Chat(ctx, siderReq, tokenStr)
    if err != nil {
        c.JSON(http.StatusInternalServerError, converter.CreateOpenAIErrorResponse(err.Error(), "api_error"))
        return
    }

    anthropicResp := converter.ConvertSiderToAnthropic(siderResp, anthropicReq.Model)
    openaiResp := converter.AnthropicToOpenAIResponse(anthropicResp, req)
    headers := converter.SessionHeadersFromSider(siderResp)

    if req.Stream {
        h.writeOpenAIStream(c, openaiResp, headers)
        return
    }

    for k, v := range headers {
        c.Header(k, v)
    }
    c.JSON(http.StatusOK, openaiResp)
}

func (h *Handler) writeOpenAIStream(c *gin.Context, resp types.OpenAIChatCompletionResponse, headers map[string]string) {
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

    base := map[string]any{
        "id":      resp.ID,
        "object":  "chat.completion.chunk",
        "created": time.Now().Unix(),
        "model":   resp.Model,
    }

    send(merge(base, map[string]any{"choices": []map[string]any{{"index": 0, "delta": map[string]any{"role": "assistant"}, "finish_reason": nil}}}))

    text := ""
    if len(resp.Choices) > 0 {
        text = resp.Choices[0].Message.Content
    }

    parts := splitWordsWithSpaces(text)
    for _, p := range parts {
        send(merge(base, map[string]any{"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": p}, "finish_reason": nil}}}))
    }

    send(merge(base, map[string]any{"choices": []map[string]any{{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}}, "usage": resp.Usage}))
    w.Write([]byte("data: [DONE]\n\n"))
    flusher.Flush()
}

func merge(a map[string]any, b map[string]any) map[string]any {
    out := map[string]any{}
    for k, v := range a {
        out[k] = v
    }
    for k, v := range b {
        out[k] = v
    }
    return out
}
