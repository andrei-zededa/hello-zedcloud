package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"runtime/metrics"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

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
		_, _ = fmt.Fprintf(w, "%s", dump)
		fmt.Printf("%s\n", dump)
	})
}

// displayVer is an HTTP handler that is used on the `/_/version` path and
// which will return the version of this web server app.
func displayVer(version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		_, _ = fmt.Fprintf(w, "Version: %s\n", version)
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

		_, _ = fmt.Fprintln(w, "Environment Variables:")
		for _, v := range os.Environ() {
			_, _ = fmt.Fprintf(w, "\t%s\n", v)
		}
	})
}

// displayStats is an HTTP handler that is used on the `/_/stats` path and which
// will returns Go runtime statistics about the current process.
func displayStats(startTime time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, fmt.Sprintf("method %s not implemented for this path", r.Method),
				http.StatusNotImplemented)
			return
		}

		currTime := time.Now()
		uptime := currTime.Sub(startTime)

		_, _ = fmt.Fprintln(w, "Process Go runtime statistics:")
		_, _ = fmt.Fprintf(w, "\tUptime: %s (current time: %s, process start time: %s)\n",
			uptime.String(), currTime.String(), startTime.String())

		metric := "/sched/gomaxprocs:threads"
		val, err := getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprintf(w, "\t%s = %s\n", metric, val)

		metric = "/sched/goroutines:goroutines"
		val, err = getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprintf(w, "\t%s = %s\n", metric, val)

		// percent := (cpu_sec / uptime_sec * goroutines) * 100
		metric = "/cpu/classes/user:cpu-seconds"
		val, err = getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprintf(w, "\t%s = %s\n", metric, val)

		metric = "/memory/classes/total:bytes"
		val, err = getRuntimeStat(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprintf(w, "\t%s = %s\n", metric, val)
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

		_ = logger.Flush(w)
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
		defer func() { _ = file.Close() }()

		// Create uploads directory if it doesn't exist.
		uploadID := strings.ReplaceAll(quickID(12), "=", "_")
		uploadDir := filepath.Join(uploadPath, uploadID)
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
		defer func() { _ = f.Close() }()

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
		_, _ = fmt.Fprintf(w, "Successfully uploaded file '%s' as '%s' (%d bytes / %s). SHA256 checksum: %s",
			handler.Filename, dst, n, humanize.Bytes(uint64(n)), hashString)
	})
}
