package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Database    DatabaseConfig
	TigerBeetle TigerBeetleConfig
	Redis       RedisConfig
	Server      ServerConfig
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	URL      string
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
}

// TigerBeetleConfig holds TigerBeetle configuration.
type TigerBeetleConfig struct {
	ClusterID uint64
	Addresses []string
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	URL string
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port int
	Env  string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{}

	// Database
	cfg.Database.URL = getEnv("DATABASE_URL", "postgresql://kovra:kovra_dev@localhost:5432/kovra?sslmode=disable")
	cfg.Database.MaxConns = int32(getEnvInt("DATABASE_MAX_CONNS", 25))

	// TigerBeetle
	cfg.TigerBeetle.ClusterID = uint64(getEnvInt("TB_CLUSTER_ID", 0))
	addresses := getEnv("TB_ADDRESSES", "3000")
	cfg.TigerBeetle.Addresses = parseAddresses(addresses)

	// Redis
	cfg.Redis.URL = getEnv("REDIS_URL", "redis://localhost:6379")

	// Server
	cfg.Server.Port = getEnvInt("API_PORT", 8080)
	cfg.Server.Env = getEnv("ENV", "development")

	return cfg, nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// parseAddresses parses comma-separated TigerBeetle addresses.
// Accepts either port numbers (3000,3001,3002) or full addresses (127.0.0.1:3000).
func parseAddresses(s string) []string {
	parts := strings.Split(s, ",")
	addresses := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// If it's just a port number, prepend localhost
		if !strings.Contains(p, ":") {
			p = fmt.Sprintf("127.0.0.1:%s", p)
		}
		addresses = append(addresses, p)
	}
	return addresses
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
