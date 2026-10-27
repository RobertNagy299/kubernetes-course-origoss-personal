package redisstore

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJSONLoggerPrintfProducesValidJSON(t *testing.T) {
	var output bytes.Buffer
	logger := newJSONLogger(&output)

	logger.Printf(context.Background(), "connection failed: %s", "timeout")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	if entry["level"] != "error" {
		t.Errorf("level = %v, want error", entry["level"])
	}

	if entry["component"] != "go-redis" {
		t.Errorf("component = %v, want go-redis", entry["component"])
	}

	if entry["message"] != "connection failed: timeout" {
		t.Errorf("message = %v, want connection failed: timeout", entry["message"])
	}

	timestamp, ok := entry["time"].(string)
	if !ok {
		t.Fatalf("time field should be a string, got %T", entry["time"])
	}

	if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		t.Errorf("time field is not a valid RFC3339Nano timestamp: %v", err)
	}
}

func TestJSONLoggerPrintfWritesOneLinePerCall(t *testing.T) {
	var output bytes.Buffer
	logger := newJSONLogger(&output)

	logger.Printf(context.Background(), "first message")
	logger.Printf(context.Background(), "second message")

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2; output: %q", len(lines), output.String())
	}

	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
	}

	if !strings.Contains(lines[0], "first message") {
		t.Errorf("first line does not contain the first message: %s", lines[0])
	}

	if !strings.Contains(lines[1], "second message") {
		t.Errorf("second line does not contain the second message: %s", lines[1])
	}
}

func TestJSONLoggerPrintfConcurrentCalls(t *testing.T) {
	var output bytes.Buffer
	logger := newJSONLogger(&output)

	const calls = 100

	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			logger.Printf(context.Background(), "message %d", i)
		}(i)
	}

	wg.Wait()

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != calls {
		t.Fatalf("got %d log lines, want %d", len(lines), calls)
	}

	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
	}
}
