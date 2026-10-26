package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddlewareWritesOneJSONEntry(t *testing.T) {
	var output bytes.Buffer

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})

	handler := newLoggingMiddleware(next, &output)

	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want exactly 1: %q",
			len(lines), output.String())
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}

	if entry["method"] != http.MethodPost {
		t.Errorf("method = %v, want %q", entry["method"], http.MethodPost)
	}

	if entry["path"] != "/items" {
		t.Errorf("path = %v, want %q", entry["path"], "/items")
	}

	if entry["status"] != float64(http.StatusCreated) {
		t.Errorf("status = %v, want %d", entry["status"], http.StatusCreated)
	}

	if _, ok := entry["duration_ms"]; !ok {
		t.Error("log entry is missing duration_ms")
	}
}
