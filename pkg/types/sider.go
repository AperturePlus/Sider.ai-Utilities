package types

// Sider API request/response types (aligned with existing TS client)

type SiderMultiContent struct {
    Type         string `json:"type"`
    Text         string `json:"text"`
    UserInputText string `json:"user_input_text"`
}

type SiderTools struct {
    Auto      []string        `json:"auto"`
    Image     *SiderImageTool `json:"image,omitempty"`
    Search    *SiderSearchTool `json:"search,omitempty"`
    WebBrowse *SiderWebBrowseTool `json:"web_browse,omitempty"`
    Extras    map[string]any  `json:"-"`
}

type SiderImageTool struct {
    QualityLevel string `json:"quality_level"`
}

type SiderSearchTool struct {
    Enabled    bool  `json:"enabled"`
    MaxResults int   `json:"max_results,omitempty"`
}

type SiderWebBrowseTool struct {
    Enabled bool `json:"enabled"`
    Timeout int  `json:"timeout,omitempty"`
}

type SiderRequest struct {
    CID              string                   `json:"cid"`
    ParentMessageID  string                   `json:"parent_message_id,omitempty"`
    Model            string                   `json:"model"`
    From             string                   `json:"from"`
    FilterSearchHistory bool                  `json:"filter_search_history,omitempty"`
    ChatModels       []string                 `json:"chat_models,omitempty"`
    Quote            any                      `json:"quote,omitempty"`
    ClientPrompt     map[string]any           `json:"client_prompt,omitempty"`
    MultiContent     []SiderMultiContent      `json:"multi_content"`
    PromptTemplates  []SiderPromptTemplate    `json:"prompt_templates"`
    Tools            SiderTools               `json:"tools"`
    ExtraInfo        *SiderExtraInfo          `json:"extra_info,omitempty"`
    OutputLanguage   string                   `json:"output_language,omitempty"`
    ThinkMode        *SiderThinkMode          `json:"think_mode,omitempty"`
}

type SiderPromptTemplate struct {
    Key        string         `json:"key"`
    Attributes map[string]any `json:"attributes"`
}

type SiderExtraInfo struct {
    OriginURL   string `json:"origin_url"`
    OriginTitle string `json:"origin_title"`
}

type SiderThinkMode struct {
    Enable bool `json:"enable"`
}

// SSE raw payloads

type SiderSSEResponse struct {
    Code int             `json:"code"`
    Msg  string          `json:"msg"`
    Data SiderResponseData `json:"data"`
}

type SiderResponseData struct {
    Type string `json:"type"`
    Model string `json:"model"`

    // message_start
    MessageStart *SiderMessageStart `json:"message_start,omitempty"`

    // reasoning_content
    ReasoningContent *SiderReasoningContent `json:"reasoning_content,omitempty"`

    // text
    Text string `json:"text,omitempty"`

    // tool_call variants
    ToolCall *SiderToolCall `json:"tool_call,omitempty"`
}

type SiderMessageStart struct {
    CID               string `json:"cid"`
    UserMessageID     string `json:"user_message_id"`
    AssistantMessageID string `json:"assistant_message_id"`
}

type SiderReasoningContent struct {
    Status string `json:"status"`
    Text   string `json:"text"`
}

type SiderToolCall struct {
    ID     string                 `json:"id"`
    Name   string                 `json:"name"`
    Status string                 `json:"status"`
    Search map[string]any         `json:"search,omitempty"`
    Result any                    `json:"result,omitempty"`
    Error  string                 `json:"error,omitempty"`
    Progress any                  `json:"progress,omitempty"`
}

// Parsed response after SSE aggregation.
type SiderParsedResponse struct {
    ReasoningParts []string
    TextParts      []string
    ToolResults    []SiderToolResult
    Model          string
    ConversationID string
    MessageIDs     *SiderMessageIDs
}

type SiderToolResult struct {
    ToolName string `json:"toolName"`
    ToolID   string `json:"toolId"`
    Result   any    `json:"result"`
    Status   string `json:"status"`
    Error    string `json:"error,omitempty"`
}
