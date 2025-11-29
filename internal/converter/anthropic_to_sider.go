package converter

import (
    "errors"
    "fmt"
    "regexp"
    "strings"

    "sider2api/pkg/types"
)

// ConvertOptions controls Anthropic->Sider conversion.
type ConvertOptions struct {
    ConversationID  string
    ParentMessageID string
    ContinuousCID   string
}

// ConvertAnthropicToSider builds a SiderRequest from an AnthropicRequest (non-historical path).
func ConvertAnthropicToSider(req types.AnthropicRequest, opts ConvertOptions) (types.SiderRequest, error) {
    if err := ValidateAnthropicRequest(req); err != nil {
        return types.SiderRequest{}, err
    }

    userMessages := filterMessages(req.Messages, "user")
    if len(userMessages) == 0 {
        return types.SiderRequest{}, errors.New("no user message found in request")
    }
    lastUser := userMessages[len(userMessages)-1]
    currentUserInput := ExtractTextContent(lastUser.Content)

    siderModel := MapModelName(req.Model)
    outputLanguage := DetermineOutputLanguage(currentUserInput)
    promptTemplates := DefaultPromptTemplates()
    thinkMode := BuildThinkMode(req)
    tools := BuildTools(req)
    clientPrompt := BuildClientPrompt(req)

    text := buildRequestText(req, currentUserInput)

    sr := types.SiderRequest{
        CID:     opts.ConversationID,
        ParentMessageID: opts.ParentMessageID,
        Model:   siderModel,
        From:    "chat",
        MultiContent: []types.SiderMultiContent{{
            Type:          "text",
            Text:          text,
            UserInputText: currentUserInput,
        }},
        PromptTemplates: promptTemplates,
        Tools:           tools,
        ClientPrompt:    clientPrompt,
        ExtraInfo: &types.SiderExtraInfo{
            OriginURL:   "chrome-extension://dhoenijjpgpeimemopealfcbiecgceod/standalone.html?from=sidebar",
            OriginTitle: "Sider",
        },
        OutputLanguage: outputLanguage,
        ThinkMode:      &types.SiderThinkMode{Enable: thinkMode},
    }

    return sr, nil
}

// ValidateAnthropicRequest performs basic validation similar to TS implementation.
func ValidateAnthropicRequest(req types.AnthropicRequest) error {
    if req.Model == "" {
        return errors.New("missing required field: model")
    }
    if len(req.Messages) == 0 {
        return errors.New("messages array cannot be empty")
    }
    hasUser := false
    for _, m := range req.Messages {
        if m.Role == "user" {
            hasUser = true
        }
        if m.Role != "user" && m.Role != "assistant" {
            return errors.New("invalid message role. must be 'user' or 'assistant'")
        }
        if m.Content == nil {
            return errors.New("message content cannot be empty")
        }
    }
    if !hasUser {
        return errors.New("at least one user message is required")
    }
    return nil
}

// ExtractTextContent handles string or []AnthropicContent content payloads.
func ExtractTextContent(content any) string {
    switch v := content.(type) {
    case string:
        return strings.TrimSpace(v)
    case []types.AnthropicContent:
        var b strings.Builder
        for _, c := range v {
            if c.Type == "text" && c.Text != "" {
                if b.Len() > 0 {
                    b.WriteString("\n")
                }
                b.WriteString(c.Text)
            }
        }
        return strings.TrimSpace(b.String())
    default:
        return ""
    }
}

// MapModelName converts Anthropic model names to Sider equivalents.
func MapModelName(model string) string {
    normalized := strings.ToLower(model)
    mapping := map[string]string{
        "claude-3.7-sonnet":      "claude-3.7-sonnet-think",
        "claude-3-7-sonnet":      "claude-3.7-sonnet-think",
        "claude-3.7":             "claude-3.7-sonnet-think",
        "claude-4-sonnet":        "claude-4-sonnet-think",
        "claude-4":               "claude-4-sonnet-think",
        "claude-sonnet-4":        "claude-4-sonnet-think",
        "claude-3-sonnet":        "claude-3.7-sonnet-think",
        "claude-sonnet":          "claude-3.7-sonnet-think",
    }
    if mapped, ok := mapping[normalized]; ok {
        return mapped
    }
    return model
}

// DetermineOutputLanguage infers output language from latest user input.
func DetermineOutputLanguage(input string) string {
    if input == "" {
        return "en"
    }
    cjk := regexp.MustCompile(`[一-鿿]`)
    if cjk.MatchString(input) {
        return "zh-CN"
    }
    return "en"
}

// BuildThinkMode decides whether to enable thinking mode.
func BuildThinkMode(req types.AnthropicRequest) bool {
    if req.Metadata != nil && req.Metadata.ThinkEnabled != nil {
        return *req.Metadata.ThinkEnabled
    }
    return true
}

// BuildClientPrompt keeps only safe numeric knobs.
func BuildClientPrompt(req types.AnthropicRequest) map[string]any {
    prompt := map[string]any{}
    if req.Temperature != nil {
        t := *req.Temperature
        if t >= 0 && t <= 1 {
            prompt["temperature"] = t
        }
    }
    return prompt
}

// DefaultPromptTemplates mirrors TS defaults.
func DefaultPromptTemplates() []types.SiderPromptTemplate {
    return []types.SiderPromptTemplate{
        {Key: "artifacts", Attributes: map[string]any{"lang": "original"}},
        {Key: "thinking_mode", Attributes: map[string]any{}},
    }
}

// BuildTools maps Anthropic tools to Sider tools config.
func BuildTools(req types.AnthropicRequest) types.SiderTools {
    metadataSearch := false
    if req.Metadata != nil && req.Metadata.SearchEnabled != nil {
        metadataSearch = *req.Metadata.SearchEnabled
    }

    tools := types.SiderTools{Auto: []string{}}

    if len(req.Tools) == 0 {
        if metadataSearch {
            tools.Auto = append(tools.Auto, "search")
        }
        return tools
    }

    nameMap := map[string]string{
        "create_image":  "create_image",
        "generate_image": "create_image",
        "image_generation": "create_image",
        "web_search":    "search",
        "search_web":    "search",
        "internet_search": "search",
        "browse_web":    "web_browse",
        "web_browsing":  "web_browse",
        "visit_url":     "web_browse",
    }

    for _, tool := range req.Tools {
        mapped := nameMap[tool.Name]
        if mapped == "" {
            mapped = tool.Name
        }
        if !contains(tools.Auto, mapped) {
            tools.Auto = append(tools.Auto, mapped)
        }
        switch mapped {
        case "create_image":
            tools.Image = &types.SiderImageTool{QualityLevel: "high"}
        case "search":
            tools.Search = &types.SiderSearchTool{Enabled: true, MaxResults: 10}
        case "web_browse":
            tools.WebBrowse = &types.SiderWebBrowseTool{Enabled: true, Timeout: 30}
        }
    }

    if metadataSearch {
        if !contains(tools.Auto, "search") {
            tools.Auto = append(tools.Auto, "search")
        }
    } else {
        // remove search if metadata explicitly disabled
        filtered := tools.Auto[:0]
        for _, t := range tools.Auto {
            if t != "search" {
                filtered = append(filtered, t)
            }
        }
        tools.Auto = filtered
    }

    return tools
}

func contains(list []string, v string) bool {
    for _, item := range list {
        if item == v {
            return true
        }
    }
    return false
}

// buildRequestText injects minimal context similar to TS simplified mode.
func buildRequestText(req types.AnthropicRequest, current string) string {
    if len(req.Messages) == 1 {
        if req.System != "" {
            return strings.TrimSpace(req.System + "\n\n" + current)
        }
        return current
    }

    var context strings.Builder
    if req.System != "" {
        context.WriteString("System: ")
        context.WriteString(req.System)
        context.WriteString("\n\n")
    }

    // include last up to 2 previous messages (excluding last user)
    if len(req.Messages) > 1 {
        start := len(req.Messages) - 3
        if start < 0 {
            start = 0
        }
        for i := start; i < len(req.Messages)-1; i++ {
            m := req.Messages[i]
            content := ExtractTextContent(m.Content)
            if len(content) > 100 {
                content = content[:100] + "..."
            }
            if content != "" {
                role := "Human"
                if m.Role == "assistant" {
                    role = "Assistant"
                }
                context.WriteString(fmt.Sprintf("%s: %s\n", role, content))
            }
        }
    }

    if context.Len() == 0 {
        return current
    }

    if context.Len() > 300 {
        trimmed := context.String()[:300]
        context.Reset()
        context.WriteString(trimmed)
        context.WriteString("...\n")
    }

    context.WriteString("Current: ")
    context.WriteString(current)
    return context.String()
}

// filterMessages selects messages by role.
func filterMessages(messages []types.AnthropicMessage, role string) []types.AnthropicMessage {
    out := make([]types.AnthropicMessage, 0, len(messages))
    for _, m := range messages {
        if m.Role == role {
            out = append(out, m)
        }
    }
    return out
}
