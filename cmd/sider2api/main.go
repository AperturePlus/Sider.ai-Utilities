package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "sider2api",
		Short:   "Sider2API - Bridge between Anthropic API and Sider",
		Long:    `A proxy server that converts Anthropic Claude API requests to Sider format.`,
		Version: version,
	}

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(chatCmd())
	rootCmd.AddCommand(tuiCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
