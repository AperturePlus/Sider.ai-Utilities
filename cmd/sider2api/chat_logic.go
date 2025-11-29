package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/reeflective/readline"
	"github.com/spf13/cobra"

	"sider2api/internal/config"
	"sider2api/internal/converter"
	appLog "sider2api/internal/log"
	"sider2api/internal/session"
	"sider2api/internal/siderclient"
	"sider2api/pkg/types"
)

const (
	chatPromptColor  = "\u001b[36m"
	chatGreen        = "\u001b[32m"
	chatYellow       = "\u001b[33m"
	chatMagenta      = "\u001b[35m"
	chatRed          = "\u001b[31m"
	chatGray         = "\u001b[90m"
	chatResetColor   = "\u001b[0m"
	chatCyan         = "\u001b[36m"
	chatDefaultModel = "claude-haiku-4.5"
)

func chatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Interactive chat with AI models",
		Long:  `Start an interactive chat session with AI models. Supports syntax highlighting and command completion.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChat()
		},
	}
}

// Simple CLI chat that talks to Sider directly using the Go converters/client.
func runChat() error {
	cfg, err := config.Parse([]string{})
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}
	if cfg.SiderAPIToken == "" {
		return fmt.Errorf("SIDER_API_TOKEN is required (set in .env or env)")
	}

	// set reasonable defaults for chat
	if cfg.Port == 0 {
		cfg.Port = 4141
	}

	_ = appLog.New(cfg.LogLevel)

	sessions := session.NewSiderSessionManager(cfg.SiderSessionMaxAge, cfg.ContinuousCID)
	client := siderclient.New(cfg.BaseURL, cfg.ConversationURL, cfg.ChatTimeout, cfg.ConversationTimeout, sessions)

	model := chatDefaultModel

	fmt.Println("Sider2API CLI chat. Commands: " +
		colorize("/model <name>", chatMagenta) + ", " +
		colorize("/models", chatMagenta) + ", " +
		colorize("/think on|off", chatMagenta) + ", " +
		colorize("/search on|off", chatMagenta) + ", " +
		colorize("/reset", chatMagenta) + ", " +
		colorize("/exit", chatMagenta))

	conversationID := ""
	parentMessageID := ""
	var history []types.AnthropicMessage
	thinkEnabled := true
	searchEnabled := false

	executor := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		if strings.HasPrefix(line, "/") {
			if handleCommand(line, &model, &thinkEnabled, &searchEnabled, &history, &conversationID, &parentMessageID) {
				return
			}
			// unknown command fall-through to chat
		}

		history = append(history, types.AnthropicMessage{Role: "user", Content: line})
		fmt.Print("\033[1A\r\033[K") // Move up one line and clear it
		printLine("You", line, chatCyan)
		anthropicReq := types.AnthropicRequest{
			Model:    model,
			Messages: history,
			Metadata: &types.AnthropicMetadata{
				ThinkEnabled:  &thinkEnabled,
				SearchEnabled: &searchEnabled,
			},
		}

		siderReq, err := converter.ConvertAnthropicToSider(anthropicReq, converter.ConvertOptions{
			ConversationID:  conversationID,
			ParentMessageID: parentMessageID,
			ContinuousCID:   cfg.ContinuousCID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sconvert error:%s %v\n", chatRed, chatResetColor, err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.ChatTimeout)
		defer cancel()

		// Stream output
		var thinkStarted bool
		var textStarted bool
		var toolActive bool
		var lastReasoningLen int
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		var spinnerIdx int

		resp, err := client.ChatStream(ctx, siderReq, cfg.SiderAPIToken, func(evt types.SiderSSEResponse, partial types.SiderParsedResponse) {
			// Handle tool calls (search, etc.)
			if evt.Data.Type == "tool_call" || evt.Data.Type == "tool_call_start" || evt.Data.Type == "tool_call_progress" {
				if evt.Data.ToolCall != nil {
					if !toolActive {
						if thinkStarted {
							fmt.Print("\n")
						}
						toolActive = true
					}

					// Show spinner with tool name
					toolName := evt.Data.ToolCall.Name
					if toolName == "" {
						toolName = "tool"
					}
					status := evt.Data.ToolCall.Status
					frame := spinnerFrames[spinnerIdx%len(spinnerFrames)]
					spinnerIdx++

					fmt.Printf("\r%s%s %s %s...%s", chatYellow, frame, toolName, status, chatResetColor)
				}
			}

			// Tool completed
			if evt.Data.Type == "tool_call_result" {
				if toolActive {
					fmt.Print("\r\033[K") // Clear the spinner line
					toolActive = false
				}
			}

			// Stream thinking content
			if thinkEnabled && evt.Data.Type == "reasoning_content" && evt.Data.ReasoningContent != nil && evt.Data.ReasoningContent.Text != "" {
				if toolActive {
					fmt.Print("\r\033[K") // Clear spinner if active
					toolActive = false
				}
				if !thinkStarted {
					fmt.Printf("%s%s (think):%s ", chatGray, model, chatResetColor)
					thinkStarted = true
				}
				fmt.Print(evt.Data.ReasoningContent.Text)
				lastReasoningLen += len(evt.Data.ReasoningContent.Text)
			}

			// When text starts, finish think section and start text section
			if evt.Data.Type == "text" && evt.Data.Text != "" {
				if toolActive {
					fmt.Print("\r\033[K") // Clear spinner if active
					toolActive = false
				}
				if !textStarted {
					if thinkStarted {
						fmt.Print("\n")
					}
					fmt.Printf("%s%s:%s ", chatGreen, model, chatResetColor)
					textStarted = true
				}
				fmt.Print(evt.Data.Text)
			}
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "\n%schat error:%s %v\n", chatRed, chatResetColor, err)
			return
		}

		fmt.Print("\n")

		if resp.ConversationID != "" {
			conversationID = resp.ConversationID
		}
		if resp.MessageIDs != nil {
			parentMessageID = resp.MessageIDs.Assistant
		}

		text := strings.TrimSpace(strings.Join(resp.TextParts, ""))
		history = append(history, types.AnthropicMessage{Role: "assistant", Content: text})
	}

	// Setup readline with completions
	rl := readline.NewShell()
	rl.Prompt.Primary(func() string { return promptForModel(model) })

	// Set up syntax highlighter for commands
	rl.SyntaxHighlighter = func(line []rune) string {
		text := string(line)
		if strings.HasPrefix(text, "/") {
			parts := strings.Fields(text)
			if len(parts) > 0 {
				cmd := parts[0]
				// Check if it's a valid command
				validCommands := []string{"/model", "/models", "/think", "/search", "/reset", "/exit"}
				isValid := false
				for _, validCmd := range validCommands {
					if cmd == validCmd {
						isValid = true
						break
					}
				}

				if isValid {
					// Highlight valid command in chatMagenta
					highlighted := chatMagenta + cmd + chatResetColor
					if len(parts) > 1 {
						// Highlight arguments in chatCyan
						highlighted += " " + chatCyan + strings.Join(parts[1:], " ") + chatResetColor
					}
					return highlighted
				} else {
					// Invalid command - highlight in chatRed
					highlighted := chatRed + cmd + chatResetColor
					if len(parts) > 1 {
						// Keep arguments in default color
						highlighted += " " + strings.Join(parts[1:], " ")
					}
					return highlighted
				}
			}
		}
		return text
	}

	// Set up completer function
	rl.Completer = func(line []rune, pos int) readline.Completions {
		text := string(line[:pos])

		// Command completions
		if strings.HasPrefix(text, "/") {
			parts := strings.Fields(text)
			if len(parts) == 0 {
				return readline.CompleteValues("/model", "/models", "/think", "/search", "/reset", "/exit")
			}

			cmd := parts[0]

			// Complete command names
			if len(parts) == 1 && !strings.HasSuffix(text, " ") {
				return readline.CompleteValues("/model", "/models", "/think", "/search", "/reset", "/exit")
			}

			// Complete arguments
			switch cmd {
			case "/model":
				if len(parts) >= 1 {
					return readline.CompleteValues(availableModels()...)
				}
			case "/think", "/search":
				if len(parts) >= 1 {
					return readline.CompleteValues("on", "off")
				}
			}
		}

		return readline.Completions{}
	}

	// Main loop
	for {
		line, err := rl.Readline()
		if err != nil {
			return nil
		}
		executor(line)
	}
}

func promptForModel(model string) string {
	return fmt.Sprintf("%s[%s]%s > ", chatPromptColor, model, chatResetColor)
}

// handleCommand mutates state and returns true if command was handled.
func handleCommand(cmd string, model *string, thinkEnabled, searchEnabled *bool, history *[]types.AnthropicMessage, cid, parentID *string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return true
	}
	switch parts[0] {
	case "/exit":
		fmt.Println(colorize("Bye!", chatYellow))
		os.Exit(0)
	case "/reset":
		*history = nil
		*cid = ""
		*parentID = ""
		printLine("Reset", "Conversation cleared.", chatYellow)
		return true
	case "/models":
		printLine("Models", strings.Join(availableModels(), ", "), chatMagenta)
		return true
	case "/model":
		if len(parts) < 2 {
			fmt.Println(colorize("Usage: /model <name>", chatRed))
			return true
		}
		name := strings.TrimSpace(parts[1])
		if !isAllowedModel(name) {
			fmt.Println(colorize("Unknown model. Use /models to list.", chatRed))
			return true
		}
		*model = name
		printLine("Model", "Set to "+name, chatYellow)
		return true
	case "/think":
		if len(parts) < 2 {
			fmt.Println(colorize("Usage: /think on|off", chatRed))
			return true
		}
		*thinkEnabled = strings.EqualFold(parts[1], "on")
		printLine("Think", fmt.Sprintf("Think mode: %v", *thinkEnabled), chatYellow)
		return true
	case "/search":
		if len(parts) < 2 {
			fmt.Println(colorize("Usage: /search on|off", chatRed))
			return true
		}
		*searchEnabled = strings.EqualFold(parts[1], "on")
		printLine("Search", fmt.Sprintf("Search enabled: %v", *searchEnabled), chatYellow)
		return true
	}
	// unknown command
	return false
}

func availableModels() []string {
	return []string{
		"gemini-2.5-flash",
		"claude-haiku-4.5",
		"gpt-5-mini",
		"gpt-5.1",
		"claude-4.5-sonnet",
		"gemini-3.0-pro",
	}
}

func isAllowedModel(name string) bool {
	for _, m := range availableModels() {
		if m == name {
			return true
		}
	}
	return false
}

func spinner(prefix string, done <-chan struct{}) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	for {
		select {
		case <-done:
			return
		default:
			colorizedPrefix := gradient(prefix, 33, 45, i)
			fmt.Printf("\r%s %s", colorizedPrefix, frames[i%len(frames)])
			time.Sleep(120 * time.Millisecond)
			i++
		}
	}
}

func colorize(text, color string) string {
	return color + text + chatResetColor
}

func gradient(text string, startColor, endColor, step int) string {
	if text == "" {
		return ""
	}
	runes := []rune(text)
	out := strings.Builder{}
	length := len(runes)
	for idx, r := range runes {
		t := float64(idx) / float64(max(1, length-1))
		color := int(float64(startColor)*(1-t) + float64(endColor)*t)
		// small animated shift
		color = (color + step) % 256
		out.WriteString(fmt.Sprintf("\u001b[38;5;%dm%s", color, string(r)))
	}
	out.WriteString(chatResetColor)
	return out.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func printLine(label, text, color string) {
	if text == "" {
		return
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return
	}
	indent := strings.Repeat(" ", len([]rune(label))+2)
	fmt.Printf("%s%s:%s %s\n", color, label, chatResetColor, lines[0])
	for _, l := range lines[1:] {
		fmt.Printf("%s%s\n", indent, l)
	}
}
