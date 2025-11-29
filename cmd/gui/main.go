//go:build gui
// +build gui

package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "time"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"

    "sider2api/internal/config"
    appLog "sider2api/internal/log"
    "sider2api/internal/server"
)

func main() {
    cfg := config.Defaults()
    cfg.ApplyEnv()

    logger := appLog.New(cfg.LogLevel)

    gui := app.NewWithID("sider2api.gui")
    w := gui.NewWindow("Sider2API (Go) - GUI")
    w.Resize(fyne.NewSize(520, 380))

    hostEntry := widget.NewEntry()
    hostEntry.SetText(cfg.Host)

    portEntry := widget.NewEntry()
    portEntry.SetText(fmt.Sprintf("%d", cfg.Port))

    tokenEntry := widget.NewPasswordEntry()
    tokenEntry.SetText(cfg.SiderAPIToken)

    baseURLEntry := widget.NewEntry()
    baseURLEntry.SetText(cfg.BaseURL)

    allowDummy := widget.NewCheck("Allow dummy token", nil)
    allowDummy.SetChecked(cfg.AllowDummy)

    useEnv := widget.NewCheck("Use env token when header missing", nil)
    useEnv.SetChecked(cfg.UseEnvToken)

    enableUI := widget.NewCheck("Serve UI", nil)
    enableUI.SetChecked(cfg.EnableUI)

    statusLabel := widget.NewLabel("Idle")

    var httpSrv *http.Server

    startBtn := widget.NewButton("Start Server", func() {
        if httpSrv != nil {
            statusLabel.SetText("Server already running")
            return
        }

        portVal, err := strconv.Atoi(portEntry.Text)
        if err != nil {
            statusLabel.SetText("Invalid port")
            return
        }

        cfg.Host = hostEntry.Text
        cfg.Port = portVal
        cfg.SiderAPIToken = tokenEntry.Text
        cfg.BaseURL = baseURLEntry.Text
        cfg.AllowDummy = allowDummy.Checked
        cfg.UseEnvToken = useEnv.Checked
        cfg.EnableUI = enableUI.Checked

        srv := server.New(cfg, logger)

        httpSrv = &http.Server{Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), Handler: srv.Engine}

        go func() {
            statusLabel.SetText(fmt.Sprintf("Running on %s:%d", cfg.Host, cfg.Port))
            if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                statusLabel.SetText("Server error: " + err.Error())
                httpSrv = nil
            }
        }()
    })

    stopBtn := widget.NewButton("Stop Server", func() {
        if httpSrv == nil {
            statusLabel.SetText("Server not running")
            return
        }
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := httpSrv.Shutdown(ctx); err != nil {
            statusLabel.SetText("Shutdown error: " + err.Error())
        } else {
            statusLabel.SetText("Stopped")
        }
        httpSrv = nil
    })

    form := widget.NewForm(
        widget.NewFormItem("Host", hostEntry),
        widget.NewFormItem("Port", portEntry),
        widget.NewFormItem("Token", tokenEntry),
        widget.NewFormItem("Base URL", baseURLEntry),
    )

    checks := container.NewVBox(allowDummy, useEnv, enableUI)
    buttons := container.NewHBox(startBtn, stopBtn)

    w.SetContent(container.NewVBox(
        form,
        checks,
        buttons,
        statusLabel,
    ))

    w.ShowAndRun()
    _ = logger // keep logger referenced to avoid GC before windows close
    _ = os.Environ()
}
