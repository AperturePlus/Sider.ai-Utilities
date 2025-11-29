package server

import (
    "fmt"
    "log/slog"
    "net/http"
    "strings"
    "time"

    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"

    "sider2api/internal/config"
    "sider2api/internal/handlers"
    "sider2api/internal/session"
    "sider2api/internal/siderclient"
)

// Server wraps the Gin engine and dependencies.
type Server struct {
    Engine   *gin.Engine
    Handler  *handlers.Handler
    Sessions *session.SiderSessionManager
    Client   *siderclient.Client
}

// New constructs a configured Gin server with routes and middleware.
func New(cfg config.Config, logger *slog.Logger) *Server {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()

    r.Use(gin.Recovery())
    r.Use(gin.Logger())

    r.Use(cors.New(cors.Config{
        AllowAllOrigins:  true,
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Content-Type", "Authorization", "X-Conversation-ID", "X-Parent-Message-ID"},
        ExposeHeaders:    []string{"X-Conversation-ID", "X-Assistant-Message-ID", "X-User-Message-ID"},
        AllowCredentials: false,
        MaxAge:           12 * time.Hour,
    }))

    sessions := session.NewSiderSessionManager(cfg.SiderSessionMaxAge, cfg.ContinuousCID)
    client := siderclient.New(cfg.BaseURL, cfg.ConversationURL, cfg.ChatTimeout, cfg.ConversationTimeout, sessions)
    handler := handlers.New(cfg, client, sessions, logger)

    // public routes
    r.GET("/health", handler.Health)
    r.GET("/", handler.Root)
    r.GET("/api/test", handler.Test)

    // UI routes if enabled
    if handler.UIAvailable() {
        ui := r.Group("/ui")
        handler.ServeUI(ui)
    }

    // authenticated routes
    authGroup := r.Group("/")
    authGroup.Use(AuthMiddleware(cfg, logger))
    authGroup.POST("/v1/messages", handler.PostMessages)
    authGroup.POST("/v1/messages/count_tokens", handler.CountTokens)
    authGroup.POST("/v1/chat/completions", handler.PostChatCompletions)

    return &Server{Engine: r, Handler: handler, Sessions: sessions, Client: client}
}

// Run starts the HTTP server.
func (s *Server) Run(host string, port int) error {
    addr := fmt.Sprintf("%s:%d", host, port)
    return s.Engine.Run(addr)
}

// AuthMiddleware enforces Bearer auth, allowing env token or dummy token when configured.
func AuthMiddleware(cfg config.Config, logger *slog.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := ""
        authHeader := c.GetHeader("Authorization")
        if authHeader != "" {
            t, err := extractBearer(authHeader)
            if err != nil {
                c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "authentication_error", "message": err.Error()}})
                return
            }
            token = t
        }

        if token == "" && cfg.UseEnvToken && cfg.SiderAPIToken != "" {
            token = cfg.SiderAPIToken
        }

        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "authentication_error", "message": "Missing Authorization token"}})
            return
        }

        if !cfg.AllowDummy && token == "dummy" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "authentication_error", "message": "Dummy token not allowed"}})
            return
        }

        c.Set("authToken", token)
        c.Next()
    }
}

func extractBearer(header string) (string, error) {
    parts := strings.SplitN(header, " ", 2)
    if len(parts) != 2 {
        return "", fmt.Errorf("invalid Authorization header format")
    }
    if !strings.EqualFold(parts[0], "Bearer") {
        return "", fmt.Errorf("invalid Authorization header format")
    }
    token := strings.TrimSpace(parts[1])
    if token == "" {
        return "", fmt.Errorf("empty token in Authorization header")
    }
    return token, nil
}
