package common

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr  string
	GRPCAddr  string
	BaseURL   string
	Postgres  PostgresConfig
	Redis     RedisConfig
	NATS      NATSConfig
	ClickHouse ClickHouseConfig
}

type PostgresConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	TTL      time.Duration
}

type NATSConfig struct {
	URL string
}

type ClickHouseConfig struct {
	Addr     string
	Database string
	Username string
	Password string
}

func LoadConfig() Config {
	return Config{
		HTTPAddr: envOrDefault("HTTP_ADDR", ":8080"),
		GRPCAddr: envOrDefault("GRPC_ADDR", ":50051"),
		BaseURL:  envOrDefault("BASE_URL", "http://localhost:8080"),
		Postgres: PostgresConfig{
			DSN: envOrDefault("POSTGRES_DSN", "postgres://snip:snip@localhost:5432/snip?sslmode=disable"),
		},
		Redis: RedisConfig{
			Addr:     envOrDefault("REDIS_ADDR", "localhost:6379"),
			Password: envOrDefault("REDIS_PASSWORD", ""),
			DB:       envOrDefaultInt("REDIS_DB", 0),
			TTL:      time.Hour,
		},
		NATS: NATSConfig{
			URL: envOrDefault("NATS_URL", "nats://localhost:4222"),
		},
		ClickHouse: ClickHouseConfig{
			Addr:     envOrDefault("CLICKHOUSE_ADDR", "localhost:9000"),
			Database: envOrDefault("CLICKHOUSE_DB", "snip"),
			Username: envOrDefault("CLICKHOUSE_USER", "default"),
			Password: envOrDefault("CLICKHOUSE_PASSWORD", ""),
		},
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
