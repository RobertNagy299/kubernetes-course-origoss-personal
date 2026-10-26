package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// VisitCounter describes the Redis operation the HTTP layer needs.
// The Redis implementation will satisfy this interface.
type VisitCounter interface {
	Increment(ctx context.Context) (int64, error)
}

type Handler struct {
	greeting string
	counter  VisitCounter
	hostname string
}

func NewHandler(greeting string, counter VisitCounter) *Handler {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return &Handler{
		greeting: greeting,
		counter:  counter,
		hostname: hostname,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /slow", h.slow)

	return mux
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	visits, err := h.counter.Increment(r.Context())
	if err != nil {
		log.Printf("increment visit counter: %v", err)
		http.Error(w, "visit counter unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(
		w,
		"%s (visit %d, served by %s)",
		h.greeting,
		visits,
		h.hostname,
	)
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (h *Handler) slow(w http.ResponseWriter, r *http.Request) {
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		w.WriteHeader(http.StatusOK)
	case <-r.Context().Done():
		// The client disconnected or the request was canceled.
		return
	}
}
