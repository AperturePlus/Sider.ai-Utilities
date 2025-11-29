package converter

import (
    "fmt"
    "math/rand"
    "strings"
    "time"

    "sider2api/pkg/types"
)

// ConvertSiderToAnthropic maps parsed Sider response into Anthropic response.
func ConvertSiderToAnthropic(resp types.SiderParsedResponse, originalModel string) types.AnthropicResponse {
    combined := combineTextParts(resp)
    usage := estimateUsage(resp, combined)

    ar := types.AnthropicResponse{
        ID:         generateResponseID(),
        Type:       "message",
        Role:       "assistant",
        Content:    []types.AnthropicResponseContent{{Type: "text", Text: combined}},
        Model:      originalModel,
        StopReason: "end_turn",
        Usage:      usage,
    }

    if resp.ConversationID != "" {
        ar.SiderSession = &types.SiderSessionInfo{
            ConversationID: resp.ConversationID,
            MessageIDs: resp.MessageIDs,
            ToolResults: resp.ToolResults,
            ReasoningParts: resp.ReasoningParts,
        }
    }

    return ar
}

// SessionHeadersFromSider returns headers to surface session IDs to clients.
func SessionHeadersFromSider(resp types.SiderParsedResponse) map[string]string {
    headers := map[string]string{}
    if resp.ConversationID != "" {
        headers["X-Conversation-ID"] = resp.ConversationID
        if resp.MessageIDs != nil {
            if resp.MessageIDs.Assistant != "" {
                headers["X-Assistant-Message-ID"] = resp.MessageIDs.Assistant
            }
            if resp.MessageIDs.User != "" {
                headers["X-User-Message-ID"] = resp.MessageIDs.User
            }
        }
    }
    return headers
}

// CreateErrorResponse wraps an error into AnthropicResponse.
func CreateErrorResponse(err error, model string) types.AnthropicResponse {
    return types.AnthropicResponse{
        ID:         generateResponseID(),
        Type:       "message",
        Role:       "assistant",
        Content:    []types.AnthropicResponseContent{{Type: "text", Text: "Error: " + err.Error()}},
        Model:      model,
        StopReason: "end_turn",
        Usage:      types.AnthropicUsage{InputTokens: 0, OutputTokens: 0},
    }
}

func combineTextParts(resp types.SiderParsedResponse) string {
    reasoning := strings.TrimSpace(strings.Join(resp.ReasoningParts, ""))
    finalText := strings.TrimSpace(strings.Join(resp.TextParts, ""))

    var b strings.Builder
    if reasoning != "" {
        b.WriteString("<think>\n")
        b.WriteString(reasoning)
        b.WriteString("\n</think>\n\n")
    }
    b.WriteString(finalText)

    if b.Len() == 0 {
        return "Response received but no text content was generated."
    }
    return b.String()
}

func estimateUsage(resp types.SiderParsedResponse, output string) types.AnthropicUsage {
    outputTokens := int((len(output) + 3) / 4)
    reasoningTokens := int((len(strings.Join(resp.ReasoningParts, "")) + 3) / 4)
    return types.AnthropicUsage{
        InputTokens:  10,
        OutputTokens: outputTokens + reasoningTokens,
    }
}

func generateResponseID() string {
    ts := time.Now().UnixMilli()
    randPart := rand.Intn(1_000_000)
    return fmt.Sprintf("msg_%d_%06d", ts, randPart)
}
