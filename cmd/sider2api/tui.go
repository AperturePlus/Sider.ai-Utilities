package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"sider2api/internal/config"
	"sider2api/internal/converter"
	appLog "sider2api/internal/log"
	"sider2api/internal/session"
	"sider2api/internal/siderclient"
	"sider2api/pkg/types"
)

var (
	tuiHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("219")).Bold(true)
	tuiStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	tuiUserStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	tuiAIStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	tuiThinkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	tuiErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	tuiPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)
)

const tuiDefaultModelName = "claude-4.5-sonnet"

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Terminal UI chat interface",
		Long:  `Start a terminal UI chat interface with a more visual experience.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
}

func runTUI() error {
	cfg, err := config.Parse([]string{})
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}
	if cfg.SiderAPIToken == "" {
		return fmt.Errorf("SIDER_API_TOKEN is required (set in .env or env)")
	}

	logger := appLog.New(cfg.LogLevel)
	_ = logger

	sessions := session.NewSiderSessionManager(cfg.SiderSessionMaxAge, cfg.ContinuousCID)
	client := siderclient.New(cfg.BaseURL, cfg.ConversationURL, cfg.ChatTimeout, cfg.ConversationTimeout, sessions)

	p := tea.NewProgram(initialTUIModel(cfg, client, sessions), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// tea messages
type chatResultMsg struct {
	resp types.SiderParsedResponse
}

type chatErrorMsg struct {
	err error
}

type statusMsg string

type tuiModel struct {
	cfg             config.Config
	client          *siderclient.Client
	sessions        *session.SiderSessionManager
	input           textinput.Model
	viewport        viewport.Model
	width           int
	height          int
	messages        []string
	modelName       string
	think           bool
	search          bool
	conversationID  string
	parentMessageID string
	history         []types.AnthropicMessage
	sending         bool
	statusLine      string
}

func initialTUIModel(cfg config.Config, client *siderclient.Client, sessions *session.SiderSessionManager) tuiModel {
	ti := textinput.New()
	ti.Placeholder = "Type message or /commands..."
	ti.Focus()

	vp := viewport.Model{Height: 20, Width: 80}
	vp.YPosition = 1

	return tuiModel{
		cfg:        cfg,
		client:     client,
		sessions:   sessions,
		input:      ti,
		viewport:   vp,
		modelName:  tuiDefaultModelName,
		think:      true,
		search:     false,
		messages:   []string{"Welcome to Sider2API TUI. Use /model <name>, /models, /think on|off, /search on|off, /reset, /exit."},
		statusLine: "Ready",
	}
}

// Update
func (m tuiModel) Init() tea.Cmd { return textinput.Blink }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			line := strings.TrimSpace(m.input.Value())
			if line == "" {
				return m, nil
			}
			m.input.SetValue("")
			if strings.HasPrefix(line, "/") {
				handled, status := m.handleCommand(line)
				if handled {
					if status != "" {
						m.setStatus(status)
					}
					m.syncViewport()
					return m, nil
				}
			}
			return m.sendUserMessage(line)
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 5
		m.syncViewport()
	case chatResultMsg:
		m.sending = false
		m.statusLine = "Received response"
		m.renderAI(msg.resp)
		m.syncViewport()
	case chatErrorMsg:
		m.sending = false
		m.statusLine = fmt.Sprintf("Error: %v", msg.err)
		m.messages = append(m.messages, tuiErrorStyle.Render("[error] "+msg.err.Error()))
		m.syncViewport()
	case statusMsg:
		m.statusLine = string(msg)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *tuiModel) sendUserMessage(line string) (tea.Model, tea.Cmd) {
	m.messages = append(m.messages, tuiUserStyle.Render("You:")+" "+line)
	m.history = append(m.history, types.AnthropicMessage{Role: "user", Content: line})
	m.sending = true
	m.statusLine = "Sending..."
	m.syncViewport()

	return m, tea.Batch(tea.Cmd(func() tea.Msg {
		anthropicReq := types.AnthropicRequest{
			Model:    m.modelName,
			Messages: m.history,
			Metadata: &types.AnthropicMetadata{ThinkEnabled: &m.think, SearchEnabled: &m.search},
		}
		siderReq, err := converter.ConvertAnthropicToSider(anthropicReq, converter.ConvertOptions{
			ConversationID:  m.conversationID,
			ParentMessageID: m.parentMessageID,
			ContinuousCID:   m.cfg.ContinuousCID,
		})
		if err != nil {
			return chatErrorMsg{err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), m.cfg.ChatTimeout)
		defer cancel()
		resp, err := m.client.Chat(ctx, siderReq, m.cfg.SiderAPIToken)
		if err != nil {
			return chatErrorMsg{err}
		}
		return chatResultMsg{resp: resp}
	}))
}

func (m *tuiModel) renderAI(resp types.SiderParsedResponse) {
	if resp.ConversationID != "" {
		m.conversationID = resp.ConversationID
	}
	if resp.MessageIDs != nil {
		m.parentMessageID = resp.MessageIDs.Assistant
	}
	reasoning := strings.TrimSpace(strings.Join(resp.ReasoningParts, ""))
	if reasoning != "" {
		m.messages = append(m.messages, tuiThinkStyle.Render("AI (think):\n"+reasoning))
	}
	text := strings.TrimSpace(strings.Join(resp.TextParts, ""))
	m.messages = append(m.messages, tuiAIStyle.Render("AI:")+" "+text)
	m.history = append(m.history, types.AnthropicMessage{Role: "assistant", Content: text})
}

func (m *tuiModel) handleCommand(cmd string) (bool, string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return true, ""
	}
	switch parts[0] {
	case "/exit":
		os.Exit(0)
	case "/reset":
		m.history = nil
		m.messages = append(m.messages, tuiStatusStyle.Render("[reset] conversation cleared"))
		m.conversationID = ""
		m.parentMessageID = ""
		return true, "Reset"
	case "/models":
		m.messages = append(m.messages, tuiStatusStyle.Render("Available models: "+strings.Join(tuiAvailableModels(), ", ")))
		return true, ""
	case "/model":
		if len(parts) < 2 {
			m.messages = append(m.messages, tuiStatusStyle.Render("Usage: /model <name>"))
			return true, ""
		}
		name := parts[1]
		if !tuiIsAllowedModel(name) {
			m.messages = append(m.messages, tuiStatusStyle.Render("Unknown model. Use /models."))
			return true, ""
		}
		m.modelName = name
		return true, fmt.Sprintf("Model set to %s", name)
	case "/think":
		if len(parts) < 2 {
			m.messages = append(m.messages, tuiStatusStyle.Render("Usage: /think on|off"))
			return true, ""
		}
		m.think = strings.EqualFold(parts[1], "on")
		return true, fmt.Sprintf("Think mode: %v", m.think)
	case "/search":
		if len(parts) < 2 {
			m.messages = append(m.messages, tuiStatusStyle.Render("Usage: /search on|off"))
			return true, ""
		}
		m.search = strings.EqualFold(parts[1], "on")
		return true, fmt.Sprintf("Search: %v", m.search)
	}
	return false, ""
}

func (m *tuiModel) syncViewport() {
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
}

func (m *tuiModel) setStatus(s string) {
	m.statusLine = s
}

// View
func (m tuiModel) View() string {
	header := tuiHeaderStyle.Render(fmt.Sprintf("Sider2API TUI | model=%s | think=%v | search=%v", m.modelName, m.think, m.search))
	status := tuiStatusStyle.Render(m.statusLine)
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.viewport.View(),
		status,
		tuiPromptStyle.Render(m.input.View()),
	)
}

// Helpers
func tuiAvailableModels() []string {
	return []string{
		"gemini-2.5-flash",
		"claude-haiku-4.5",
		"gpt-5-mini",
		"gpt-5.1",
		"claude-4.5-sonnet",
		"gemini-3.0-pro",
	}
}

func tuiIsAllowedModel(name string) bool {
	for _, m := range tuiAvailableModels() {
		if m == name {
			return true
		}
	}
	return false
}
