//go:build !gui
// +build !gui

package main

import (
	"fmt"
	"log/slog"

	"sider2api/internal/config"
)

func runWithGUI(cfg config.Config, logger *slog.Logger) error {
	return fmt.Errorf("GUI support not compiled in. Rebuild with -tags=gui")
}
