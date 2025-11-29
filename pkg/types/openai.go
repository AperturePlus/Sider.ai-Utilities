package types

// OpenAI Chat Completions compatible types (subset used by project)

type OpenAIChatCompletionRequest struct {
    Model            string                   `json:"model"`
    Messages         []OpenAIChatMessage      `json:"messages"`
    MaxTokens        *int                     `json:"max_tokens,omitempty"`
    Temperature      *float64                 `json:"temperature,omitempty"`
    TopP             *float64                 `json:"top_p,omitempty"`
    Stream           bool                     `json:"stream,omitempty"`
    Tools            []OpenAIToolDefinition   `json:"tools,omitempty"`
    ToolChoice       any                      `json:"tool_choice,omitempty"` // "none" | "auto" | {type:function}
    ResponseFormat   map[string]any           `json:"response_format,omitempty"`
}

type OpenAIChatMessage struct {
    Role       string                   `json:"role"`
    Content    any                      `json:"content"` // string or []OpenAIChatMessageContent
    Name       string                   `json:"name,omitempty"`
    ToolCallID string                   `json:"tool_call_id,omitempty"`
}

type OpenAIChatMessageContent struct {
    Type     string                 `json:"type"`
    Text     string                 `json:"text,omitempty"`
    Content  string                 `json:"content,omitempty"` // for tool_result
    ImageURL *OpenAIImageURL        `json:"image_url,omitempty"`
}

type OpenAIImageURL struct {
    URL string `json:"url"`
}

type OpenAIToolDefinition struct {
    Type     string               `json:"type"`
    Function OpenAIFunctionSpec   `json:"function"`
}

type OpenAIFunctionSpec struct {
    Name        string         `json:"name"`
    Description string         `json:"description,omitempty"`
    Parameters  map[string]any `json:"parameters,omitempty"`
}

// OpenAIChatCompletionResponse is the non-streaming form.
type OpenAIChatCompletionResponse struct {
    ID      string                         `json:"id"`
    Object  string                         `json:"object"`
    Created int64                          `json:"created"`
    Model   string                         `json:"model"`
    Choices []OpenAIChatCompletionChoice   `json:"choices"`
    Usage   OpenAIUsage                    `json:"usage"`
    SiderSession *SiderSessionInfo         `json:"sider_session,omitempty"`
}

type OpenAIChatCompletionChoice struct {
    Index        int                      `json:"index"`
    Message      OpenAIChatMessageSimple  `json:"message"`
    FinishReason string                   `json:"finish_reason"`
    Logprobs     any                      `json:"logprobs"`
}

type OpenAIChatMessageSimple struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type OpenAIUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

// Streaming chunk type.
type OpenAIChatCompletionChunk struct {
    ID      string                         `json:"id"`
    Object  string                         `json:"object"`
    Created int64                          `json:"created"`
    Model   string                         `json:"model"`
    Choices []OpenAIChatCompletionDelta    `json:"choices"`
    Usage   *OpenAIUsage                   `json:"usage,omitempty"`
}

type OpenAIChatCompletionDelta struct {
    Index        int                      `json:"index"`
    Delta        OpenAIChatDeltaContent   `json:"delta"`
    FinishReason string                   `json:"finish_reason"`
}

type OpenAIChatDeltaContent struct {
    Role    string `json:"role,omitempty"`
    Content string `json:"content,omitempty"`
}

// OpenAIErrorResponse matches OpenAI error structure.
type OpenAIErrorResponse struct {
    Error OpenAIError `json:"error"`
}

type OpenAIError struct {
    Message string `json:"message"`
    Type    string `json:"type"`
    Code    string `json:"code,omitempty"`
}
