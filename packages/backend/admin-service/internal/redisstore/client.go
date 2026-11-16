// Package redisstore 提供领域端口的 Redis 实现，当前用于存放刷新令牌。
package redisstore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Connect 建立 Redis 连接并 Ping 确认可用。
func Connect(ctx context.Context, addr string, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}
