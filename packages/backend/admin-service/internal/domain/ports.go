package domain

import (
	"context"
	"time"
)

// UserRepository 持久化用户，并提供关联角色的查询能力。
type UserRepository interface {
	List(ctx context.Context, query ListQuery) (PageResult[User], error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	CountByRoles(ctx context.Context, roleIDs []string) (map[string]int, error)
}

// RoleRepository 持久化角色，UserCount 在查询时聚合得到。
type RoleRepository interface {
	List(ctx context.Context, query ListQuery) (PageResult[Role], error)
	GetByIDs(ctx context.Context, ids []string) ([]Role, error)
}

// PermissionRepository 返回扁平权限节点，由领域层组装成树。
type PermissionRepository interface {
	ListAll(ctx context.Context) ([]Permission, error)
}

// TenantRepository 持久化租户。
type TenantRepository interface {
	List(ctx context.Context, query ListQuery) (PageResult[Tenant], error)
	GetByIDs(ctx context.Context, ids []string) ([]Tenant, error)
}

// Claims 是令牌中包含的主体信息。出于安全考虑，令牌不携带权限列表。
type Claims struct {
	UserID    string
	TenantID  string
	TokenID   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// TokenProvider 负责签发、校验与吊销访问令牌。
type TokenProvider interface {
	Issue(ctx context.Context, claims Claims) (string, error)
	Verify(ctx context.Context, raw string) (*Claims, error)
	Revoke(ctx context.Context, tokenID string, expiresAt time.Time) error
}

// TokenBlacklist 记录已吊销令牌的 ID，用于退出登录后的失效校验。
type TokenBlacklist interface {
	IsRevoked(ctx context.Context, tokenID string) (bool, error)
	Add(ctx context.Context, tokenID string, expiresAt time.Time) error
}

// RefreshTokenStore 持久化刷新令牌，生产环境由 Redis 实现。
type RefreshTokenStore interface {
	Save(ctx context.Context, token string, session RefreshSession, ttl time.Duration) error
	Get(ctx context.Context, token string) (*RefreshSession, error)
	Delete(ctx context.Context, token string) error
}

// PasswordHasher 抽象密码哈希算法，便于测试时替换。
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}
