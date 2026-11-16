// Package config 从环境变量读取服务配置，并给出本地开发默认值。
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 是服务运行所需的全部配置。
type Config struct {
	HTTPAddr       string
	GRPCAddr       string
	MongoURI       string
	MongoDatabase  string
	JWTSecret      string
	TokenTTL       time.Duration
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	RefreshTTL     time.Duration
	ServiceToken   string
	BCryptCost     int
	AllowedOrigins []string
}

// Load 读取环境变量并填充默认值。
func Load() Config {
	return Config{
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
		GRPCAddr:       env("GRPC_ADDR", ":9090"),
		MongoURI:       env("MONGO_URI", "mongodb://127.0.0.1:27017"),
		MongoDatabase:  env("MONGO_DATABASE", "unoadmin"),
		JWTSecret:      env("JWT_SECRET", "dev-only-change-me"),
		TokenTTL:       time.Duration(envInt("TOKEN_TTL_SECONDS", 3600)) * time.Second,
		RedisAddr:      env("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:  env("REDIS_PASSWORD", ""),
		RedisDB:        envInt("REDIS_DB", 0),
		RefreshTTL:     time.Duration(envInt("REFRESH_TTL_SECONDS", 7*24*3600)) * time.Second,
		ServiceToken:   env("GRPC_SERVICE_TOKEN", "dev-service-token"),
		BCryptCost:     envInt("BCRYPT_COST", bcryptDefaultCost),
		AllowedOrigins: splitComma(env("ALLOWED_ORIGINS", "http://localhost:5173")),
	}
}

// bcryptDefaultCost 与 bcrypt 默认代价保持一致，避免直接依赖加密库。
const bcryptDefaultCost = 10

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitComma(value string) []string {
	parts := []string{}
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
