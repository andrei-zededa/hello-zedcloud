package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	"github.com/conduitio/bwlimit"
	"github.com/dustin/go-humanize"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

const defaultVersion = "0.0.0-dev"

// Config holds the configuration for the server.
type Config struct {
	// Listen is the address (host:port) on which the server should listen.
	Listen string

	// StaticDir is the directory from which to serve static files.
	StaticDir string

	// BwLimit limits the read and write bandwidth (each, not combined) of
	// the entire server. A string like `2m, 2mb, 2M or 2MB`, all meaning
	// 2 megabytes per second.
	BwLimit string

	// Username for HTTP basic authentication. Empty string disables authentication.
	Username string

	// Password for HTTP basic authentication.
	Password string

	// Version is the version string of this web server app.
	Version string
}

// Server represents the web server instance.
type Server struct {
	config    Config
	limit     bwlimit.Byte
	logger    *slog.Logger
	teeLogger *TeeLogHandler
	httpSrv   *http.Server
	listener  net.Listener
	startTime time.Time
}

// New creates a new Server instance with the given configuration.
func New(config Config) (*Server, error) {
	config.Version = strings.TrimSpace(config.Version)
	if len(config.Version) == 0 {
		// Set default version if not provided.
		config.Version = strings.TrimSpace(defaultVersion)
	}

	// Validate configuration.
	if config.Listen == "" {
		return nil, fmt.Errorf("listen address cannot be empty")
	}
	if config.StaticDir == "" {
		return nil, fmt.Errorf("static directory cannot be empty")
	}

	// Parse the bandwidth limit.
	limit := 2 * bwlimit.GB
	if len(config.BwLimit) > 0 && config.BwLimit != "0" {
		x, err := humanize.ParseBytes(config.BwLimit)
		if err != nil {
			return nil, fmt.Errorf("invalid bandwidth limit '%s': %w", config.BwLimit, err)
		}
		limit = bwlimit.Byte(int(x))
	}

	s := &Server{
		config: config,
		limit:  limit,
	}

	return s, nil
}

// Serve initializes and starts the server. Will block if the server starts
// successfully.
func (s *Server) Serve() error {
	s.startTime = time.Now()
	// Create listener with bandwidth limiting.
	ln, err := net.Listen("tcp", s.config.Listen)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = bwlimit.NewListener(ln, s.limit, s.limit)

	// Set up the tee log handler.
	s.teeLogger = NewTeeLogHandler(tint.NewHandler(os.Stderr, &tint.Options{
		NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
		Level:      slog.LevelDebug,
		TimeFormat: time.DateTime,
	}))
	s.logger = slog.New(s.teeLogger)

	// Create a new ServeMux for this server instance.
	mux := http.NewServeMux()

	// Create a file server handler to serve static files.
	fs := http.FileServer(http.Dir(s.config.StaticDir))

	// Serve static files.
	mux.Handle("/", loggingMidd(s.logger, fs))

	// Add HTTP handlers for the custom paths supported by the server.
	mux.Handle("/_/version", loggingMidd(s.logger, displayVer(s.config.Version)))
	mux.Handle("/_/stats", loggingMidd(s.logger, displayStats(s.startTime)))
	mux.Handle("/_/echo", loggingMidd(s.logger, reqDump()))

	// Configure authenticated endpoints if credentials are provided.
	if len(s.config.Username) > 0 {
		username := s.config.Username
		password := s.config.Password

		s.logger.Info("HTTP Basic authentication credentials", "username", username,
			"password", password)

		mux.Handle("/_/env", loggingMidd(s.logger, basicAuth(displayEnv(), username, password)))
		mux.Handle("/_/logs", loggingMidd(s.logger, basicAuth(displayLogs(s.teeLogger), username, password)))
		mux.Handle("/_/crash", loggingMidd(s.logger, basicAuth(shouldCrash(), username, password)))
		mux.Handle("/_/alloc", loggingMidd(s.logger, basicAuth(allocMemoryHandler(), username, password)))
		mux.Handle("/_/upload", loggingMidd(s.logger, basicAuth(uploadHandler(filepath.Join(s.config.StaticDir, "_", "uploads")), username, password)))
	} else {
		mux.Handle("/_/env", loggingMidd(s.logger, displayEnv()))
		mux.Handle("/_/logs", loggingMidd(s.logger, displayLogs(s.teeLogger)))
		mux.Handle("/_/crash", loggingMidd(s.logger, shouldCrash()))
		mux.Handle("/_/alloc", loggingMidd(s.logger, allocMemoryHandler()))
		mux.Handle("/_/upload", loggingMidd(s.logger, uploadHandler(filepath.Join(s.config.StaticDir, "_", "uploads"))))
	}

	// Create HTTP server.
	s.httpSrv = &http.Server{
		Handler: mux,
	}

	// Log startup message.
	s.logger.Info("Starting server", "version", s.config.Version, "address", s.config.Listen,
		"static_dir", s.config.StaticDir, "bandwidth_limit", humanize.Bytes(uint64(s.limit)))

	// Start serving (blocking call).
	if err := s.httpSrv.Serve(s.listener); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Server error", "error", err)
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

// Logger returns the server's logger instance.
func (s *Server) Logger() *slog.Logger {
	return s.logger
}
