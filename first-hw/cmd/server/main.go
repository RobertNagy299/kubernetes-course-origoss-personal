package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RobertNagy299/first-hw/internal/config"
	"github.com/RobertNagy299/first-hw/internal/redisstore"
	"github.com/RobertNagy299/first-hw/internal/web"
)

func main() {
	os.Exit(run())
}

func run() int {
	redisstore.ConfigureLogging()

	cfg, err := config.Load()
	if err != nil {
		writeJSONLog(map[string]any{
			"event": "startup_error",
			"error": err.Error(),
		})
		return 1
	}

	store, err := redisstore.New(cfg.RedisURL)
	if err != nil {
		writeJSONLog(map[string]any{
			"event": "redis_configuration_error",
			"error": err.Error(),
		})
		return 1
	}
	defer store.Close()

	// Administrative mode: reset the counter and exit without starting HTTP.
	if len(os.Args) == 2 && os.Args[1] == "reset" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := store.Reset(ctx); err != nil {
			writeJSONLog(map[string]any{
				"event": "counter_reset_error",
				"error": err.Error(),
			})
			return 1
		}

		writeJSONLog(map[string]any{
			"event":  "counter_reset",
			"status": "success",
		})
		return 0
	}

	if len(os.Args) > 1 {
		writeJSONLog(map[string]any{
			"event": "usage_error",
			"error": "the only supported argument is reset",
		})
		return 2
	}

	handler := web.NewHandler(cfg.Greeting, store)
	loggedHandler := web.NewLoggingMiddleware(handler.Routes())

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           loggedHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}
		serverErrors <- nil
	}()

	writeJSONLog(map[string]any{
		"event": "server_started",
		"port":  cfg.Port,
	})

	// Wait for either a termination signal or a server failure.
	signalCtx, stopSignals := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stopSignals()

	select {
	case err := <-serverErrors:
		if err != nil {
			writeJSONLog(map[string]any{
				"event": "server_error",
				"error": err.Error(),
			})
			return 1
		}
		return 0

	case <-signalCtx.Done():
		writeJSONLog(map[string]any{
			"event": "shutdown_started",
		})
	}

	// Stop accepting new connections and give active requests time to finish.
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		writeJSONLog(map[string]any{
			"event": "shutdown_error",
			"error": err.Error(),
		})

		// Force remaining connections closed if graceful shutdown times out.
		if closeErr := server.Close(); closeErr != nil {
			writeJSONLog(map[string]any{
				"event": "server_close_error",
				"error": closeErr.Error(),
			})
		}
		return 1
	}

	// ListenAndServe should now have returned ErrServerClosed.
	<-serverErrors

	writeJSONLog(map[string]any{
		"event":  "server_stopped",
		"status": "success",
	})
	return 0
}

func writeJSONLog(event map[string]any) {
	if err := json.NewEncoder(os.Stdout).Encode(event); err != nil {
		// Avoid printing non-JSON diagnostics to stderr.
		_, _ = fmt.Fprintln(os.Stdout, `{"event":"logging_error"}`)
	}
}
