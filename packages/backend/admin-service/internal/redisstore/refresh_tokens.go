package redisstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/unoadmin/admin-service/internal/domain"
)

// RefreshTokenStore 将刷新令牌存入 Redis，键带统一前缀并依赖 TTL 自动过期。
type RefreshTokenStore struct {
	client *redis.Client
	prefix string
}

// NewRefreshTokenStore 构造刷新令牌存储。
func NewRefreshTokenStore(client *redis.Client) *RefreshTokenStore {
	return &RefreshTokenStore{client: client, prefix: "unoadmin:refresh:"}
}

// Save 写入令牌与会话信息的映射，并设置过期时间。
func (s *RefreshTokenStore) Save(ctx context.Context, token string, session domain.RefreshSession, ttl time.Duration) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(token), data, ttl).Err()
}

// Get 读取令牌对应的会话信息；令牌不存在时返回 domain.ErrNotFound。
func (s *RefreshTokenStore) Get(ctx context.Context, token string) (*domain.RefreshSession, error) {
	raw, err := s.client.Get(ctx, s.key(token)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var session domain.RefreshSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Delete 删除令牌，用于退出登录与刷新轮换。
func (s *RefreshTokenStore) Delete(ctx context.Context, token string) error {
	return s.client.Del(ctx, s.key(token)).Err()
}

// key 为令牌加上命名空间前缀，避免与其它业务的 Redis 键冲突。
func (s *RefreshTokenStore) key(token string) string {
	return s.prefix + token
}
