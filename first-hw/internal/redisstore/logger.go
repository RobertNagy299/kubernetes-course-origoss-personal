package redisstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type jsonLogger struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func (l *jsonLogger) Printf(
	ctx context.Context,
	format string,
	v ...interface{},
) {
	_ = ctx // go-redis supplies a context; it isn't needed for this log entry.

	entry := map[string]any{
		"time":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":     "error",
		"component": "go-redis",
		"message":   fmt.Sprintf(format, v...),
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	_ = l.encoder.Encode(entry)
}

// ConfigureLogging redirects go-redis's internal logs to JSON on stdout.
// Call this before creating or using a Redis client.
func ConfigureLogging() {
	redis.SetLogger(&jsonLogger{
		encoder: json.NewEncoder(os.Stdout),
	})
}
