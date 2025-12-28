// cmd/glr/main.go
// GoLlama Runtime entry point
//
// LEARN: main.go should be minimal - just configuration and wiring.
// Business logic belongs in internal/ packages.

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/khaaliswooden-max/go_project/internal/audit"
	"github.com/khaaliswooden-max/go_project/internal/ollama"
	"github.com/khaaliswooden-max/go_project/internal/server"
)

func main() {
	// LEARN: Use flags for configuration that changes per deployment.
	// Use environment variables for secrets.
	// Use config files for complex configuration.
	
	var (
		addr       = flag.String("addr", envOrDefault("GLR_ADDR", ":8080"), "Server listen address")
		ollamaURL  = flag.String("ollama-url", envOrDefault("OLLAMA_URL", "http://localhost:11434"), "Ollama API URL")
		timeout    = flag.Duration("timeout", 30*time.Second, "Request timeout")
		logLevel   = flag.String("log-level", envOrDefault("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
		auditFile  = flag.String("audit-file", envOrDefault("AUDIT_FILE", ""), "Audit log file (empty = stdout)")
	)
	flag.Parse()

	// Setup structured logging
	// LEARN: slog is the standard structured logger as of Go 1.21.
	// It replaces the older log package for new code.
	level := parseLogLevel(*logLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	// Create Ollama client
	ollamaClient, err := ollama.New(ollama.Config{
		BaseURL: *ollamaURL,
		Timeout: *timeout,
	})
	if err != nil {
		logger.Error("failed to create ollama client", "error", err)
		os.Exit(1)
	}

	// Create audit logger
	var auditLogger *audit.Logger
	if *auditFile != "" {
		fileLogger, err := audit.NewFileLogger(*auditFile)
		if err != nil {
			logger.Error("failed to create audit logger", "error", err, "file", *auditFile)
			os.Exit(1)
		}
		defer fileLogger.Close()
		auditLogger = fileLogger.Logger
	} else {
		auditLogger = audit.NewDefault()
	}

	// Create and run server
	srv := server.New(
		server.Config{
			Addr:            *addr,
			ReadTimeout:     *timeout,
			WriteTimeout:    *timeout * 2, // Allow longer for generation
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
		ollamaClient,
		auditLogger,
		logger,
	)

	// Log startup configuration
	logger.Info("starting glr",
		"addr", *addr,
		"ollama_url", *ollamaURL,
		"timeout", timeout.String(),
		"log_level", *logLevel,
	)

	// Run server (blocks until shutdown)
	if err := srv.Run(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// envOrDefault returns the environment variable value or a default.
//
// LEARN: This pattern allows configuration via env vars or flags.
// Flags take precedence (they override env var defaults).
func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// parseLogLevel converts string to slog.Level.
func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// usage prints usage information.
func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GoLlama Runtime (GLR) - LLM Tool Orchestration Server\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  GLR_ADDR      Server listen address (default: :8080)\n")
		fmt.Fprintf(os.Stderr, "  OLLAMA_URL    Ollama API URL (default: http://localhost:11434)\n")
		fmt.Fprintf(os.Stderr, "  LOG_LEVEL     Log level: debug, info, warn, error (default: info)\n")
		fmt.Fprintf(os.Stderr, "  AUDIT_FILE    Audit log file path (default: stdout)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                           # Start with defaults\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -addr :9090               # Custom port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -ollama-url http://gpu:11434  # Remote Ollama\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  OLLAMA_URL=http://gpu:11434 %s   # Via env var\n", os.Args[0])
	}
}
