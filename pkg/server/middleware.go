package server

import (
	"log/slog"
	"net/http"
)

// basicAuth wraps a handler with HTTP Basic Authentication and logs authentication failures.
func basicAuth(handler http.Handler, username, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the per-request logger and id.
		reqLogger, ok := r.Context().Value(loggerKey).(*slog.Logger)
		if !ok {
			http.Error(w, "Internal server error: missing logger", http.StatusInternalServerError)
			return
		}
		id, ok := r.Context().Value(requestIDKey).(string)
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
