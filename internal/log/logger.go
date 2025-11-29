package log

import (
    "log/slog"
    "os"
    "strings"
)

// New constructs a slog.Logger with the requested level.
func New(level string) *slog.Logger {
    handlerOpts := &slog.HandlerOptions{Level: levelFromString(level)}
    handler := slog.NewTextHandler(os.Stdout, handlerOpts)
    return slog.New(handler)
}

func levelFromString(level string) slog.Level {
    switch strings.ToLower(level) {
    case "debug":
        return slog.LevelDebug
    case "warn", "warning":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
