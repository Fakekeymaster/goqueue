package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration for the app.
type Config struct {
	RedisAddr	string
	RedisPass	string
	RedisDB		int
	APIPort		string
	WorkerCount	int
	MaxRetries	int
}

// Load reads config from environment variables.
//2nd arg is the default value if the env var is not found.
func Load() *Config {
    return &Config{
        RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
        RedisPass:   getEnv("REDIS_PASS", ""),
        RedisDB:     getEnvInt("REDIS_DB", 0),
        APIPort:     getEnv("API_PORT", "8080"),
        WorkerCount: getEnvInt("WORKER_COUNT", 5),
        MaxRetries:  getEnvInt("MAX_RETRIES", 3),
    }
}

// getEnv returns the env variable value or a fallback.
func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

// getEnvInt returns the env variable as int or a fallback.
func getEnvInt(key string, fallback int) int {
    if v := os.Getenv(key); v != "" {
        if i, err := strconv.Atoi(v); err == nil {
            return i
        }
    }
    return fallback
}

