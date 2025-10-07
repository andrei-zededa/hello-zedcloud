package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TeeLogHandler will handle logging to 2 different destinations:
//   - `next` (which is another `slog.Handler`, most likely one configured to write to `stdout`).
//   - And an in-memory string buffer which can then be written to any `io.Writer` separately.
type TeeLogHandler struct {
	me   slog.Handler
	mu   sync.Mutex
	buff *strings.Builder
	next slog.Handler
}

// Enabled just returns true since anyway the "internal" logger will write any
// message at any level to the buffer. The `next` logger might not actually
// process all messages. Enabled is used such that a `TeeLogHandler` will statisfy
// the `slog.Hander` interface.
func (t *TeeLogHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return true
}

// WithAttrs is used such that a `TeeLogHandler` will statisfy the `slog.Hander`
// interface. Currently this is NOT implemented and just returns a __copy__ of
// `t` without any additional attributes.
func (t *TeeLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return t
}

// WithGroup is used such that a `TeeLogHandler` will statisfy the `slog.Hander`
// interface. Currently this is NOT implemented and just returns a __copy__ of
// `t` without the new group name.
func (t *TeeLogHandler) WithGroup(name string) slog.Handler {
	return t
}

// Handle a log record.
func (t *TeeLogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Send the log record to the next handler.
	if err := t.next.Handle(ctx, r); err != nil {
		return fmt.Errorf("%w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Write to the in-memory buffer.
	if err := t.me.Handle(ctx, r); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// Flush will write the contents of the in-memory buffer with all the logs
// gathered so far to `w`. Subsequent calls to Flush will repeat all the
// previously written logs.
func (t *TeeLogHandler) Flush(w io.Writer) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := w.Write([]byte(t.buff.String())); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// NewTeeLogHandler creates and initializes a new TeeLogHandler.
func NewTeeLogHandler(handler slog.Handler) *TeeLogHandler {
	t := TeeLogHandler{
		buff: &strings.Builder{},
		next: handler,
	}
	// Create a new logger that will write to the buffer any message at any
	// logging level.
	t.me = slog.NewTextHandler(t.buff, &slog.HandlerOptions{Level: slog.LevelDebug})

	return &t
}

const quickIDNotRandom = "000000"

// quickID generate a small string random ID. Although unlikely the call to
// crypto/rand can fail and it such a case quickID just returns the fixed and
// obviously not random string of "000000". Since this is only used for logging
// in an *example* app ...
func quickID(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		// NOTE: Indeed returing this fixed string is not very useful.
		return quickIDNotRandom
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// getClientIP extracts the real client IP address from an HTTP request. It checks
// headers in the following priority order:
//  1. Fly-Client-IP (used by Fly.io proxy)
//  2. X-Real-IP (used by Nginx and others)
//  3. First entry in X-Forwarded-For (common proxy header)
//  4. Falls back to the RemoteAddr from the request
func getClientIP(r *http.Request) string {
	// Check for Fly-Client-IP header (Fly.io specific).
	if flyClientIP := r.Header.Get("Fly-Client-IP"); flyClientIP != "" {
		return flyClientIP
	}

	// Check for X-Real-IP header (commonly set by Nginx and others).
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Check for X-Forwarded-For header (common for proxies).
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For might contain multiple IPs (client, proxy1, proxy2, ...),
		// get the first one which is typically the client IP.
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			// Trim space that might be present after the comma.
			return strings.TrimSpace(ips[0])
		}
	}

	// Fall back to RemoteAddr if no headers are found. RemoteAddr is in
	// format "IP:port", so strip off the port if present.
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	// Remove IPv6 brackets if present.
	ip = strings.TrimPrefix(ip, "[")
	ip = strings.TrimSuffix(ip, "]")

	return ip
}

// loggingMidd is an HTTP middleware that logs each request and adds the logger and request ID to the context.
func loggingMidd(logger *slog.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := quickID(6)

		// NOTE: the default text log format in Go's slog doesn't display
		// all structured fields in the same way as the JSON format would.
		// The id is actually being captured and added to the logger as a
		// field, but it's not being displayed in the default text output
		// format. The default text formatter only shows a limited set of
		// attributes in the log line itself.
		reqLogger := logger.With("id", id)
		reqLogger.Info("Request received", "id", id, "method", r.Method,
			"url", r.URL.Path, "client_addr", getClientIP(r))

		// Add then logger and request ID to the context.
		ctx := context.WithValue(r.Context(), "logger", reqLogger)
		ctx = context.WithValue(ctx, "request_id", id)

		// Call the handler with the updated context.
		h.ServeHTTP(w, r.WithContext(ctx))

		dur := time.Since(start)
		reqLogger.Info("Request finished", "id", id, "duration", dur)
	})
}
