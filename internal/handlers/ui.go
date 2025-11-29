package handlers

import (
    "bytes"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    uiassets "sider2api/ui"
)

func (h *Handler) UIIndex(c *gin.Context) {
    html := uiassets.ChatHTML
    token := h.Config.SiderAPIToken
    if token == "" {
        token = ""
    }
    rendered := bytes.ReplaceAll(html, []byte("{{API_TOKEN}}"), []byte(token))
    c.Data(http.StatusOK, "text/html; charset=utf-8", rendered)
}

func (h *Handler) UIStyles(c *gin.Context) {
    c.Data(http.StatusOK, "text/css; charset=utf-8", uiassets.StylesCSS)
}

func (h *Handler) UIScript(c *gin.Context) {
    // ensure application/javascript content type
    c.Data(http.StatusOK, "application/javascript; charset=utf-8", uiassets.AppJS)
}

// ServeUI is a helper to register UI routes when enabled.
func (h *Handler) ServeUI(r *gin.RouterGroup) {
    r.GET("/", h.UIIndex)
    r.GET("/styles.css", h.UIStyles)
    r.GET("/app.js", h.UIScript)
}

// UIAvailable returns false when UI disabled via config.
func (h *Handler) UIAvailable() bool {
    return h.Config.EnableUI && strings.TrimSpace(string(uiassets.ChatHTML)) != ""
}
