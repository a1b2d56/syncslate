package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	DBPath        string
	DBMaxReaders  int
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	UploadDir     string
	MaxUploadSize int64
	RateLimitRPM  int
	LogLevel      string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		DBPath:        getEnv("DB_PATH", "./syncslate.db"),
		DBMaxReaders:  getEnvAsInt("DB_MAX_READERS", 4),
		JWTSecret:     getEnv("JWT_SECRET", "super-secret-syncslate-key-change-in-production-12345"),
		JWTAccessTTL:  getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL: getEnvAsDuration("JWT_REFRESH_TTL", 720*time.Hour), // 30 days
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadSize: getEnvAsInt64("MAX_UPLOAD_SIZE", 52428800), // 50MB
		RateLimitRPM:  getEnvAsInt("RATE_LIMIT_RPM", 100),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if valStr := os.Getenv(key); valStr != "" {
		if d, err := time.ParseDuration(valStr); err == nil {
			return d
		}
	}
	return defaultVal
}
