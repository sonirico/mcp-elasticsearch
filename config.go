package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Elasticsearch ElasticsearchConfig
	Server        ServerConfig
	Logging       LoggingConfig
}

type ElasticsearchConfig struct {
	URL      string
	APIKey   string
	Username string
	Password string
}

type ServerConfig struct {
	Name    string
	Version string
}

type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

func loadConfig() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Overload()

	config := &Config{
		Elasticsearch: ElasticsearchConfig{
			URL:      getEnv("ES_URL", "http://localhost:9200"),
			APIKey:   getEnv("ES_API_KEY", ""),
			Username: getEnv("ES_USERNAME", ""),
			Password: getEnv("ES_PASSWORD", ""),
		},
		Server: ServerConfig{
			Name:    getEnv("MCP_ES_SERVER_NAME", "mcp-elasticsearch 🔍"),
			Version: version,
		},
		Logging: LoggingConfig{
			Level:  getEnv("MCP_ES_LOG_LEVEL", "info"),
			Format: getEnv("MCP_ES_LOG_FORMAT", "console"),
			Output: getEnv("MCP_ES_LOG_OUTPUT", "stderr"),
		},
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func validateConfig(config *Config) error {
	if config.Elasticsearch.URL == "" {
		return fmt.Errorf("ES_URL environment variable is required")
	}

	// Either API key or username/password authentication must be provided
	if config.Elasticsearch.APIKey == "" &&
		(config.Elasticsearch.Username == "" || config.Elasticsearch.Password == "") {
		return fmt.Errorf("either ES_API_KEY or ES_USERNAME+ES_PASSWORD must be provided")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
