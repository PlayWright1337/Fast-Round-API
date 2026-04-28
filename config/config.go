package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	APIKey          string
	GinMode         string
	MaxBodyBytes    int64
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(getenv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(getenv("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err)
	}

	readTimeout, err := time.ParseDuration(getenv("READ_TIMEOUT", "5s"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid READ_TIMEOUT: %w", err)
	}

	writeTimeout, err := time.ParseDuration(getenv("WRITE_TIMEOUT", "5s"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid WRITE_TIMEOUT: %w", err)
	}

	idleTimeout, err := time.ParseDuration(getenv("IDLE_TIMEOUT", "60s"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid IDLE_TIMEOUT: %w", err)
	}

	maxBodyBytes, err := strconv.ParseInt(getenv("MAX_BODY_BYTES", "1048576"), 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("invalid MAX_BODY_BYTES: %w", err)
	}

	cfg := Config{
		Port:            getenv("PORT", "8080"),
		RedisAddr:       getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		RedisDB:         redisDB,
		APIKey:          os.Getenv("API_KEY"),
		GinMode:         getenv("GIN_MODE", "release"),
		MaxBodyBytes:    maxBodyBytes,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
