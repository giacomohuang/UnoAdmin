// Package testutil 提供领域端口的内存实现，供单元测试替身使用。
package testutil

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/unoadmin/admin-service/internal/domain"
)

// sequence 为内存仓储生成稳定、自增的 ID。
var sequence int64

// nextID 生成带业务前缀的自增 ID。
func nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, atomic.AddInt64(&sequence, 1))
}

// AccountRepo 是账户仓储的内存实现。
type AccountRepo struct {
	Accounts map[string]domain.Account
}

// NewAccountRepo 用给定账户构造内存仓储。
func NewAccountRepo(accounts ...domain.Account) *AccountRepo {
	r := &AccountRepo{Accounts: map[string]domain.Account{}}
	for _, account := range accounts {
		r.Accounts[account.ID] = account
	}
	return r
}

// List 按搜索词、租户、主体与状态过滤后分页。
func (r *AccountRepo) List(_ context.Context, q domain.ListQuery) (domain.PageResult[domain.Account], error) {
	q = q.Normalize()
	matched := []domain.Account{}
	for _, account := range r.Accounts {
		if q.TenantID != "" && account.TenantID != q.TenantID {
			continue
		}
		if q.LegalEntityID != "" && !contains(account.LegalEntityIDs, q.LegalEntityID) {
			continue
		}
		if len(q.LegalEntityIDs) > 0 && !intersects(account.LegalEntityIDs, q.LegalEntityIDs) {
			continue
		}
		if q.Status != "" && account.Status != q.Status {
			continue
		}
		if q.Search != "" && !containsFold(account.Username+account.DisplayName+account.Email+account.Phone, q.Search) {
			continue
		}
		matched = append(matched, account)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })
	return page(matched, q), nil
}

// GetByID 按 ID 查找账户。
func (r *AccountRepo) GetByID(_ context.Context, id string) (*domain.Account, error) {
	account, ok := r.Accounts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &account, nil
}

// GetByUsername 按用户名查找账户。
func (r *AccountRepo) GetByUsername(_ context.Context, username string) (*domain.Account, error) {
	for _, account := range r.Accounts {
		if account.Username == username {
			return &account, nil
		}
	}
	return nil, domain.ErrNotFound
}

// Create 写入账户并拒绝重复用户名。
func (r *AccountRepo) Create(_ context.Context, account domain.Account) (*domain.Account, error) {
	for _, existing := range r.Accounts {
		if existing.Username == account.Username {
			return nil, domain.ErrConflict
		}
	}
	account.ID = nextID("acc")
	r.Accounts[account.ID] = account
	return &account, nil
}

// Update 覆盖账户记录。
func (r *AccountRepo) Update(_ context.Context, account domain.Account) (*domain.Account, error) {
	if _, ok := r.Accounts[account.ID]; !ok {
		return nil, domain.ErrNotFound
	}
	r.Accounts[account.ID] = account
	return &account, nil
}

// UpdateStatus 修改账户状态。
func (r *AccountRepo) UpdateStatus(_ context.Context, id string, status string) error {
	account, ok := r.Accounts[id]
	if !ok {
		return domain.ErrNotFound
	}
	account.Status = status
	r.Accounts[id] = account
	return nil
}

// CountByRoles 统计每个角色关联的账户数。
func (r *AccountRepo) CountByRoles(_ context.Context, roleIDs []string) (map[string]int, error) {
	counts := map[string]int{}
	for _, account := range r.Accounts {
		for _, roleID := range account.RoleIDs {
			counts[roleID]++
		}
	}
	return counts, nil
}

// CountByTenant 统计指定租户下的账户数量。
func (r *AccountRepo) CountByTenant(_ context.Context, tenantID string) (int, error) {
	total := 0
	for _, account := range r.Accounts {
		if account.TenantID == tenantID {
			total++
		}
	}
	return total, nil
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

// List 按租户、状态与搜索词过滤后分页。
func (r *RoleRepo) List(_ context.Context, q domain.ListQuery) (domain.PageResult[domain.Role], error) {
	q = q.Normalize()
	matched := []domain.Role{}
	for _, role := range r.Roles {
		if q.TenantID != "" && role.TenantID != q.TenantID {
			continue
		}
		if q.Status != "" && role.Status != q.Status {
			continue
		}
		if q.Search != "" && !containsFold(role.Name+role.Code+role.Description+strings.Join(role.PermissionCodes, " "), q.Search) {
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

// Create 写入角色并拒绝租户内重复编码。
func (r *RoleRepo) Create(_ context.Context, role domain.Role) (*domain.Role, error) {
	for _, existing := range r.Roles {
		if existing.TenantID == role.TenantID && existing.Code == role.Code {
			return nil, domain.ErrConflict
		}
	}
	role.ID = nextID("role")
	r.Roles[role.ID] = role
	return &role, nil
}

// Update 覆盖角色记录。
func (r *RoleRepo) Update(_ context.Context, role domain.Role) (*domain.Role, error) {
	if _, ok := r.Roles[role.ID]; !ok {
		return nil, domain.ErrNotFound
	}
	r.Roles[role.ID] = role
	return &role, nil
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

// Get 按 ID 查询权限节点。
func (r *PermissionRepo) Get(_ context.Context, id string) (*domain.Permission, error) {
	for _, node := range r.Nodes {
		if node.ID == id {
			return &node, nil
		}
	}
	return nil, domain.ErrNotFound
}

// Create 写入权限节点并拒绝重复编码。
func (r *PermissionRepo) Create(_ context.Context, permission domain.Permission) (*domain.Permission, error) {
	for _, node := range r.Nodes {
		if node.Code == permission.Code {
			return nil, domain.ErrConflict
		}
	}
	permission.ID = nextID("perm")
	r.Nodes = append(r.Nodes, permission)
	return &permission, nil
}

// Update 覆盖权限节点。
func (r *PermissionRepo) Update(_ context.Context, permission domain.Permission) (*domain.Permission, error) {
	for index, node := range r.Nodes {
		if node.ID != permission.ID {
			continue
		}
		if node.Code != permission.Code {
			for _, other := range r.Nodes {
				if other.Code == permission.Code {
					return nil, domain.ErrConflict
				}
			}
		}
		r.Nodes[index] = permission
		return &permission, nil
	}
	return nil, domain.ErrNotFound
}

// Delete 移除权限节点。
func (r *PermissionRepo) Delete(_ context.Context, id string) error {
	for index, node := range r.Nodes {
		if node.ID == id {
			r.Nodes = append(r.Nodes[:index], r.Nodes[index+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

// Reorder 按给定顺序重写同级节点的 sortOrder。
func (r *PermissionRepo) Reorder(_ context.Context, ids []string) error {
	order := make(map[string]int, len(ids))
	for index, id := range ids {
		order[id] = index
	}
	for index, node := range r.Nodes {
		if sort, ok := order[node.ID]; ok {
			r.Nodes[index].SortOrder = sort
		}
	}
	return nil
}

// TenantRepo 是租户仓储的内存实现。
type TenantRepo struct {
	Tenants map[string]domain.Tenant
}

// NewTenantRepo 用给定租户构造内存仓储。
func NewTenantRepo(tenants ...domain.Tenant) *TenantRepo {
	r := &TenantRepo{Tenants: map[string]domain.Tenant{}}
	for _, tenant := range tenants {
		r.Tenants[tenant.ID] = tenant
	}
	return r
}

// List 按搜索词过滤后分页。
func (r *TenantRepo) List(_ context.Context, q domain.ListQuery) (domain.PageResult[domain.Tenant], error) {
	q = q.Normalize()
	matched := []domain.Tenant{}
	for _, tenant := range r.Tenants {
		if q.Status != "" && tenant.Status != q.Status {
			continue
		}
		if q.Search != "" && !containsFold(tenant.Name+tenant.Code+tenant.Contact, q.Search) {
			continue
		}
		matched = append(matched, tenant)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })
	return page(matched, q), nil
}

// ListAll 返回全部租户。
func (r *TenantRepo) ListAll(_ context.Context) ([]domain.Tenant, error) {
	items := make([]domain.Tenant, 0, len(r.Tenants))
	for _, tenant := range r.Tenants {
		items = append(items, tenant)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

// Get 按 ID 查找租户。
func (r *TenantRepo) Get(_ context.Context, id string) (*domain.Tenant, error) {
	tenant, ok := r.Tenants[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &tenant, nil
}

// GetByIDs 返回给定 ID 对应的租户，忽略不存在的 ID。
func (r *TenantRepo) GetByIDs(_ context.Context, ids []string) ([]domain.Tenant, error) {
	result := []domain.Tenant{}
	for _, id := range ids {
		if tenant, ok := r.Tenants[id]; ok {
			result = append(result, tenant)
		}
	}
	return result, nil
}

// Create 写入租户并拒绝重复编码。
func (r *TenantRepo) Create(_ context.Context, tenant domain.Tenant) (*domain.Tenant, error) {
	for _, existing := range r.Tenants {
		if existing.Code == tenant.Code {
			return nil, domain.ErrConflict
		}
	}
	tenant.ID = nextID("tenant")
	r.Tenants[tenant.ID] = tenant
	return &tenant, nil
}

// Update 覆盖租户记录。
func (r *TenantRepo) Update(_ context.Context, tenant domain.Tenant) (*domain.Tenant, error) {
	if _, ok := r.Tenants[tenant.ID]; !ok {
		return nil, domain.ErrNotFound
	}
	r.Tenants[tenant.ID] = tenant
	return &tenant, nil
}

// Count 返回租户总数。
func (r *TenantRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.Tenants)), nil
}

// LegalEntityRepo 是经营主体仓储的内存实现。
type LegalEntityRepo struct {
	Entities map[string]domain.LegalEntity
}

// NewLegalEntityRepo 用给定主体构造内存仓储。
func NewLegalEntityRepo(entities ...domain.LegalEntity) *LegalEntityRepo {
	r := &LegalEntityRepo{Entities: map[string]domain.LegalEntity{}}
	for _, entity := range entities {
		r.Entities[entity.ID] = entity
	}
	return r
}

// List 按租户、状态与搜索词过滤。
func (r *LegalEntityRepo) List(_ context.Context, q domain.ListQuery) ([]domain.LegalEntity, error) {
	items := []domain.LegalEntity{}
	for _, entity := range r.Entities {
		if q.TenantID != "" && entity.TenantID != q.TenantID {
			continue
		}
		if q.Status != "" && entity.Status != q.Status {
			continue
		}
		if q.Search != "" && !containsFold(entity.Name+entity.Code+entity.Address, q.Search) {
			continue
		}
		items = append(items, entity)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

// Get 按 ID 查找主体。
func (r *LegalEntityRepo) Get(_ context.Context, id string) (*domain.LegalEntity, error) {
	entity, ok := r.Entities[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &entity, nil
}

// GetByIDs 返回给定 ID 对应的主体。
func (r *LegalEntityRepo) GetByIDs(_ context.Context, ids []string) ([]domain.LegalEntity, error) {
	result := []domain.LegalEntity{}
	for _, id := range ids {
		if entity, ok := r.Entities[id]; ok {
			result = append(result, entity)
		}
	}
	return result, nil
}

// Create 写入主体并拒绝租户内重复编码。
func (r *LegalEntityRepo) Create(_ context.Context, entity domain.LegalEntity) (*domain.LegalEntity, error) {
	for _, existing := range r.Entities {
		if existing.TenantID == entity.TenantID && existing.Code == entity.Code {
			return nil, domain.ErrConflict
		}
	}
	entity.ID = nextID("le")
	r.Entities[entity.ID] = entity
	return &entity, nil
}

// Update 覆盖主体记录。
func (r *LegalEntityRepo) Update(_ context.Context, entity domain.LegalEntity) (*domain.LegalEntity, error) {
	if _, ok := r.Entities[entity.ID]; !ok {
		return nil, domain.ErrNotFound
	}
	r.Entities[entity.ID] = entity
	return &entity, nil
}

// CountByTenant 统计每个租户的主体数量。
func (r *LegalEntityRepo) CountByTenant(_ context.Context, tenantID string) (map[string]int, error) {
	counts := map[string]int{}
	for _, entity := range r.Entities {
		if tenantID == "" || entity.TenantID == tenantID {
			counts[entity.TenantID]++
		}
	}
	return counts, nil
}

// OrgRepo 是组织仓储的内存实现。
type OrgRepo struct {
	Units map[string]domain.OrgUnit
}

// NewOrgRepo 用给定节点构造内存仓储。
func NewOrgRepo(units ...domain.OrgUnit) *OrgRepo {
	r := &OrgRepo{Units: map[string]domain.OrgUnit{}}
	for _, unit := range units {
		r.Units[unit.ID] = unit
	}
	return r
}

// ListAll 按租户与主体过滤组织节点。
func (r *OrgRepo) ListAll(_ context.Context, tenantID string, legalEntityID string) ([]domain.OrgUnit, error) {
	items := []domain.OrgUnit{}
	for _, unit := range r.Units {
		if tenantID != "" && unit.TenantID != tenantID {
			continue
		}
		if legalEntityID != "" && unit.LegalEntityID != legalEntityID {
			continue
		}
		items = append(items, unit)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

// Get 按 ID 查找组织节点。
func (r *OrgRepo) Get(_ context.Context, id string) (*domain.OrgUnit, error) {
	unit, ok := r.Units[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &unit, nil
}

// GetByIDs 返回给定 ID 对应的组织节点。
func (r *OrgRepo) GetByIDs(_ context.Context, ids []string) ([]domain.OrgUnit, error) {
	result := []domain.OrgUnit{}
	for _, id := range ids {
		if unit, ok := r.Units[id]; ok {
			result = append(result, unit)
		}
	}
	return result, nil
}

// Create 写入组织节点。
func (r *OrgRepo) Create(_ context.Context, unit domain.OrgUnit) (*domain.OrgUnit, error) {
	unit.ID = nextID("org")
	r.Units[unit.ID] = unit
	return &unit, nil
}

// Update 覆盖组织节点。
func (r *OrgRepo) Update(_ context.Context, unit domain.OrgUnit) (*domain.OrgUnit, error) {
	if _, ok := r.Units[unit.ID]; !ok {
		return nil, domain.ErrNotFound
	}
	r.Units[unit.ID] = unit
	return &unit, nil
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

// intersects 判断两个字符串切片是否有交集。
func intersects(items []string, targets []string) bool {
	for _, item := range items {
		if contains(targets, item) {
			return true
		}
	}
	return false
}
