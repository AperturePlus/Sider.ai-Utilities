package handlers

import (
    "log/slog"

    "sider2api/internal/config"
    "sider2api/internal/session"
    "sider2api/internal/siderclient"
)

// Handler aggregates dependencies used by HTTP handlers.
type Handler struct {
    Config   config.Config
    Client   *siderclient.Client
    Sessions *session.SiderSessionManager
    Logger   *slog.Logger
}

func New(cfg config.Config, client *siderclient.Client, sessions *session.SiderSessionManager, logger *slog.Logger) *Handler {
    return &Handler{Config: cfg, Client: client, Sessions: sessions, Logger: logger}
}
