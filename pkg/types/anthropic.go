package types

// Anthropic API types (aligned to TS definitions)
// Reference: https://docs.anthropic.com/claude/reference/messages_post

// AnthropicMessage represents a message in the conversation.
type AnthropicMessage struct {
    Role    string             `json:"role"`
    Content any                `json:"content"` // string or []AnthropicContent
}

// AnthropicContent covers text/image/tool payloads.
type AnthropicContent struct {
    Type   string               `json:"type"`
    Text   string               `json:"text,omitempty"`
    Source *AnthropicImageSource `json:"source,omitempty"`
    // Tool use / result
    ID       string                 `json:"id,omitempty"`
    Name     string                 `json:"name,omitempty"`
    Input    map[string]any         `json:"input,omitempty"`
    Content  []AnthropicContent     `json:"content,omitempty"`
    IsError  *bool                  `json:"is_error,omitempty"`
}

// AnthropicImageSource represents base64 image content.
type AnthropicImageSource struct {
    Type     string `json:"type"`
    MediaType string `json:"media_type"`
    Data     string `json:"data"`
}

// AnthropicRequest mirrors the messages API request body.
type AnthropicRequest struct {
    Model       string              `json:"model"`
    Messages    []AnthropicMessage  `json:"messages"`
    MaxTokens   *int                `json:"max_tokens,omitempty"`
    Temperature *float64            `json:"temperature,omitempty"`
    TopP        *float64            `json:"top_p,omitempty"`
    TopK        *int                `json:"top_k,omitempty"`
    StopSeq     []string            `json:"stop_sequences,omitempty"`
    Stream      bool                `json:"stream,omitempty"`
    System      string              `json:"system,omitempty"`
    Tools       []AnthropicTool     `json:"tools,omitempty"`
    ToolChoice  *AnthropicToolChoice `json:"tool_choice,omitempty"`
    Metadata    *AnthropicMetadata  `json:"metadata,omitempty"`
}

// AnthropicMetadata holds optional request metadata.
type AnthropicMetadata struct {
    UserID       string `json:"user_id,omitempty"`
    ThinkEnabled *bool  `json:"think_enabled,omitempty"`
    SearchEnabled *bool `json:"search_enabled,omitempty"`
}

// AnthropicResponse represents the non-streaming response.
type AnthropicResponse struct {
    ID         string                     `json:"id"`
    Type       string                     `json:"type"`
    Role       string                     `json:"role"`
    Content    []AnthropicResponseContent `json:"content"`
    Model      string                     `json:"model"`
    StopReason string                     `json:"stop_reason"`
    StopSeq    string                     `json:"stop_sequence,omitempty"`
    Usage      AnthropicUsage             `json:"usage"`
    // Extended Sider session info (custom)
    SiderSession *SiderSessionInfo `json:"sider_session,omitempty"`
}

// AnthropicUsage captures token counts.
type AnthropicUsage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
}

// AnthropicResponseContent is text-only in this project.
type AnthropicResponseContent struct {
    Type string `json:"type"`
    Text string `json:"text"`
}

// AnthropicStreamEvent is used for SSE.
type AnthropicStreamEvent struct {
    Type        string                     `json:"type"`
    Message     *AnthropicResponse         `json:"message,omitempty"`
    Content     *AnthropicResponseContent  `json:"content_block,omitempty"`
    Delta       *AnthropicTextDelta        `json:"delta,omitempty"`
    Usage       *AnthropicUsage            `json:"usage,omitempty"`
}

// AnthropicTextDelta describes incremental text tokens.
type AnthropicTextDelta struct {
    Type string `json:"type"`
    Text string `json:"text"`
}

// AnthropicError is used for error responses.
type AnthropicError struct {
    Type  string                `json:"type"`
    Error AnthropicErrorDetails `json:"error"`
}

// AnthropicErrorDetails details error reason.
type AnthropicErrorDetails struct {
    Type    string `json:"type"`
    Message string `json:"message"`
}

// Token count request/response.
type AnthropicTokenCountRequest struct {
    Model    string             `json:"model"`
    Messages []AnthropicMessage `json:"messages"`
    System   string             `json:"system,omitempty"`
}

type AnthropicTokenCountResponse struct {
    InputTokens int `json:"input_tokens"`
}

// Tool definitions.
type AnthropicTool struct {
    Name        string                  `json:"name"`
    Description string                  `json:"description,omitempty"`
    InputSchema AnthropicToolInputSchema `json:"input_schema"`
}

type AnthropicToolInputSchema struct {
    Type       string            `json:"type"`
    Properties map[string]any    `json:"properties"`
    Required   []string          `json:"required,omitempty"`
}

type AnthropicToolChoice struct {
    Type string `json:"type"`
    Name string `json:"name,omitempty"`
}

// Tool use / result content.
type AnthropicToolUse struct {
    Type string         `json:"type"`
    ID   string         `json:"id"`
    Name string         `json:"name"`
    Input map[string]any `json:"input"`
}

type AnthropicToolResult struct {
    Type     string             `json:"type"`
    ToolUseID string            `json:"tool_use_id"`
    Content  []AnthropicContent `json:"content,omitempty"`
    IsError  bool               `json:"is_error,omitempty"`
}

// SiderSessionInfo extends responses with upstream session IDs.
type SiderSessionInfo struct {
    ConversationID string               `json:"conversation_id"`
    MessageIDs     *SiderMessageIDs     `json:"message_ids,omitempty"`
    ToolResults    []SiderToolResult    `json:"tool_results,omitempty"`
    ReasoningParts []string             `json:"reasoning_parts,omitempty"`
}

// SiderMessageIDs holds user/assistant ids.
type SiderMessageIDs struct {
    User      string `json:"user,omitempty"`
    Assistant string `json:"assistant,omitempty"`
}

// HealthResponse is used by /health.
type HealthResponse struct {
    Status    string `json:"status"`
    Service   string `json:"service"`
    Version   string `json:"version"`
    Timestamp string `json:"timestamp"`
    TechStack string `json:"tech_stack,omitempty"`
}
