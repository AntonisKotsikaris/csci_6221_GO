package config

import (
	"log"
	"os"
	"strconv"
)

/*
ServerConfig holds configuration for the GoLlama server
*/
type ServerConfig struct {
	Port              int
	QueueSize         int
	ConcurrentWorkers int
	MaxRetries        int
	DefaultMaxTokens  int
}

/*
LoadServerConfig loads server configuration from environment variables with defaults
*/
func LoadServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:              getEnvInt("GOLLAMA_PORT", 9000),
		QueueSize:         getEnvInt("QUEUE_SIZE", 5000),
		ConcurrentWorkers: getEnvInt("CONCURRENT_WORKERS", 10),
		MaxRetries:        getEnvInt("MAX_RETRIES", 3),
		DefaultMaxTokens:  getEnvInt("DEFAULT_MAX_TOKENS", 100),
	}
}

/*
getEnvInt retrieves an integer from environment variables or returns default
*/
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid value for %s, using default %d", key, defaultValue)
	}
	return defaultValue
}
