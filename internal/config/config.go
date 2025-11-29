package config

import (
    "bufio"
    "flag"
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"
)

// Config aggregates runtime options shared by CLI and GUI.
type Config struct {
    Host              string
    Port              int
    BaseURL           string
    ConversationURL   string
    SiderAPIToken     string
    AllowDummy        bool
    UseEnvToken       bool
    EnableUI          bool
    LogLevel          string
    ChatTimeout       time.Duration
    ConversationTimeout time.Duration
    CleanupInterval   time.Duration
    SessionMaxAge     time.Duration
    SiderSessionMaxAge time.Duration
    ContinuousCID     string
}

// Defaults returns baseline configuration.
func Defaults() Config {
    return Config{
        Host:               "0.0.0.0",
        Port:               4141,
        BaseURL:            "https://sider.ai/api/chat/v1/completions",
        ConversationURL:    "https://sider.ai/api/chat/v1/conversation/messages",
        AllowDummy:         true,
        UseEnvToken:        true,
        EnableUI:           true,
        LogLevel:           "info",
        ChatTimeout:        120 * time.Second,
        ConversationTimeout: 10 * time.Second,
        CleanupInterval:    15 * time.Minute,
        SessionMaxAge:      24 * time.Hour,
        SiderSessionMaxAge: 2 * time.Hour,
        ContinuousCID:      "continuous-conversation",
    }
}

// ApplyEnv overlays environment variables onto the config before flag parsing.
func (c *Config) ApplyEnv() {
    if v := os.Getenv("HOST"); v != "" {
        c.Host = v
    }
    if v := os.Getenv("PORT"); v != "" {
        if p, err := strconv.Atoi(v); err == nil {
            c.Port = p
        }
    }
    if v := os.Getenv("SIDER_API_TOKEN"); v != "" {
        c.SiderAPIToken = v
    }
    if v := os.Getenv("SIDER_BASE_URL"); v != "" {
        c.BaseURL = v
    }
    if v := os.Getenv("SIDER_CONVERSATION_URL"); v != "" {
        c.ConversationURL = v
    }
    if v := os.Getenv("ALLOW_DUMMY"); v != "" {
        c.AllowDummy = v == "1" || v == "true"
    }
    if v := os.Getenv("USE_ENV_TOKEN"); v != "" {
        c.UseEnvToken = v == "1" || v == "true"
    }
    if v := os.Getenv("ENABLE_UI"); v != "" {
        c.EnableUI = v == "1" || v == "true"
    }
    if v := os.Getenv("LOG_LEVEL"); v != "" {
        c.LogLevel = v
    }
    if v := os.Getenv("CHAT_TIMEOUT"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            c.ChatTimeout = d
        }
    }
    if v := os.Getenv("CONV_TIMEOUT"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            c.ConversationTimeout = d
        }
    }
    if v := os.Getenv("CLEANUP_INTERVAL"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            c.CleanupInterval = d
        }
    }
    if v := os.Getenv("SESSION_MAX_AGE"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            c.SessionMaxAge = d
        }
    }
    if v := os.Getenv("SIDER_SESSION_MAX_AGE"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            c.SiderSessionMaxAge = d
        }
    }
    if v := os.Getenv("CONTINUOUS_CID"); v != "" {
        c.ContinuousCID = v
    }
}

// Parse builds config from env + flags. Flags override env, which override defaults.
func Parse(args []string) (Config, error) {
    cfg := Defaults()

    // Auto-load .env if present
    if loaded, err := loadDotEnv(".env"); err != nil {
        return cfg, fmt.Errorf("load .env: %w", err)
    } else if !loaded {
        fmt.Fprintln(os.Stderr, "[sider2api] .env not found; using environment variables and flags")
    }

    cfg.ApplyEnv()

    fs := flag.NewFlagSet("sider2api", flag.ContinueOnError)

    fs.StringVar(&cfg.Host, "host", cfg.Host, "listen host")
    fs.IntVar(&cfg.Port, "port", cfg.Port, "listen port")
    fs.StringVar(&cfg.BaseURL, "base-url", cfg.BaseURL, "Sider chat completions endpoint")
    fs.StringVar(&cfg.ConversationURL, "conv-url", cfg.ConversationURL, "Sider conversation history endpoint")
    fs.StringVar(&cfg.SiderAPIToken, "token", cfg.SiderAPIToken, "Sider API token")
    fs.BoolVar(&cfg.AllowDummy, "allow-dummy", cfg.AllowDummy, "allow dummy token for testing")
    fs.BoolVar(&cfg.UseEnvToken, "use-env-token", cfg.UseEnvToken, "fallback to environment token when Authorization header missing")
    fs.BoolVar(&cfg.EnableUI, "ui", cfg.EnableUI, "serve embedded UI")
    fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "log level (debug,info,warn,error)")
    fs.DurationVar(&cfg.ChatTimeout, "chat-timeout", cfg.ChatTimeout, "Sider chat timeout")
    fs.DurationVar(&cfg.ConversationTimeout, "conv-timeout", cfg.ConversationTimeout, "Sider conversation history timeout")
    fs.DurationVar(&cfg.CleanupInterval, "cleanup-interval", cfg.CleanupInterval, "session cleanup interval")
    fs.DurationVar(&cfg.SessionMaxAge, "session-max-age", cfg.SessionMaxAge, "conversation session max age")
    fs.DurationVar(&cfg.SiderSessionMaxAge, "sider-session-max-age", cfg.SiderSessionMaxAge, "sider session max age")
    fs.StringVar(&cfg.ContinuousCID, "continuous-cid", cfg.ContinuousCID, "reserved CID for inferred continuous conversations")

    if err := fs.Parse(args); err != nil {
        // propagate flag errors to caller for CLI to display
        return cfg, fmt.Errorf("parse flags: %w", err)
    }

    return cfg, nil
}

// loadDotEnv loads KEY=VALUE pairs from a .env file into process env.
// Returns true if file was found and loaded, false if not present.
func loadDotEnv(path string) (bool, error) {
    f, err := os.Open(path)
    if err != nil {
        if os.IsNotExist(err) {
            return false, nil
        }
        return false, err
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])
        val = strings.Trim(val, `"'`)
        if key != "" {
            os.Setenv(key, val)
        }
    }
    if err := scanner.Err(); err != nil {
        return true, err
    }
    return true, nil
}
