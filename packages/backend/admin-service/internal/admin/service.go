// Package admin 实现多租户管理端页面所需的查询与写入用例。
//
// 所有方法都接收调用主体（Principal），由服务层强制租户与经营主体边界，
// 处理器不得把请求参数直接当作范围条件使用。
package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/unoadmin/admin-service/internal/domain"
)

// Service 编排租户、经营主体、组织、账户、角色与权限的读写逻辑。
type Service struct {
	accounts    domain.AccountRepository
	roles       domain.RoleRepository
	permissions domain.PermissionRepository
	tenants     domain.TenantRepository
	entities    domain.LegalEntityRepository
	orgs        domain.OrgRepository
	hasher      domain.PasswordHasher
}

// NewService 构造基础管理服务。
func NewService(
	accounts domain.AccountRepository,
	roles domain.RoleRepository,
	permissions domain.PermissionRepository,
	tenants domain.TenantRepository,
	entities domain.LegalEntityRepository,
	orgs domain.OrgRepository,
	hasher domain.PasswordHasher,
) *Service {
	return &Service{
		accounts: accounts, roles: roles, permissions: permissions,
		tenants: tenants, entities: entities, orgs: orgs, hasher: hasher,
	}
}

// TenantInput 是新建或更新租户时允许提交的字段。
type TenantInput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Mode    string `json:"mode"`
	Plan    string `json:"plan"`
	Contact string `json:"contact"`
	Status  string `json:"status"`
}

// LegalEntityInput 是新建或更新经营主体时允许提交的字段。
type LegalEntityInput struct {
	ID       string `json:"id"`
	TenantID string `json:"tenantId"`
	ParentID string `json:"parentId"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Address  string `json:"address"`
	Status   string `json:"status"`
}

// OrgInput 是新建或更新组织节点时允许提交的字段。
type OrgInput struct {
	ID            string `json:"id"`
	LegalEntityID string `json:"legalEntityId"`
	ParentID      string `json:"parentId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Sort          int    `json:"sort"`
	Status        string `json:"status"`
}

// AccountInput 是新建或更新账户时允许提交的字段。
type AccountInput struct {
	ID             string   `json:"id"`
	AccountName    string   `json:"accountName"`
	DisplayName    string   `json:"displayName"`
	Password       string   `json:"password"`
	Email          string   `json:"email"`
	Phone          string   `json:"phone"`
	Status         string   `json:"status"`
	DataScope      string   `json:"dataScope"`
	LegalEntityIDs []string `json:"legalEntityIds"`
	OrgIDs         []string `json:"orgIds"`
	RoleIDs        []string `json:"roleIds"`
}

// RoleInput 是新建或更新角色时允许提交的字段。
type RoleInput struct {
	ID              string   `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Status          string   `json:"status"`
	DataScope       string   `json:"dataScope"`
	LegalEntityID   string   `json:"legalEntityId"`
	PermissionCodes []string `json:"permissionCodes"`
}

// PermissionInput 是新建或更新权限节点时允许提交的字段。
type PermissionInput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Type      string `json:"type"`
	Scope     string `json:"scope"`
	ParentID  string `json:"parentId"`
	SortOrder int    `json:"sortOrder"`
}

// PermissionReorderInput 是同级权限拖拽排序的请求体。
type PermissionReorderInput struct {
	ParentID string   `json:"parentId"`
	IDs      []string `json:"ids"`
}

// TenantOverview 是租户列表的一行，主体列表与统计数量由 Tenant 的内嵌字段承载。
type TenantOverview struct {
	domain.Tenant
}

// ListTenants 分页查询租户；非平台管理员只能看到自己的租户。
func (s *Service) ListTenants(ctx context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[TenantOverview], error) {
	if !principal.PlatformAdmin {
		tenant, err := s.tenants.Get(ctx, principal.TenantID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.PageResult[TenantOverview]{Items: []TenantOverview{}, Total: 0}, nil
		}
		if err != nil {
			return domain.PageResult[TenantOverview]{}, err
		}
		overview, err := s.decorateTenant(ctx, *tenant)
		if err != nil {
			return domain.PageResult[TenantOverview]{}, err
		}
		return domain.PageResult[TenantOverview]{Items: []TenantOverview{overview}, Total: 1}, nil
	}

	result, err := s.tenants.List(ctx, query)
	if err != nil {
		return domain.PageResult[TenantOverview]{}, err
	}
	items := make([]TenantOverview, 0, len(result.Items))
	for _, tenant := range result.Items {
		overview, err := s.decorateTenant(ctx, tenant)
		if err != nil {
			return domain.PageResult[TenantOverview]{}, err
		}
		items = append(items, overview)
	}
	return domain.PageResult[TenantOverview]{Items: items, Total: result.Total}, nil
}

// decorateTenant 补充租户的主体列表与统计数量。
func (s *Service) decorateTenant(ctx context.Context, tenant domain.Tenant) (TenantOverview, error) {
	entities, err := s.entities.List(ctx, domain.ListQuery{TenantID: tenant.ID})
	if err != nil {
		return TenantOverview{}, err
	}
	accountCount, err := s.accounts.CountByTenant(ctx, tenant.ID)
	if err != nil {
		return TenantOverview{}, err
	}
	tenant.LegalEntities = domain.SortLegalEntities(entities)
	tenant.EntityCount = len(entities)
	tenant.AccountCount = accountCount
	return TenantOverview{Tenant: tenant}, nil
}

// SaveTenant 创建或更新租户，并按形态约束经营主体数量。
//
// 单体租户创建时自动生成唯一的经营主体，保证「租户 → 主体」一一对应；
// 集团租户可由管理员继续添加多个经营主体。
func (s *Service) SaveTenant(ctx context.Context, principal *domain.Principal, input TenantInput) (*TenantOverview, error) {
	if !principal.PlatformAdmin {
		return nil, domain.ErrForbidden
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	if input.Name == "" || input.Code == "" {
		return nil, fmt.Errorf("%w: tenant name and code are required", domain.ErrInvalid)
	}
	if err := domain.ValidateTenantMode(input.Mode, 0); err != nil {
		return nil, err
	}
	if input.Status == "" {
		input.Status = domain.TenantActive
	}
	if input.Plan == "" {
		input.Plan = "standard"
	}

	now := time.Now().UTC()
	if input.ID == "" {
		tenant, err := s.tenants.Create(ctx, domain.Tenant{
			Name: input.Name, Code: input.Code, Mode: input.Mode, Plan: input.Plan,
			Contact: input.Contact, Status: input.Status, CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return nil, err
		}
		if tenant.Mode == domain.TenantModeStandalone {
			if _, err := s.entities.Create(ctx, domain.LegalEntity{
				TenantID: tenant.ID, Type: domain.LegalEntityTypeCompany,
				Name: tenant.Name, Code: tenant.Code, Status: domain.TenantActive,
				CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return nil, err
			}
		}
		overview, err := s.decorateTenant(ctx, *tenant)
		if err != nil {
			return nil, err
		}
		return &overview, nil
	}

	current, err := s.tenants.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	entities, err := s.entities.List(ctx, domain.ListQuery{TenantID: current.ID})
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateTenantMode(input.Mode, len(entities)); err != nil {
		return nil, err
	}
	if current.Mode == domain.TenantModeGroup && input.Mode == domain.TenantModeStandalone && len(entities) > 1 {
		return nil, fmt.Errorf("%w: remove extra legal entities before switching to standalone", domain.ErrConflict)
	}

	current.Name = input.Name
	current.Code = input.Code
	current.Mode = input.Mode
	current.Plan = input.Plan
	current.Contact = input.Contact
	current.Status = input.Status
	current.UpdatedAt = now
	updated, err := s.tenants.Update(ctx, *current)
	if err != nil {
		return nil, err
	}
	overview, err := s.decorateTenant(ctx, *updated)
	if err != nil {
		return nil, err
	}
	return &overview, nil
}

// ListLegalEntities 返回主体范围内的经营主体列表。
func (s *Service) ListLegalEntities(ctx context.Context, principal *domain.Principal, query domain.ListQuery) ([]domain.LegalEntity, error) {
	tenantID := s.scopeTenant(principal, query.TenantID)
	if tenantID == "" {
		return []domain.LegalEntity{}, nil
	}
	query.TenantID = tenantID
	entities, err := s.entities.List(ctx, query)
	if err != nil {
		return nil, err
	}
	return domain.SortLegalEntities(domain.VisibleEntities(principal, entities)), nil
}

// SaveLegalEntity 创建或更新经营主体。
//
// 集团租户只允许一个集团总部主体，其余公司必须挂在集团主体之下；
// 单体租户不允许再新增主体。
func (s *Service) SaveLegalEntity(ctx context.Context, principal *domain.Principal, input LegalEntityInput) (*domain.LegalEntity, error) {
	tenantID := s.scopeTenant(principal, input.TenantID)
	if tenantID == "" {
		return nil, domain.ErrForbidden
	}
	tenant, err := s.tenants.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !principal.PlatformAdmin && !domain.CanAccessEntity(principal, input.ID) && input.ID != "" {
		return nil, domain.ErrForbidden
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	if input.Name == "" || input.Code == "" {
		return nil, fmt.Errorf("%w: entity name and code are required", domain.ErrInvalid)
	}
	if input.Type != domain.LegalEntityTypeGroup && input.Type != domain.LegalEntityTypeCompany {
		return nil, fmt.Errorf("%w: unsupported entity type %q", domain.ErrInvalid, input.Type)
	}
	if input.Status == "" {
		input.Status = domain.TenantActive
	}

	existing, err := s.entities.List(ctx, domain.ListQuery{TenantID: tenantID})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	if input.ID == "" {
		if err := domain.ValidateTenantMode(tenant.Mode, len(existing)+1); err != nil {
			return nil, err
		}
		if input.Type == domain.LegalEntityTypeGroup {
			for _, entity := range existing {
				if entity.Type == domain.LegalEntityTypeGroup {
					return nil, fmt.Errorf("%w: tenant already owns a group entity", domain.ErrConflict)
				}
			}
		}
		if input.Type == domain.LegalEntityTypeCompany && input.ParentID != "" {
			parent, err := s.entities.Get(ctx, input.ParentID)
			if err != nil {
				return nil, err
			}
			if parent.TenantID != tenantID || parent.Type != domain.LegalEntityTypeGroup {
				return nil, fmt.Errorf("%w: company must belong to a group entity in the same tenant", domain.ErrInvalid)
			}
		}
		return s.entities.Create(ctx, domain.LegalEntity{
			TenantID: tenantID, ParentID: input.ParentID, Type: input.Type,
			Name: input.Name, Code: input.Code, Address: input.Address,
			Status: input.Status, CreatedAt: now, UpdatedAt: now,
		})
	}

	current, err := s.entities.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if current.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}
	if input.ParentID == current.ID {
		return nil, fmt.Errorf("%w: entity cannot be its own parent", domain.ErrInvalid)
	}
	if current.Type == domain.LegalEntityTypeCompany && input.ParentID == "" {
		return nil, fmt.Errorf("%w: company must belong to a group entity", domain.ErrInvalid)
	}
	current.ParentID = input.ParentID
	current.Type = input.Type
	current.Name = input.Name
	current.Code = input.Code
	current.Address = input.Address
	current.Status = input.Status
	current.UpdatedAt = now
	return s.entities.Update(ctx, *current)
}

// ListOrgs 返回指定经营主体下的组织树。
func (s *Service) ListOrgs(ctx context.Context, principal *domain.Principal, entityID string) ([]domain.OrgUnit, error) {
	if entityID == "" {
		return []domain.OrgUnit{}, nil
	}
	entity, err := s.entities.Get(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if !domain.CanAccessTenant(principal, entity.TenantID) || !domain.CanAccessEntity(principal, entity.ID) {
		return nil, domain.ErrForbidden
	}
	nodes, err := s.orgs.ListAll(ctx, entity.TenantID, entity.ID)
	if err != nil {
		return nil, err
	}
	return domain.BuildOrgTree(nodes), nil
}

// SaveOrg 创建或更新组织节点，并维护从根到自身的路径。
func (s *Service) SaveOrg(ctx context.Context, principal *domain.Principal, input OrgInput) (*domain.OrgUnit, error) {
	entity, err := s.entities.Get(ctx, input.LegalEntityID)
	if err != nil {
		return nil, err
	}
	if !domain.CanAccessTenant(principal, entity.TenantID) || !domain.CanAccessEntity(principal, entity.ID) {
		return nil, domain.ErrForbidden
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, fmt.Errorf("%w: organization name is required", domain.ErrInvalid)
	}
	if input.Status == "" {
		input.Status = domain.TenantActive
	}
	now := time.Now().UTC()

	existing, err := s.orgs.ListAll(ctx, entity.TenantID, entity.ID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.OrgUnit, len(existing))
	for _, node := range existing {
		byID[node.ID] = node
	}

	path := []string{}
	if input.ParentID != "" {
		parent, ok := byID[input.ParentID]
		if !ok {
			return nil, fmt.Errorf("%w: parent organization not found in this entity", domain.ErrInvalid)
		}
		if input.ID != "" && contains(parent.Path, input.ID) {
			return nil, fmt.Errorf("%w: organization cannot move under its own descendant", domain.ErrInvalid)
		}
		path = append(path, parent.Path...)
	}

	if input.ID == "" {
		created, err := s.orgs.Create(ctx, domain.OrgUnit{
			TenantID: entity.TenantID, LegalEntityID: entity.ID, ParentID: input.ParentID,
			Name: input.Name, Type: input.Type, Path: append(path, ""), Sort: input.Sort,
			Status: input.Status, CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return nil, err
		}
		// 创建后路径包含自身 ID，需要再写回一次。
		created.Path = append(path, created.ID)
		if _, err := s.orgs.Update(ctx, *created); err != nil {
			return nil, err
		}
		return created, nil
	}

	current, ok := byID[input.ID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	current.ParentID = input.ParentID
	current.Name = input.Name
	current.Type = input.Type
	current.Path = append(path, current.ID)
	current.Sort = input.Sort
	current.Status = input.Status
	current.UpdatedAt = now
	return s.orgs.Update(ctx, current)
}

// ListAccounts 分页查询账户，并填充角色名与主体名。
func (s *Service) ListAccounts(ctx context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[domain.Account], error) {
	query.TenantID = s.scopeTenant(principal, query.TenantID)
	if query.TenantID == "" {
		return domain.PageResult[domain.Account]{Items: []domain.Account{}, Total: 0}, nil
	}
	// 主体级账户只能看到与自己授权主体有交集的账户，避免跨主体泄露人员信息。
	if !principal.PlatformAdmin && principal.DataScope != domain.DataScopeTenant {
		if len(principal.LegalEntityIDs) == 0 {
			return domain.PageResult[domain.Account]{Items: []domain.Account{}, Total: 0}, nil
		}
		query.LegalEntityIDs = principal.LegalEntityIDs
	}
	result, err := s.accounts.List(ctx, query)
	if err != nil {
		return domain.PageResult[domain.Account]{}, err
	}
	roleIDs := make([]string, 0)
	entityIDs := make([]string, 0)
	for _, account := range result.Items {
		roleIDs = append(roleIDs, account.RoleIDs...)
		entityIDs = append(entityIDs, account.LegalEntityIDs...)
	}
	roles, err := s.roles.GetByIDs(ctx, roleIDs)
	if err != nil {
		return domain.PageResult[domain.Account]{}, err
	}
	entities, err := s.entities.GetByIDs(ctx, entityIDs)
	if err != nil {
		return domain.PageResult[domain.Account]{}, err
	}
	roleNames := make(map[string]string, len(roles))
	for _, role := range roles {
		roleNames[role.ID] = role.Name
	}
	entityNames := make(map[string]string, len(entities))
	for _, entity := range entities {
		entityNames[entity.ID] = entity.Name
	}

	items := make([]domain.Account, 0, len(result.Items))
	for _, account := range result.Items {
		public := account.Public()
		public.RoleNames = make([]string, 0, len(account.RoleIDs))
		for _, id := range account.RoleIDs {
			if name, ok := roleNames[id]; ok {
				public.RoleNames = append(public.RoleNames, name)
			}
		}
		public.LegalEntityNames = make([]string, 0, len(account.LegalEntityIDs))
		for _, id := range account.LegalEntityIDs {
			if name, ok := entityNames[id]; ok {
				public.LegalEntityNames = append(public.LegalEntityNames, name)
			}
		}
		items = append(items, public)
	}
	return domain.PageResult[domain.Account]{Items: items, Total: result.Total}, nil
}

// GetAccount 查询单个账户，供 gRPC 与内部调用使用。
func (s *Service) GetAccount(ctx context.Context, id string) (domain.Account, error) {
	account, err := s.accounts.GetByID(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}
	roles, err := s.roles.GetByIDs(ctx, account.RoleIDs)
	if err != nil {
		return domain.Account{}, err
	}
	public := account.Public()
	public.RoleNames = make([]string, 0, len(roles))
	for _, role := range roles {
		public.RoleNames = append(public.RoleNames, role.Name)
	}
	return public, nil
}

// SaveAccount 创建或更新账户，并校验主体、组织与角色的租户归属。
func (s *Service) SaveAccount(ctx context.Context, principal *domain.Principal, input AccountInput) (*domain.Account, error) {
	tenantID := s.scopeTenant(principal, "")
	if tenantID == "" {
		return nil, domain.ErrForbidden
	}
	input.AccountName = strings.TrimSpace(input.AccountName)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" {
		return nil, fmt.Errorf("%w: display name is required", domain.ErrInvalid)
	}
	if input.DataScope == "" {
		input.DataScope = domain.DataScopeEntity
	}
	if err := domain.ValidateDataScope(input.DataScope); err != nil {
		return nil, err
	}
	if input.Status == "" {
		input.Status = domain.UserActive
	}

	entities, err := s.entities.List(ctx, domain.ListQuery{TenantID: tenantID})
	if err != nil {
		return nil, err
	}
	validEntity := make(map[string]bool, len(entities))
	for _, entity := range entities {
		validEntity[entity.ID] = true
	}
	for _, id := range input.LegalEntityIDs {
		if !validEntity[id] {
			return nil, fmt.Errorf("%w: legal entity %s is not in the current tenant", domain.ErrInvalid, id)
		}
	}
	// 主体级操作者只能在自己的授权主体内授权，且不能创建租户级账户。
	if !principal.PlatformAdmin && principal.DataScope != domain.DataScopeTenant {
		if input.DataScope == domain.DataScopeTenant {
			return nil, domain.ErrForbidden
		}
		for _, id := range input.LegalEntityIDs {
			if !contains(principal.LegalEntityIDs, id) {
				return nil, domain.ErrForbidden
			}
		}
	}
	if len(input.LegalEntityIDs) == 0 && input.DataScope != domain.DataScopeTenant {
		return nil, fmt.Errorf("%w: entity or self scope requires at least one legal entity", domain.ErrInvalid)
	}

	roles, err := s.roles.GetByIDs(ctx, input.RoleIDs)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.TenantID != tenantID {
			return nil, fmt.Errorf("%w: role %s belongs to another tenant", domain.ErrInvalid, role.ID)
		}
	}

	now := time.Now().UTC()
	if input.ID == "" {
		if input.AccountName == "" || len(input.Password) < 8 {
			return nil, fmt.Errorf("%w: account name and an 8+ char password are required", domain.ErrInvalid)
		}
		hash, err := s.hasher.Hash(input.Password)
		if err != nil {
			return nil, err
		}
		created, err := s.accounts.Create(ctx, domain.Account{
			AccountName: input.AccountName, PasswordHash: hash, DisplayName: input.DisplayName,
			Email: input.Email, Phone: input.Phone, Status: input.Status, TenantID: tenantID,
			DataScope: input.DataScope, LegalEntityIDs: nonNil(input.LegalEntityIDs),
			OrgIDs: nonNil(input.OrgIDs), RoleIDs: nonNil(input.RoleIDs),
			CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return nil, err
		}
		public := created.Public()
		return &public, nil
	}

	current, err := s.accounts.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if current.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}
	// 主体级操作者不能编辑授权主体之外的账户。
	if !principal.PlatformAdmin && principal.DataScope != domain.DataScopeTenant &&
		!intersects(current.LegalEntityIDs, principal.LegalEntityIDs) {
		return nil, domain.ErrForbidden
	}
	current.DisplayName = input.DisplayName
	current.Email = input.Email
	current.Phone = input.Phone
	current.Status = input.Status
	current.DataScope = input.DataScope
	current.LegalEntityIDs = nonNil(input.LegalEntityIDs)
	current.OrgIDs = nonNil(input.OrgIDs)
	current.RoleIDs = nonNil(input.RoleIDs)
	current.UpdatedAt = now
	if input.Password != "" {
		if len(input.Password) < 8 {
			return nil, fmt.Errorf("%w: password must contain at least 8 characters", domain.ErrInvalid)
		}
		hash, err := s.hasher.Hash(input.Password)
		if err != nil {
			return nil, err
		}
		current.PasswordHash = hash
	}
	updated, err := s.accounts.Update(ctx, *current)
	if err != nil {
		return nil, err
	}
	public := updated.Public()
	return &public, nil
}

// UpdateAccountStatus 修改账户启用状态，非法状态直接拒绝。
func (s *Service) UpdateAccountStatus(ctx context.Context, principal *domain.Principal, id string, status string) error {
	if status != domain.UserActive && status != domain.UserDisabled {
		return fmt.Errorf("%w: unsupported account status %q", domain.ErrInvalid, status)
	}
	account, err := s.accounts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !domain.CanAccessTenant(principal, account.TenantID) {
		return domain.ErrForbidden
	}
	return s.accounts.UpdateStatus(ctx, id, status)
}

// ListRoles 分页查询角色，并统计每个角色关联的账户数。
func (s *Service) ListRoles(ctx context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[domain.Role], error) {
	query.TenantID = s.scopeTenant(principal, query.TenantID)
	if query.TenantID == "" {
		return domain.PageResult[domain.Role]{Items: []domain.Role{}, Total: 0}, nil
	}
	result, err := s.roles.List(ctx, query)
	if err != nil {
		return domain.PageResult[domain.Role]{}, err
	}
	roleIDs := make([]string, 0, len(result.Items))
	for _, role := range result.Items {
		roleIDs = append(roleIDs, role.ID)
	}
	counts, err := s.accounts.CountByRoles(ctx, roleIDs)
	if err != nil {
		return domain.PageResult[domain.Role]{}, err
	}
	items := make([]domain.Role, 0, len(result.Items))
	for _, role := range result.Items {
		role.UserCount = counts[role.ID]
		items = append(items, role)
	}
	return domain.PageResult[domain.Role]{Items: items, Total: result.Total}, nil
}

// SaveRole 创建或更新角色，并校验权限码是否在平台权限目录中。
func (s *Service) SaveRole(ctx context.Context, principal *domain.Principal, input RoleInput) (*domain.Role, error) {
	tenantID := s.scopeTenant(principal, "")
	if tenantID == "" {
		return nil, domain.ErrForbidden
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	if input.Name == "" || input.Code == "" {
		return nil, fmt.Errorf("%w: role name and code are required", domain.ErrInvalid)
	}
	if input.DataScope == "" {
		input.DataScope = domain.DataScopeEntity
	}
	if err := domain.ValidateDataScope(input.DataScope); err != nil {
		return nil, err
	}
	if input.Status == "" {
		input.Status = domain.TenantActive
	}

	nodes, err := s.permissions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		known[node.Code] = true
	}
	for _, code := range input.PermissionCodes {
		if code != "*" && !known[code] {
			return nil, fmt.Errorf("%w: unknown permission %q", domain.ErrInvalid, code)
		}
	}

	now := time.Now().UTC()
	if input.ID == "" {
		return s.roles.Create(ctx, domain.Role{
			TenantID: tenantID, LegalEntityID: input.LegalEntityID, Code: input.Code,
			Name: input.Name, Description: input.Description, Status: input.Status,
			DataScope: input.DataScope, PermissionCodes: nonNil(input.PermissionCodes),
			CreatedAt: now, UpdatedAt: now,
		})
	}
	current, err := s.roles.GetByIDs(ctx, []string{input.ID})
	if err != nil {
		return nil, err
	}
	if len(current) == 0 {
		return nil, domain.ErrNotFound
	}
	role := current[0]
	if role.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}
	role.Name = input.Name
	role.Code = input.Code
	role.Description = input.Description
	role.Status = input.Status
	role.DataScope = input.DataScope
	role.LegalEntityID = input.LegalEntityID
	role.PermissionCodes = nonNil(input.PermissionCodes)
	role.UpdatedAt = now
	return s.roles.Update(ctx, role)
}

// ListPermissions 返回按声明顺序组装的权限树。
func (s *Service) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	nodes, err := s.permissions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return domain.BuildPermissionTree(nodes), nil
}

// SavePermission 创建或更新权限节点。
//
// 权限目录是平台级配置，只有平台运营管理员可以维护；父节点必须存在，
// 且不能把节点移动到自己的后代之下，避免形成环。
func (s *Service) SavePermission(ctx context.Context, principal *domain.Principal, input PermissionInput) (*domain.Permission, error) {
	if !principal.PlatformAdmin {
		return nil, domain.ErrForbidden
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.TrimSpace(input.Code)
	if input.Name == "" || input.Code == "" {
		return nil, fmt.Errorf("%w: permission name and code are required", domain.ErrInvalid)
	}
	if !permissionCodePattern.MatchString(input.Code) {
		return nil, fmt.Errorf("%w: permission code %q is invalid", domain.ErrInvalid, input.Code)
	}
	switch input.Type {
	case "menu", "page", "action":
	default:
		return nil, fmt.Errorf("%w: unsupported permission type %q", domain.ErrInvalid, input.Type)
	}
	switch input.Scope {
	case domain.PermissionScopePlatform, domain.PermissionScopeTenant, domain.PermissionScopeEntity:
	default:
		return nil, fmt.Errorf("%w: unsupported permission scope %q", domain.ErrInvalid, input.Scope)
	}

	nodes, err := s.permissions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		if node.Code == input.Code && node.ID != input.ID {
			return nil, fmt.Errorf("%w: permission code %q already exists", domain.ErrConflict, input.Code)
		}
	}
	if input.ParentID != "" {
		if input.ParentID == input.ID {
			return nil, fmt.Errorf("%w: permission cannot be its own parent", domain.ErrInvalid)
		}
		if !permissionHasID(nodes, input.ParentID) {
			return nil, fmt.Errorf("%w: parent permission not found", domain.ErrInvalid)
		}
		if input.ID != "" && isPermissionDescendant(nodes, input.ParentID, input.ID) {
			return nil, fmt.Errorf("%w: permission cannot move under its own descendant", domain.ErrInvalid)
		}
	}

	permission := domain.Permission{
		ID: input.ID, Name: input.Name, Code: input.Code, Type: input.Type,
		Scope: input.Scope, ParentID: input.ParentID, SortOrder: input.SortOrder,
	}
	if input.ID == "" {
		return s.permissions.Create(ctx, permission)
	}
	current, err := s.permissions.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	permission.ID = current.ID
	return s.permissions.Update(ctx, permission)
}

// ReorderPermissions 按拖拽结果重写同级权限的顺序，并返回最新权限树。
func (s *Service) ReorderPermissions(ctx context.Context, principal *domain.Principal, input PermissionReorderInput) ([]domain.Permission, error) {
	if !principal.PlatformAdmin {
		return nil, domain.ErrForbidden
	}
	if len(input.IDs) == 0 {
		return nil, fmt.Errorf("%w: permission ids are required", domain.ErrInvalid)
	}
	nodes, err := s.permissions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.Permission, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	for _, id := range input.IDs {
		node, ok := byID[id]
		if !ok {
			return nil, domain.ErrNotFound
		}
		// MillerColumns 每次只提交同一列的节点，跨父级提交视为非法请求。
		if node.ParentID != input.ParentID {
			return nil, fmt.Errorf("%w: permissions must share the same parent", domain.ErrInvalid)
		}
	}
	if err := s.permissions.Reorder(ctx, input.IDs); err != nil {
		return nil, err
	}
	updated, err := s.permissions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return domain.BuildPermissionTree(updated), nil
}

// DeletePermission 删除叶子权限；仍被角色引用或存在子节点时拒绝删除。
func (s *Service) DeletePermission(ctx context.Context, principal *domain.Principal, id string) error {
	if !principal.PlatformAdmin {
		return domain.ErrForbidden
	}
	nodes, err := s.permissions.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if node.ParentID == id {
			return fmt.Errorf("%w: remove child permissions first", domain.ErrConflict)
		}
	}
	return s.permissions.Delete(ctx, id)
}

// permissionCodePattern 限定权限码字符集，避免与路由/日志字段冲突。
var permissionCodePattern = regexp.MustCompile(`^[a-z0-9:._-]{2,64}$`)

// permissionHasID 判断权限目录中是否存在指定节点。
func permissionHasID(nodes []domain.Permission, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

// isPermissionDescendant 判断 candidate 是否位于 ancestor 的子树中。
func isPermissionDescendant(nodes []domain.Permission, candidate string, ancestor string) bool {
	byID := make(map[string]domain.Permission, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	visited := map[string]bool{}
	cursor := candidate
	for cursor != "" && !visited[cursor] {
		visited[cursor] = true
		node, ok := byID[cursor]
		if !ok {
			return false
		}
		if node.ParentID == ancestor {
			return true
		}
		cursor = node.ParentID
	}
	return false
}

// Stats 汇总当前租户的规模指标，供工作台展示。
func (s *Service) Stats(ctx context.Context, principal *domain.Principal) (map[string]int, error) {
	stats := map[string]int{"tenants": 0, "legalEntities": 0, "accounts": 0, "roles": 0}
	if principal.PlatformAdmin {
		total, err := s.tenants.Count(ctx)
		if err != nil {
			return nil, err
		}
		stats["tenants"] = int(total)
	}
	if principal.TenantID == "" {
		return stats, nil
	}
	entities, err := s.entities.List(ctx, domain.ListQuery{TenantID: principal.TenantID})
	if err != nil {
		return nil, err
	}
	stats["legalEntities"] = len(domain.VisibleEntities(principal, entities))
	// 账户数量按主体范围统计，保持与账户列表相同的可见性规则。
	accountQuery := domain.ListQuery{TenantID: principal.TenantID, Page: 1, PageSize: 1}
	if !principal.PlatformAdmin && principal.DataScope != domain.DataScopeTenant {
		accountQuery.LegalEntityIDs = principal.LegalEntityIDs
	}
	accounts, err := s.accounts.List(ctx, accountQuery)
	if err != nil {
		return nil, err
	}
	stats["accounts"] = int(accounts.Total)
	roles, err := s.roles.List(ctx, domain.ListQuery{TenantID: principal.TenantID, Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	stats["roles"] = int(roles.Total)
	if principal.PlatformAdmin {
		stats["tenants"] = maxInt(stats["tenants"], 1)
	}
	return stats, nil
}

// scopeTenant 解析请求的租户范围，非平台管理员强制收敛到自身租户。
func (s *Service) scopeTenant(principal *domain.Principal, requested string) string {
	if principal == nil {
		return ""
	}
	if !principal.PlatformAdmin {
		return principal.TenantID
	}
	if requested != "" {
		return requested
	}
	return principal.TenantID
}

// nonNil 保证下发给前端的数组字段不为 null。
func nonNil(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

// contains 判断字符串切片是否包含目标值。
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

// maxInt 返回两个整数中的较大值。
func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
