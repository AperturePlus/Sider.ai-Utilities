//go:build gui
// +build gui

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"sider2api/internal/config"
	"sider2api/internal/server"
)

func runWithGUI(cfg config.Config, logger *slog.Logger) error {
	gui := app.NewWithID("sider2api.gui")
	w := gui.NewWindow("Sider2API Server")
	w.Resize(fyne.NewSize(520, 380))

	// Create form fields
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

	statusLabel := widget.NewLabel("Server stopped")

	var httpSrv *http.Server
	var srv *server.Server

	// Auto-start server
	startServer := func() {
		if httpSrv != nil {
			statusLabel.SetText("Server already running")
			return
		}

		srv = server.New(cfg, logger)
		httpSrv = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler: srv.Engine,
		}

		go func() {
			statusLabel.SetText(fmt.Sprintf("✓ Running on %s:%d", cfg.Host, cfg.Port))
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				statusLabel.SetText("✗ Server error: " + err.Error())
				httpSrv = nil
			}
		}()
	}

	stopServer := func() {
		if httpSrv == nil {
			statusLabel.SetText("Server not running")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(ctx); err != nil {
			statusLabel.SetText("✗ Shutdown error: " + err.Error())
		} else {
			statusLabel.SetText("Server stopped")
		}
		httpSrv = nil
	}

	startBtn := widget.NewButton("Start Server", startServer)
	stopBtn := widget.NewButton("Stop Server", stopServer)

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

	// Setup system tray if supported
	if desk, ok := gui.(desktop.App); ok {
		menu := fyne.NewMenu("Sider2API",
			fyne.NewMenuItem("Show", func() {
				w.Show()
			}),
			fyne.NewMenuItem("Start Server", startServer),
			fyne.NewMenuItem("Stop Server", stopServer),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				stopServer()
				gui.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)

		// Minimize to tray instead of closing
		w.SetCloseIntercept(func() {
			w.Hide()
		})
	}

	// Auto-start server on launch
	startServer()

	w.ShowAndRun()
	stopServer()

	return nil
}
