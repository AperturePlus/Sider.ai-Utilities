package converter

import (
    "strings"
    "time"

    "sider2api/pkg/types"
)

// OpenAIToAnthropic converts OpenAI chat completion request to Anthropic format.
func OpenAIToAnthropic(req types.OpenAIChatCompletionRequest) types.AnthropicRequest {
    var anthropicMessages []types.AnthropicMessage
    var systemMessages []string

    for _, m := range req.Messages {
        switch m.Role {
        case "system":
            systemMessages = append(systemMessages, normalizeOpenAIContent(m.Content))
        case "user":
            anthropicMessages = append(anthropicMessages, types.AnthropicMessage{Role: "user", Content: normalizeOpenAIContent(m.Content)})
        case "assistant":
            anthropicMessages = append(anthropicMessages, types.AnthropicMessage{Role: "assistant", Content: normalizeOpenAIContent(m.Content)})
        case "tool", "function":
            anthropicMessages = append(anthropicMessages, types.AnthropicMessage{Role: "assistant", Content: buildToolResultContent(m)})
        }
    }

    system := strings.TrimSpace(strings.Join(systemMessages, "\n\n"))

    tools := convertOpenAITools(req.Tools)
    toolChoice := convertOpenAIToolChoice(req.ToolChoice)

    ar := types.AnthropicRequest{
        Model:    req.Model,
        Messages: anthropicMessages,
        Stream:   req.Stream,
    }
    if req.MaxTokens != nil {
        ar.MaxTokens = req.MaxTokens
    }
    if req.Temperature != nil {
        ar.Temperature = req.Temperature
    }
    if req.TopP != nil {
        ar.TopP = req.TopP
    }
    if system != "" {
        ar.System = system
    }
    if len(tools) > 0 {
        ar.Tools = tools
    }
    if toolChoice != nil {
        ar.ToolChoice = toolChoice
    }

    return ar
}

// AnthropicToOpenAIResponse reshapes Anthropic response to OpenAI chat completion response.
func AnthropicToOpenAIResponse(resp types.AnthropicResponse, req types.OpenAIChatCompletionRequest) types.OpenAIChatCompletionResponse {
    text := ""
    for _, c := range resp.Content {
        if c.Type == "text" {
            text += c.Text
        }
    }

    choice := types.OpenAIChatCompletionChoice{
        Index: 0,
        Message: types.OpenAIChatMessageSimple{Role: "assistant", Content: text},
        FinishReason: mapStopReason(resp.StopReason),
        Logprobs: nil,
    }

    usage := types.OpenAIUsage{
        PromptTokens:     resp.Usage.InputTokens,
        CompletionTokens: resp.Usage.OutputTokens,
        TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
    }

    id := resp.ID
    if strings.HasPrefix(id, "msg_") {
        id = "chatcmpl-" + strings.TrimPrefix(id, "msg_")
    } else if !strings.HasPrefix(id, "chatcmpl-") {
        id = "chatcmpl-" + id
    }

    return types.OpenAIChatCompletionResponse{
        ID:      id,
        Object:  "chat.completion",
        Created: time.Now().Unix(),
        Model:   req.Model,
        Choices: []types.OpenAIChatCompletionChoice{choice},
        Usage:   usage,
        SiderSession: resp.SiderSession,
    }
}

// CreateOpenAIErrorResponse builds an OpenAI-style error object.
func CreateOpenAIErrorResponse(message, typ string) types.OpenAIErrorResponse {
    return types.OpenAIErrorResponse{Error: types.OpenAIError{Message: message, Type: typ}}
}

// Helpers

func normalizeOpenAIContent(content any) string {
    switch v := content.(type) {
    case string:
        return v
    case []types.OpenAIChatMessageContent:
        var b strings.Builder
        for _, part := range v {
            switch part.Type {
            case "text", "input_text":
                b.WriteString(part.Text)
            case "tool_result":
                b.WriteString(part.Content)
            case "image_url":
                if part.ImageURL != nil {
                    b.WriteString("[image:" + part.ImageURL.URL + "]")
                }
            }
            if b.Len() > 0 {
                b.WriteString("\n")
            }
        }
        return strings.TrimSpace(b.String())
    default:
        return ""
    }
}

func buildToolResultContent(msg types.OpenAIChatMessage) string {
    content := normalizeOpenAIContent(msg.Content)
    if msg.Name != "" {
        return "Tool " + msg.Name + " result:\n" + content
    }
    if content == "" {
        return "[empty tool result]"
    }
    return content
}

func convertOpenAITools(tools []types.OpenAIToolDefinition) []types.AnthropicTool {
    if len(tools) == 0 {
        return nil
    }
    out := make([]types.AnthropicTool, 0, len(tools))
    for _, t := range tools {
        props := map[string]any{}
        if t.Function.Parameters != nil {
            if p, ok := t.Function.Parameters["properties"].(map[string]any); ok {
                props = p
            }
        }
        var required []string
        if t.Function.Parameters != nil {
            if raw, ok := t.Function.Parameters["required"].([]string); ok {
                required = raw
            }
        }
        out = append(out, types.AnthropicTool{
            Name:        t.Function.Name,
            Description: t.Function.Description,
            InputSchema: types.AnthropicToolInputSchema{Type: "object", Properties: props, Required: required},
        })
    }
    return out
}

func convertOpenAIToolChoice(choice any) *types.AnthropicToolChoice {
    switch v := choice.(type) {
    case string:
        if v == "auto" || v == "none" {
            return &types.AnthropicToolChoice{Type: "auto"}
        }
    case map[string]any:
        if v["type"] == "function" {
            if fn, ok := v["function"].(map[string]any); ok {
                if name, ok2 := fn["name"].(string); ok2 {
                    return &types.AnthropicToolChoice{Type: "tool", Name: name}
                }
            }
        }
    }
    return nil
}

func mapStopReason(stop string) string {
    switch stop {
    case "max_tokens":
        return "length"
    case "stop_sequence", "end_turn":
        return "stop"
    default:
        return stop
    }
}
