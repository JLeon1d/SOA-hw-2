package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := GetRequestID(r.Context())

		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			written:        false,
		}

		var userID *string
		if claims := GetClaims(r.Context()); claims != nil {
			userID = &claims.UserID
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		logEvent := log.Info().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("endpoint", r.URL.Path).
			Int("status_code", wrapped.statusCode).
			Int64("duration_ms", duration.Milliseconds()).
			Str("timestamp", time.Now().UTC().Format(time.RFC3339))

		if userID != nil {
			logEvent = logEvent.Str("user_id", *userID)
		} else {
			logEvent = logEvent.Str("user_id", "null")
		}

		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
			logEvent = logEvent.Str("request_type", "mutating")
		}

		logEvent.Msg("HTTP request")
	})
}

func SetupLogger(level string) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.RFC3339
}
