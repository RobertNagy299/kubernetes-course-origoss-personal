package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeCounter struct {
	mu    sync.Mutex
	count int64
	err   error
	calls int
}

func (f *fakeCounter) Increment(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++
	if f.err != nil {
		return 0, f.err
	}

	f.count++
	return f.count, nil
}

func TestHealthzDoesNotUseCounter(t *testing.T) {
	counter := &fakeCounter{
		err: errors.New("Redis should not be called"),
	}
	handler := NewHandler("hello", counter).Routes()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok")
	}

	if counter.calls != 0 {
		t.Fatalf("counter called %d times, want 0", counter.calls)
	}
}

func TestHomeIncrementsCounter(t *testing.T) {
	counter := &fakeCounter{}
	handler := NewHandler("hello from tests", counter).Routes()

	for wantVisit := int64(1); wantVisit <= 2; wantVisit++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("visit %d: status = %d, want %d",
				wantVisit, rec.Code, http.StatusOK)
		}

		body := rec.Body.String()
		expectedPrefix := fmt.Sprintf("hello from tests (visit %d, served by ", wantVisit)

		if !strings.HasPrefix(body, expectedPrefix) || !strings.HasSuffix(body, ")") {
			t.Fatalf("unexpected response body: %q", body)
		}
	}

	if counter.calls != 2 {
		t.Fatalf("counter called %d times, want 2", counter.calls)
	}
}

func TestHomeReturns503WhenCounterFails(t *testing.T) {
	counter := &fakeCounter{
		err: errors.New("Redis unavailable"),
	}
	handler := NewHandler("hello", counter).Routes()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d",
			rec.Code, http.StatusServiceUnavailable)
	}
}

func TestSlowRespondsAfterDelay(t *testing.T) {
	handler := NewHandler("hello", &fakeCounter{}).Routes()

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if elapsed < 3*time.Second {
		t.Fatalf("response took %v, want at least 3 seconds", elapsed)
	}
}
