package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"sider2api/internal/config"
	appLog "sider2api/internal/log"
	"sider2api/internal/server"
)

func serveCmd() *cobra.Command {
	var withGUI bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the API server",
		Long:  `Start the Sider2API proxy server. Use --gui flag on Windows to show system tray icon.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Parse(os.Args[1:])
			if err != nil {
				return fmt.Errorf("config error: %w", err)
			}

			logger := appLog.New(cfg.LogLevel)

			// If GUI requested, delegate to GUI mode
			if withGUI {
				return runWithGUI(cfg, logger)
			}

			// Otherwise run headless server
			return runHeadlessServer(cfg, logger)
		},
	}

	cmd.Flags().BoolVar(&withGUI, "gui", false, "Show GUI window (Windows only)")

	return cmd
}

func runHeadlessServer(cfg config.Config, logger *slog.Logger) error {
	srv := server.New(cfg, logger)

	// background cleanup for sessions
	go func() {
		ticker := time.NewTicker(cfg.CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			cleaned := srv.Sessions.Cleanup()
			if cleaned > 0 {
				logger.Info("cleaned expired Sider sessions", "count", cleaned)
			}
		}
	}()

	// graceful shutdown notifier
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		s := <-sigCh
		logger.Info("received signal, exiting", "signal", s.String())
		os.Exit(0)
	}()

	logger.Info("starting Sider2API server", "host", cfg.Host, "port", cfg.Port)
	if err := srv.Run(cfg.Host, cfg.Port); err != nil {
		logger.Error("server exited with error", "error", err)
		return err
	}

	return nil
}
