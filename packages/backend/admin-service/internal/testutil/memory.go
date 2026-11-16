// Package testutil 提供领域端口的内存实现，供单元测试替身使用。
package testutil

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/unoadmin/admin-service/internal/domain"
)

// UserRepo 是用户仓储的内存实现。
type UserRepo struct {
	Users map[string]domain.User
}

// NewUserRepo 用给定用户构造内存仓储。
func NewUserRepo(users ...domain.User) *UserRepo {
	r := &UserRepo{Users: map[string]domain.User{}}
	for _, u := range users {
		r.Users[u.ID] = u
	}
	return r
}

// List 按搜索词与租户过滤后分页，结果按 ID 排序以保证断言稳定。
func (r *UserRepo) List(_ context.Context, q domain.ListQuery) (domain.PageResult[domain.User], error) {
	q = q.Normalize()
	matched := []domain.User{}
	for _, u := range r.Users {
		if q.Search != "" && !containsFold(u.Username+u.DisplayName+u.Email+strings.Join(u.RoleNames, " "), q.Search) {
			continue
		}
		if q.TenantID != "" && !contains(u.TenantIDs, q.TenantID) {
			continue
		}
		matched = append(matched, u)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })
	return page(matched, q), nil
}

// GetByID 按 ID 查找用户。
func (r *UserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.Users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &u, nil
}

// GetByUsername 按用户名查找用户。
func (r *UserRepo) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	for _, u := range r.Users {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, domain.ErrNotFound
}

// UpdateStatus 修改用户状态。
func (r *UserRepo) UpdateStatus(_ context.Context, id string, status string) error {
	u, ok := r.Users[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.Status = status
	r.Users[id] = u
	return nil
}

// CountByRoles 统计每个角色关联的用户数。
func (r *UserRepo) CountByRoles(_ context.Context, roleIDs []string) (map[string]int, error) {
	counts := map[string]int{}
	for _, u := range r.Users {
		for _, roleID := range u.RoleIDs {
			counts[roleID]++
		}
	}
	return counts, nil
}

// RoleRepo 是角色仓储的内存实现。
type RoleRepo struct {
	Roles map[string]domain.Role
}

// NewRoleRepo 用给定角色构造内存仓储。
func NewRoleRepo(roles ...domain.Role) *RoleRepo {
	r := &RoleRepo{Roles: map[string]domain.Role{}}
	for _, role := range roles {
		r.Roles[role.ID] = role
	}
	return r
}

// List 按搜索词过滤后分页。
func (r *RoleRepo) List(_ context.Context, q domain.ListQuery) (domain.PageResult[domain.Role], error) {
	q = q.Normalize()
	matched := []domain.Role{}
	for _, role := range r.Roles {
		if q.Search != "" && !containsFold(role.Name+role.Description+strings.Join(role.PermissionCodes, " "), q.Search) {
			continue
		}
		matched = append(matched, role)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })
	return page(matched, q), nil
}

// GetByIDs 返回给定 ID 对应的角色，忽略不存在的 ID。
func (r *RoleRepo) GetByIDs(_ context.Context, ids []string) ([]domain.Role, error) {
	result := []domain.Role{}
	for _, id := range ids {
		if role, ok := r.Roles[id]; ok {
			result = append(result, role)
		}
	}
	return result, nil
}

// PermissionRepo 是权限仓储的内存实现。
type PermissionRepo struct {
	Nodes []domain.Permission
}

// NewPermissionRepo 用给定节点构造内存仓储。
func NewPermissionRepo(nodes ...domain.Permission) *PermissionRepo {
	return &PermissionRepo{Nodes: nodes}
}

// ListAll 返回全部扁平权限节点。
func (r *PermissionRepo) ListAll(_ context.Context) ([]domain.Permission, error) {
	return r.Nodes, nil
}

// TenantRepo 是租户仓储的内存实现。
type TenantRepo struct {
	Tenants map[string]domain.Tenant
}

// NewTenantRepo 用给定租户构造内存仓储。
func NewTenantRepo(tenants ...domain.Tenant) *TenantRepo {
	r := &TenantRepo{Tenants: map[string]domain.Tenant{}}
	for _, t := range tenants {
		r.Tenants[t.ID] = t
	}
	return r
}

// List 按搜索词过滤后分页。
func (r *TenantRepo) List(_ context.Context, q domain.ListQuery) (domain.PageResult[domain.Tenant], error) {
	q = q.Normalize()
	matched := []domain.Tenant{}
	for _, t := range r.Tenants {
		if q.Search != "" && !containsFold(t.Name+t.Code, q.Search) {
			continue
		}
		matched = append(matched, t)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })
	return page(matched, q), nil
}

// GetByIDs 返回给定 ID 对应的租户，忽略不存在的 ID。
func (r *TenantRepo) GetByIDs(_ context.Context, ids []string) ([]domain.Tenant, error) {
	result := []domain.Tenant{}
	for _, id := range ids {
		if t, ok := r.Tenants[id]; ok {
			result = append(result, t)
		}
	}
	return result, nil
}

// TokenProvider 是令牌服务的测试替身，记录签发内容并按要求返回校验结果。
type TokenProvider struct {
	Issued []domain.Claims
	Claims *domain.Claims
	Err    error
}

// Issue 记录待签发的声明。
func (p *TokenProvider) Issue(_ context.Context, claims domain.Claims) (string, error) {
	p.Issued = append(p.Issued, claims)
	return "issued-token", nil
}

// Verify 返回预设的声明或错误。
func (p *TokenProvider) Verify(_ context.Context, _ string) (*domain.Claims, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	return p.Claims, nil
}

// Revoke 在测试替身中不执行任何操作。
func (p *TokenProvider) Revoke(_ context.Context, _ string, _ time.Time) error {
	return nil
}

// Blacklist 是令牌黑名单的内存实现。
type Blacklist struct {
	IDs map[string]bool
}

// NewBlacklist 构造空黑名单。
func NewBlacklist() *Blacklist {
	return &Blacklist{IDs: map[string]bool{}}
}

// IsRevoked 判断令牌 ID 是否已被吊销。
func (b *Blacklist) IsRevoked(_ context.Context, tokenID string) (bool, error) {
	return b.IDs[tokenID], nil
}

// Add 将令牌 ID 加入黑名单。
func (b *Blacklist) Add(_ context.Context, tokenID string, _ time.Time) error {
	b.IDs[tokenID] = true
	return nil
}

// RefreshStore 是刷新令牌存储的内存实现。
type RefreshStore struct {
	Sessions map[string]domain.RefreshSession
}

// NewRefreshStore 构造空刷新令牌存储。
func NewRefreshStore() *RefreshStore {
	return &RefreshStore{Sessions: map[string]domain.RefreshSession{}}
}

// Save 记录令牌对应的会话。
func (s *RefreshStore) Save(_ context.Context, token string, session domain.RefreshSession, _ time.Duration) error {
	s.Sessions[token] = session
	return nil
}

// Get 读取令牌对应的会话。
func (s *RefreshStore) Get(_ context.Context, token string) (*domain.RefreshSession, error) {
	session, ok := s.Sessions[token]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &session, nil
}

// Delete 删除令牌。
func (s *RefreshStore) Delete(_ context.Context, token string) error {
	delete(s.Sessions, token)
	return nil
}

// PlainHasher 是不做真实哈希的测试替身，仅用于验证调用链路。
type PlainHasher struct{}

// Hash 为测试密码加上固定前缀。
func (PlainHasher) Hash(password string) (string, error) { return "hash:" + password, nil }

// Compare 比较测试哈希与密码。
func (PlainHasher) Compare(hash string, password string) error {
	if hash != "hash:"+password {
		return errors.New("mismatched password")
	}
	return nil
}

// page 对已排序切片执行内存分页。
func page[T any](items []T, q domain.ListQuery) domain.PageResult[T] {
	total := int64(len(items))
	start := int(q.Offset())
	if start > len(items) {
		start = len(items)
	}
	end := start + int(q.Limit())
	if end > len(items) {
		end = len(items)
	}
	return domain.PageResult[T]{Items: items[start:end], Total: total}
}

func containsFold(haystack string, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
