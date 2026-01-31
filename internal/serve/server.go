// Package serve provides the HTTP server for the web UI and API
package serve

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
)

//go:embed web
var embeddedFS embed.FS

const (
	providerClaude = "claude"
	providerKimi   = "kimi"
	providerZAi    = "zai"
)

// Config holds the server configuration
type Config struct {
	Host   string
	Port   int
	WebDir string
}

// Server represents the HTTP server
type Server struct {
	config    *Config
	credsMgr  *credentials.Manager
	server    *http.Server
	providers []usage.ProviderInstance
}

// NewServer creates a new HTTP server
func NewServer(cfg *Config) *Server {
	mux := http.NewServeMux()

	s := &Server{
		config:   cfg,
		credsMgr: credentials.NewManager(),
		server: &http.Server{
			Addr:              cfg.Host + ":" + itoa(cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	// Register routes
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/v1/usage", s.handleUsage)
	mux.HandleFunc("GET /api/v1/providers", s.handleProviders)

	return s
}

// itoa converts int to string without importing strconv
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	n := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		b[n] = '-'
	}
	return string(b[n:])
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Load providers on start
	s.loadProviders()

	log.Printf("Starting server on http://%s:%d", s.config.Host, s.config.Port)

	// Shutdown on context cancellation
	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	return s.server.ListenAndServe()
}

// loadProviders loads all configured providers
func (s *Server) loadProviders() {
	s.providers = usage.GetProviders("", "", true, s.credsMgr)
}

// handleIndex serves the frontend HTML
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// First try to serve from disk (for development)
	if s.config.WebDir != "" {
		indexPath := filepath.Join(s.config.WebDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}
	}

	// Fall back to embedded filesystem
	webFS, err := fs.Sub(embeddedFS, "web")
	if err != nil {
		http.Error(w, "Failed to load embedded files", http.StatusInternalServerError)
		return
	}

	http.FileServer(http.FS(webFS)).ServeHTTP(w, r)
}

// handleUsage returns usage statistics for all providers
func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Parse query parameters
	providerFilter := r.URL.Query().Get("provider")
	accountFilter := r.URL.Query().Get("account")

	// Re-fetch providers on each request to get fresh data
	providers := s.providers
	if providerFilter != "" || accountFilter != "" {
		providers = usage.GetProviders(providerFilter, accountFilter, accountFilter == "", s.credsMgr)
	}

	stats := usage.FetchAllUsage(providers)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(stats); err != nil {
		http.Error(w, "Error encoding JSON: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleProviders returns list of available providers
func (s *Server) handleProviders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type ProviderInfo struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Accounts []string `json:"accounts"`
	}

	providerIDs := s.credsMgr.ListAvailable()
	providerList := make([]ProviderInfo, 0, len(providerIDs))

	for _, pid := range providerIDs {
		var accounts []string
		switch pid {
		case providerClaude:
			if creds, err := s.credsMgr.LoadClaude(); err == nil {
				accounts = creds.ListAccounts()
			}
			// Check for keychain credentials
			if _, _, err := usage.LoadClaudeFromKeychain(); err == nil {
				accounts = append(accounts, "default")
			}
		case providerKimi:
			if creds, err := s.credsMgr.LoadKimi(); err == nil {
				accounts = creds.ListAccounts()
			}
		case providerZAi:
			if creds, err := s.credsMgr.LoadZAi(); err == nil {
				accounts = creds.ListAccounts()
			}
		}

		name := providerName(pid)
		providerList = append(providerList, ProviderInfo{
			ID:       pid,
			Name:     name,
			Accounts: accounts,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(providerList)
}

func providerName(id string) string {
	switch id {
	case providerClaude:
		return "Claude"
	case providerKimi:
		return "Kimi"
	case providerZAi:
		return "Z.AI"
	default:
		return id
	}
}

// AutoDetectWebDir attempts to find the web directory automatically
func AutoDetectWebDir() string {
	// Try to find the web directory relative to the executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidatePaths := []string{
			filepath.Join(exeDir, "web"),
			filepath.Join(exeDir, "..", "web"),
			filepath.Join(exeDir, "..", "..", "web"),
		}
		for _, path := range candidatePaths {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Fall back to current directory
	if cwd, err := os.Getwd(); err == nil {
		candidatePath := filepath.Join(cwd, "web")
		if _, err := os.Stat(candidatePath); err == nil {
			return candidatePath
		}
	}

	// Last resort: use ./web
	return "./web"
}
