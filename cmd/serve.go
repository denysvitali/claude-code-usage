package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/denysvitali/llm-usage/internal/serve"
	"github.com/spf13/cobra"
)

var (
	serveHost   string
	servePort   int
	serveWebDir string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long:  `Start an HTTP server that serves the web UI and provides a JSON API for usage statistics.`,
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "Host to bind to")
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "Port to listen on")
	serveCmd.Flags().StringVar(&serveWebDir, "web-dir", "", "Path to web directory (default: auto-detect)")

	rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	cfg := &serve.Config{
		Host:   serveHost,
		Port:   servePort,
		WebDir: serveWebDir,
	}

	// Auto-detect web directory if not specified
	if cfg.WebDir == "" {
		cfg.WebDir = serve.AutoDetectWebDir()
	}

	s := serve.NewServer(cfg)
	if err := s.Start(ctx); err != nil && err.Error() != "http: Server closed" {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
