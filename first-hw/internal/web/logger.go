package web

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(p)
}

type requestLog struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Status   int    `json:"status"`
	Duration int64  `json:"duration_ms"`
	Error    string `json:"error,omitempty"`
}

type requestLogger struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func NewLoggingMiddleware(next http.Handler) http.Handler {
	logger := &requestLogger{
		encoder: json.NewEncoder(os.Stdout),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(sw, r)

		entry := requestLog{
			Method:   r.Method,
			Path:     r.URL.Path,
			Status:   sw.status,
			Duration: time.Since(start).Milliseconds(),
		}

		logger.mu.Lock()
		_ = logger.encoder.Encode(entry)
		logger.mu.Unlock()
	})
}
