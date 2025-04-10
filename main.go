package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	mrand "math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "embed"

	"github.com/conduitio/bwlimit"
	"github.com/dustin/go-humanize"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

var (
	//go:embed version
	version   string    // version is the version string of this web server app.
	startTime time.Time // startTime is the startup time of the current process.
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

		// NOTE: he default text log format in Go's slog doesn't display
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

// basicAuth wraps a handler with HTTP Basic Authentication and logs authentication failures.
func basicAuth(handler http.Handler, username, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the per-request logger and id.
		reqLogger, ok := r.Context().Value("logger").(*slog.Logger)
		if !ok {
			http.Error(w, "Internal server error: missing logger", http.StatusInternalServerError)
			return
		}
		id, ok := r.Context().Value("request_id").(string)
		if !ok {
			http.Error(w, "Internal server error: missing request ID", http.StatusInternalServerError)
			return
		}

		user, pass, hasAuth := r.BasicAuth()

		// Check if credentials were provided and are valid.
		if !hasAuth || user != username || pass != password {
			// Log the failed authentication attempt
			if !hasAuth {
				reqLogger.Warn("Authentication failed", "id", id,
					"reason", "no credentials provided")
			} else {
				reqLogger.Warn("Authentication failed", "id", id,
					"reason", "invalid usernamer and/or password", "user", user)
			}

			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// If we get here, credentials are valid, call the wrapped handler.
		handler.ServeHTTP(w, r)
	})
}

// reqDump is an HTTP middleware that dumps an incoming HTTP request on stdout
// and at the same time it echos it back to the client.
func reqDump() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			fmt.Printf("%s\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return

		}
		w.WriteHeader(http.StatusAlreadyReported)
		fmt.Fprintf(w, "%s", dump)
		fmt.Printf("%s\n", dump)
	})
}

// displayVer is an HTTP handler that is used on the `/_/version` path and
// which will return the version of this web server app.
func displayVer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		fmt.Fprintf(w, "Version: %s\n", version)
	})
}

// displayEnv is an HTTP handler that is used on the `/_/env` path and which
// will return all environment variables of the server process.
func displayEnv() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		fmt.Fprintln(w, "Environment Variables:")
		for _, v := range os.Environ() {
			fmt.Fprintf(w, "\t%s\n", v)
		}
	})
}

// displayStats is an HTTP handler that is used on the `/_/stats` path and which
// will returns Go runtime statistics about the current process.
func displayStats() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		currTime := time.Now()
		uptime := currTime.Sub(startTime)

		fmt.Fprintln(w, "Process Go runtime statistics:")
		fmt.Fprintf(w, "\tUptime: %s (current time: %s, process start time: %s)\n",
			uptime.String(), currTime.String(), startTime.String())

		metric := "/sched/gomaxprocs:threads"
		val, err := getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "\t%s = %s\n", metric, val)

		metric = "/sched/goroutines:goroutines"
		val, err = getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "\t%s = %s\n", metric, val)

		// percent := (cpu_sec / uptime_sec * goroutines) * 100
		metric = "/cpu/classes/user:cpu-seconds"
		val, err = getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "\t%s = %s\n", metric, val)

		metric = "/memory/classes/total:bytes"
		val, err = getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "\t%s = %s\n", metric, val)
	})
}

// displayLogs is an HTTP handler that is used on the `/_/logs` path and which
// will return all the logs of all the previous requests.
func displayLogs(logger *TeeLogHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		logger.Flush(w)
	})
}

// shouldCrash is an HTTP handler that will crash the server process (with a
// go runtime panic) if called with the appropriate query.
func shouldCrash() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		exitCode := 77

		query := r.URL.Query()
		if areYouSure, ok := query["areYouSure"]; !ok || len(areYouSure) != 1 || areYouSure[0] != "YesIAmSure" {
			http.Error(w, "I'm sorry, Dave. I'm afraid I can't do that.",
				http.StatusNotAcceptable)
			return
		}
		if ec, ok := query["exitCode"]; ok && len(ec) == 1 {
			x, err := strconv.Atoi(ec[0])
			if err != nil {
				http.Error(w, fmt.Sprintf("%s: invalid exit code", ec[0]),
					http.StatusBadRequest)
				return
			}
			exitCode = x
		}

		http.Error(w, "Dave, this conversation can serve no purpose anymore. Good-bye.",
			http.StatusInternalServerError)

		go func() {
			time.Sleep(2 * time.Second)
			os.Exit(exitCode)
		}()
	})
}

// allocMemoryHandler is an HTTP handler that will allocate memory if the
// appropriate query params are set.
func allocMemoryHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		query := r.URL.Query()
		sq, ok := query["size"]
		if !ok || len(sq) != 1 || len(sq[0]) == 0 {
			http.Error(w, "allocation size must be set",
				http.StatusBadRequest)
			return
		}
		s, err := humanize.ParseBytes(sq[0])
		if err != nil {
			http.Error(w, fmt.Sprintf("%s: invalid size", sq[0]),
				http.StatusBadRequest)
			return
		}
		size := s
		delay := 200 * time.Millisecond
		dq, ok := query["delay"]
		if ok && len(dq) == 1 || len(dq[0]) > 0 {
			d, err := time.ParseDuration(dq[0])
			if err != nil {
				http.Error(w, fmt.Sprintf("%s: invalid delay", dq[0]),
					http.StatusBadRequest)
				return
			}
			delay = d
		}

		allocMemory(size, delay)

		http.Error(w, "memory allocated", http.StatusCreated)
	})
}

// getRuntimeStat will retrive one of the supported runtime metrics (see
// https://pkg.go.dev/runtime/metrics#hdr-Supported_metrics).
//
// TODO: Currently pretty inefficient, change to a global map, etc.
func getRuntimeStat(name string) (string, error) {
	// Create a sample for the metric.
	sample := make([]metrics.Sample, 1)
	sample[0].Name = name

	// Sample the metric.
	metrics.Read(sample)

	v := sample[0].Value

	// Handle the result.
	switch v.Kind() {
	case metrics.KindUint64:
		return fmt.Sprintf("%d", v.Uint64()), nil
	case metrics.KindFloat64:
		return fmt.Sprintf("%f", v.Float64()), nil
	case metrics.KindFloat64Histogram:
		return "", fmt.Errorf("%s: histogram metric not currently supported", name)
	case metrics.KindBad:
		return "", fmt.Errorf("%s: metric no longer supported", name)
	default:
		return "", fmt.Errorf("%s: unexpected metric Kind: %v", name, sample[0].Value.Kind())
	}
}

// allocMemory will create a new slice of bytes of size `size`. It will then
// spawn a new gorouting that will periodically walk and update each byte to
// prevent the GC from freeing it. The `delay` is using during the walk, thus
// a smaller delay will cause a higher CPU utilization while a bigger one will
// decrease the CPU utilization.
func allocMemory(size uint64, delay time.Duration) {
	buff := make([]byte, size)

	for i := uint64(0); i < size; i++ {
		buff[i] = byte(mrand.Intn(256)) // Random byte between 0 and 255
	}

	go func() {
		for {
			time.Sleep(delay)
			for i := uint64(0); i < size; i++ {
				buff[i] = byte(mrand.Intn(256)) // Random byte between 0 and 255
				if i%100 == 0 {
					time.Sleep(delay)
				}
			}
		}
	}()
}

var (
	// sanitizeRE is a regexp that matches and can be used to remove common
	// problematic characters in user provided filenames:
	// - Control characters
	// - Path separators (/ and \)
	// - Characters illegal in Windows filenames (< > : " | ? *)
	// - Other potentially problematic chars ($ & ; = % ' ` ~ ! @ # ^ ( ) [ ] { } + ,)
	sanitizeRE = regexp.MustCompile(`[\\/:*?"<>|$&;=%'` + "`" + `~!@#^()\[\]{}\+,]`)

	// whitespaceRE matches on all spaces.
	whitespaceRE = regexp.MustCompile(`\s+`)
)

// sanitizeFilename removes potentially problematic characters from filenames
// and ensures the result is safe for filesystem operations.
func sanitizeFilename(filename string) string {
	sanitized := sanitizeRE.ReplaceAllString(filename, "_")
	sanitized = whitespaceRE.ReplaceAllString(sanitized, "_")

	// Remove leading/trailing periods, spaces, underscores.
	sanitized = strings.Trim(sanitized, "_ .")

	// Handle empty filename
	if sanitized == "" {
		return "unnamed_file"
	}

	// Extract any file extension.
	ext := ""
	lastDot := strings.LastIndex(sanitized, ".")
	if lastDot >= 0 {
		ext = sanitized[lastDot:]
		sanitized = sanitized[:lastDot]
	}

	// Maximum filename length on many filesystems is 255.
	maxBaseLength := 255 - len(ext)
	if len(sanitized) > maxBaseLength {
		sanitized = sanitized[:maxBaseLength]
	}

	return sanitized + ext
}

// uploadHandler is an HTTP middleware that accepts a multi-part file upload
// and saves the uploaded file locally. Not very useful for the file upload
// itself however it can be used to simulate traffic towards an edge-app instance
// (similar to if the edge-app instance would do a download). If `uploadPath`
// doesn't already exist it will be created, it can be e relative to the current
// directory where the server was started.
func uploadHandler(uploadPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only allow the POST method for uploads.
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the multipart form, 10 << 20 specifies a maximum upload of 10 MB.
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Could not parse multipart form", http.StatusBadRequest)
			return
		}

		// Get the file from the form data.
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Create uploads directory if it doesn't exist.
		uploadId := strings.ReplaceAll(quickID(12), "=", "_")
		uploadDir := filepath.Join(uploadPath, uploadId)
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			http.Error(w, "Error creating upload directory", http.StatusInternalServerError)
			return
		}

		// Create a new file.
		dst := filepath.Join(uploadDir, sanitizeFilename(handler.Filename))
		f, err := os.Create(dst)
		if err != nil {
			http.Error(w, "Error creating destination file", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		// Create a hash writer to calculate SHA256 while copying.
		hasher := sha256.New()
		multiWriter := io.MultiWriter(f, hasher)

		// Copy the uploaded file to the destination file and calculate hash simultaneously
		n, err := io.Copy(multiWriter, file)
		if err != nil {
			http.Error(w, "Error writing file", http.StatusInternalServerError)
			return
		}

		// Get the SHA256 hash as a hex string.
		hashSum := hasher.Sum(nil)
		hashString := hex.EncodeToString(hashSum)

		// Send success response.
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Successfully uploaded file '%s' as '%s' (%d bytes / %s). SHA256 checksum: %s",
			handler.Filename, dst, n, humanize.Bytes(uint64(n)), hashString)
	})
}

func main() {
	version = strings.TrimSpace(version)
	startTime = time.Now()

	// Define the CLI flags for the server listen address and static directory.
	listen := flag.String("listen", ":8080", "The address (`host:port`) on which the server should listen to. Default: `:8080`.")
	staticDir := flag.String("static", "./static", "The directory from which to serve static files.")
	bwLimitStr := flag.String("bw-limit", "2GB", "Limit the read and write bandwidth (each, not combined) of the entire server. Default: 2GB/s.")
	flag.Parse()

	// Setup a listener with a default BW limit of 2GB/s.
	limit := 2 * bwlimit.GB
	if len(*bwLimitStr) > 0 && *bwLimitStr != "0" {
		x, err := humanize.ParseBytes(*bwLimitStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid bandwidth limit '%s': %s\n", *bwLimitStr, err)
			flag.Usage()
			os.Exit(1)
		}
		limit = bwlimit.Byte(int(x))
	}
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	ln = bwlimit.NewListener(ln, limit, limit)

	// Set up the tee log handler.
	t := NewTeeLogHandler(tint.NewHandler(os.Stderr, &tint.Options{
		NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
		Level:      slog.LevelDebug,
		TimeFormat: time.DateTime,
	}))
	logger := slog.New(t)

	// Create a file server handler to serve static files.
	fs := http.FileServer(http.Dir(*staticDir))

	// Serve static files.
	http.Handle("/", loggingMidd(logger, fs))

	username := quickID(12)
	if username == quickIDNotRandom {
		fmt.Fprintf(os.Stderr, "Failed to generate a random username")
		os.Exit(1)
	}
	password := quickID(24)
	if password == quickIDNotRandom {
		fmt.Fprintf(os.Stderr, "Failed to generate a random password")
		os.Exit(1)
	}
	logger.Info("HTTP Basic authentication credentials", "username", username,
		"password", password)

	// Add HTTP handlers for the custom paths supported by the server.
	http.Handle("/_/version", loggingMidd(logger, displayVer()))
	http.Handle("/_/env", loggingMidd(logger, basicAuth(displayEnv(), username, password)))
	http.Handle("/_/logs", loggingMidd(logger, basicAuth(displayLogs(t), username, password)))
	http.Handle("/_/crash", loggingMidd(logger, basicAuth(shouldCrash(), username, password)))
	http.Handle("/_/alloc", loggingMidd(logger, basicAuth(allocMemoryHandler(), username, password)))
	http.Handle("/_/stats", loggingMidd(logger, displayStats()))
	http.Handle("/_/echo", loggingMidd(logger, reqDump()))
	http.Handle("/_/upload", loggingMidd(logger, basicAuth(uploadHandler(filepath.Join(*staticDir, "_", "uploads")), username, password)))

	// Start the server with the specified port.
	logger.Info("Starting server", "version", version, "address", *listen, "static_dir", *staticDir,
		"bandwidth_limit", humanize.Bytes(uint64(limit)))
	if err := http.Serve(ln, http.DefaultServeMux); err != nil {
		logger.Error("Server error", "error", err)
	}
}
