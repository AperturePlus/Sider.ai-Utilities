# Sider2API

A proxy server that converts Anthropic Claude API requests to Sider format, with interactive CLI tools for testing.

## Features

- **API Server**: Anthropic-compatible proxy server
- **Interactive Chat**: CLI with streaming output, syntax highlighting, and auto-completion
- **Terminal UI**: Full-screen terminal interface
- **GUI Mode**: System tray application (Windows only)

## Quick Start

```bash
# Install
go build -o sider2api ./cmd/sider2api

# Configure
echo "SIDER_API_TOKEN=your_token_here" > .env

# Run interactive chat
./sider2api chat

# Or start API server
./sider2api serve
```

## Usage

### Interactive Chat

```bash
sider2api chat
```

Features streaming output, syntax highlighting, and command completion.

**Commands:**
- `/model <name>` - Switch model
- `/models` - List available models
- `/think on|off` - Toggle extended thinking
- `/search on|off` - Toggle web search
- `/reset` - Clear conversation
- `/exit` - Quit

### API Server

```bash
# Start server
sider2api serve

# With GUI (Windows, requires -tags=gui build)
sider2api serve --gui
```

Default endpoint: `http://localhost:4141`

Compatible with Anthropic API clients.

### Terminal UI

```bash
sider2api tui
```

Full-screen terminal interface with scrollable history.

## Configuration

Create `.env` file:

```env
SIDER_API_TOKEN=your_token_here
BASE_URL=https://api.sider.ai
HOST=0.0.0.0
PORT=4141
LOG_LEVEL=info
```

## Available Models

- `claude-haiku-4.5` (default)
- `claude-4.5-sonnet`
- `gemini-2.5-flash`
- `gemini-3.0-pro`
- `gpt-5-mini`
- `gpt-5.1`

## Building

```bash
# Standard build (Linux/macOS/WSL)
go build -o sider2api ./cmd/sider2api

# Windows with GUI support
go build -tags=gui -o sider2api.exe ./cmd/sider2api
```

## Platform Support

| Platform | Server | Chat | TUI | GUI |
|----------|--------|------|-----|-----|
| Linux    | ✅     | ✅   | ✅  | ❌  |
| macOS    | ✅     | ✅   | ✅  | ❌  |
| Windows  | ✅     | ✅   | ✅  | ✅* |

*Requires `-tags=gui` build flag


