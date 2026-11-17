package domain

import (
	"context"
	"time"
)

// TenantRepository 持久化租户，并提供经营主体数量、账户数量的聚合能力。
type TenantRepository interface {
	List(ctx context.Context, query ListQuery) (PageResult[Tenant], error)
	ListAll(ctx context.Context) ([]Tenant, error)
	Get(ctx context.Context, id string) (*Tenant, error)
	GetByIDs(ctx context.Context, ids []string) ([]Tenant, error)
	Create(ctx context.Context, tenant Tenant) (*Tenant, error)
	Update(ctx context.Context, tenant Tenant) (*Tenant, error)
	Count(ctx context.Context) (int64, error)
}

// LegalEntityRepository 持久化经营主体，所有查询都要求租户范围。
type LegalEntityRepository interface {
	List(ctx context.Context, query ListQuery) ([]LegalEntity, error)
	Get(ctx context.Context, id string) (*LegalEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]LegalEntity, error)
	Create(ctx context.Context, entity LegalEntity) (*LegalEntity, error)
	Update(ctx context.Context, entity LegalEntity) (*LegalEntity, error)
	CountByTenant(ctx context.Context, tenantID string) (map[string]int, error)
}

// OrgRepository 持久化组织节点，树结构由领域层组装。
type OrgRepository interface {
	ListAll(ctx context.Context, tenantID string, legalEntityID string) ([]OrgUnit, error)
	Get(ctx context.Context, id string) (*OrgUnit, error)
	GetByIDs(ctx context.Context, ids []string) ([]OrgUnit, error)
	Create(ctx context.Context, unit OrgUnit) (*OrgUnit, error)
	Update(ctx context.Context, unit OrgUnit) (*OrgUnit, error)
}

// AccountRepository 持久化管理端账户，并提供关联角色与主体的查询能力。
type AccountRepository interface {
	List(ctx context.Context, query ListQuery) (PageResult[Account], error)
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	Create(ctx context.Context, account Account) (*Account, error)
	Update(ctx context.Context, account Account) (*Account, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	CountByRoles(ctx context.Context, roleIDs []string) (map[string]int, error)
	CountByTenant(ctx context.Context, tenantID string) (int, error)
}

// RoleRepository 持久化角色，UserCount 在查询时聚合得到。
type RoleRepository interface {
	List(ctx context.Context, query ListQuery) (PageResult[Role], error)
	GetByIDs(ctx context.Context, ids []string) ([]Role, error)
	Create(ctx context.Context, role Role) (*Role, error)
	Update(ctx context.Context, role Role) (*Role, error)
}

// PermissionRepository 维护权限目录，扁平节点由领域层组装成树。
type PermissionRepository interface {
	ListAll(ctx context.Context) ([]Permission, error)
	Get(ctx context.Context, id string) (*Permission, error)
	Create(ctx context.Context, permission Permission) (*Permission, error)
	Update(ctx context.Context, permission Permission) (*Permission, error)
	Delete(ctx context.Context, id string) error
	// Reorder 按给定顺序重写同级节点的 sortOrder。
	Reorder(ctx context.Context, ids []string) error
}

// Claims 是令牌中包含的主体信息。出于安全考虑，令牌不携带权限列表。
type Claims struct {
	AccountID     string
	TenantID      string
	LegalEntityID string
	TokenID       string
	IssuedAt      time.Time
	ExpiresAt     time.Time
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
