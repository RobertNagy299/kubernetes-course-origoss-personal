package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port     string
	Greeting string
	RedisURL string
}

func Load() (Config, error) {
	cfg := Config{
		Port:     getEnv("PORT", "8080"),
		Greeting: getEnv("GREETING", "hello from my container"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT %q: must be an integer between 1 and 65535", cfg.Port)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}
