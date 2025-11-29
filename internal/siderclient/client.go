package siderclient

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sider2api/internal/session"
	"sider2api/pkg/types"
)

// Client wraps HTTP operations to Sider API.
type Client struct {
	BaseURL             string
	ConversationURL     string
	ChatTimeout         time.Duration
	ConversationTimeout time.Duration
	HTTPClient          *http.Client
	Sessions            *session.SiderSessionManager
}

// New creates a new client with defaults.
func New(baseURL, convURL string, chatTimeout, convTimeout time.Duration, sm *session.SiderSessionManager) *Client {
	return &Client{
		BaseURL:             baseURL,
		ConversationURL:     convURL,
		ChatTimeout:         chatTimeout,
		ConversationTimeout: convTimeout,
		HTTPClient:          &http.Client{},
		Sessions:            sm,
	}
}

// StreamCallback is called for each SSE event during streaming
type StreamCallback func(event types.SiderSSEResponse, partial types.SiderParsedResponse)

// ChatStream posts a chat request and streams SSE responses via callback.
func (c *Client) ChatStream(ctx context.Context, req types.SiderRequest, authToken string, callback StreamCallback) (types.SiderParsedResponse, error) {
	var result types.SiderParsedResponse

	payload, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.ChatTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, strings.NewReader(string(payload)))
	if err != nil {
		return result, fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+authToken)
	httpReq.Header.Set("Origin", "chrome-extension://dhoenijjpgpeimemopealfcbiecgceod")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0")
	httpReq.Header.Set("X-Time-Zone", "Asia/Shanghai")
	httpReq.Header.Set("X-App-Version", "5.13.0")
	httpReq.Header.Set("X-App-Name", "ChitChat_Edge_Ext")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return result, fmt.Errorf("sider request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return result, fmt.Errorf("sider api error: %s %s", resp.Status, string(body))
	}

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		return result, errors.New("expected SSE response from Sider API")
	}

	return c.parseSSEStream(resp.Body, callback)
}

// Chat posts a chat request and parses SSE response.
func (c *Client) Chat(ctx context.Context, req types.SiderRequest, authToken string) (types.SiderParsedResponse, error) {
	var result types.SiderParsedResponse

	payload, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.ChatTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, strings.NewReader(string(payload)))
	if err != nil {
		return result, fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+authToken)
	httpReq.Header.Set("Origin", "chrome-extension://dhoenijjpgpeimemopealfcbiecgceod")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0")
	httpReq.Header.Set("X-Time-Zone", "Asia/Shanghai")
	httpReq.Header.Set("X-App-Version", "5.13.0")
	httpReq.Header.Set("X-App-Name", "ChitChat_Edge_Ext")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return result, fmt.Errorf("sider request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return result, fmt.Errorf("sider api error: %s %s", resp.Status, string(body))
	}

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		return result, errors.New("expected SSE response from Sider API")
	}

	return c.parseSSE(resp.Body)
}

func (c *Client) parseSSEStream(body io.Reader, callback StreamCallback) (types.SiderParsedResponse, error) {
	var result types.SiderParsedResponse
	result.ReasoningParts = []string{}
	result.TextParts = []string{}
	result.ToolResults = []types.SiderToolResult{}

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if dataStr == "[DONE]" {
			break
		}
		var evt types.SiderSSEResponse
		if err := json.Unmarshal([]byte(dataStr), &evt); err != nil {
			// skip malformed lines
			continue
		}
		c.processEvent(&result, evt)

		// Call callback with current event and partial result
		if callback != nil {
			callback(evt, result)
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("read sse: %w", err)
	}
	return result, nil
}

func (c *Client) parseSSE(body io.Reader) (types.SiderParsedResponse, error) {
	return c.parseSSEStream(body, nil)
}

func (c *Client) processEvent(result *types.SiderParsedResponse, evt types.SiderSSEResponse) {
	if evt.Code != 0 {
		return
	}
	data := evt.Data
	result.Model = data.Model

	switch data.Type {
	case "message_start":
		if data.MessageStart != nil {
			result.ConversationID = data.MessageStart.CID
			result.MessageIDs = &types.SiderMessageIDs{User: data.MessageStart.UserMessageID, Assistant: data.MessageStart.AssistantMessageID}
			if c.Sessions != nil {
				c.Sessions.Save(data.MessageStart.CID, data.MessageStart.UserMessageID, data.MessageStart.AssistantMessageID, data.Model)
			}
		}
	case "reasoning_content":
		if rc := data.ReasoningContent; rc != nil && rc.Text != "" {
			result.ReasoningParts = append(result.ReasoningParts, rc.Text)
		}
	case "text":
		if data.Text != "" {
			result.TextParts = append(result.TextParts, data.Text)
		}
	case "tool_call", "tool_call_start", "tool_call_progress", "tool_call_result":
		if data.ToolCall != nil {
			c.handleToolCall(result, data.ToolCall)
		}
	}
}

func (c *Client) handleToolCall(result *types.SiderParsedResponse, tc *types.SiderToolCall) {
	if result.ToolResults == nil {
		result.ToolResults = []types.SiderToolResult{}
	}
	var existing *types.SiderToolResult
	for i := range result.ToolResults {
		if result.ToolResults[i].ToolID == tc.ID {
			existing = &result.ToolResults[i]
			break
		}
	}
	if existing == nil {
		result.ToolResults = append(result.ToolResults, types.SiderToolResult{
			ToolName: tc.Name,
			ToolID:   tc.ID,
			Status:   tc.Status,
			Result:   tc.Search,
			Error:    tc.Error,
		})
		return
	}
	existing.Status = tc.Status
	if tc.Search != nil || tc.Result != nil {
		if tc.Search != nil {
			existing.Result = tc.Search
		} else {
			existing.Result = tc.Result
		}
	}
	if tc.Error != "" {
		existing.Error = tc.Error
	}
}

// ConversationHistory fetches a conversation transcript (optional path matching TS async flow).
type ConversationHistoryResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Conversation any `json:"conversation"`
		Messages     []struct {
			ID              string `json:"id"`
			ParentMessageID string `json:"parent_message_id"`
			Role            string `json:"role"`
			MultiContent    []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"multi_content"`
		} `json:"messages"`
	} `json:"data"`
}

// FetchConversationHistory is provided for parity but not yet used deeply.
func (c *Client) FetchConversationHistory(ctx context.Context, cid, authToken string, limit int) (*ConversationHistoryResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.ConversationTimeout)
	defer cancel()

	payload := map[string]any{"cid": cid, "limit": limit}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ConversationURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Origin", "chrome-extension://dhoenijjpgpeimemopealfcbiecgceod")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0")
	req.Header.Set("X-Time-Zone", "Asia/Shanghai")
	req.Header.Set("X-App-Version", "5.13.0")
	req.Header.Set("X-App-Name", "ChitChat_Edge_Ext")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("conversation history http error: %s", resp.Status)
	}

	var parsed ConversationHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("conversation history error: %s", parsed.Msg)
	}
	return &parsed, nil
}
