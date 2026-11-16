// Package admin 实现基础管理页面对应的查询与状态变更用例。
package admin

import (
	"context"
	"fmt"

	"github.com/unoadmin/admin-service/internal/domain"
)

// Service 编排用户、角色、权限、租户的查询逻辑。
type Service struct {
	users       domain.UserRepository
	roles       domain.RoleRepository
	permissions domain.PermissionRepository
	tenants     domain.TenantRepository
}

// NewService 构造基础管理服务。
func NewService(
	users domain.UserRepository,
	roles domain.RoleRepository,
	permissions domain.PermissionRepository,
	tenants domain.TenantRepository,
) *Service {
	return &Service{users: users, roles: roles, permissions: permissions, tenants: tenants}
}

// ListUsers 分页查询用户，并将角色 ID 转成前端需要的角色名。
func (s *Service) ListUsers(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.User], error) {
	result, err := s.users.List(ctx, query)
	if err != nil {
		return domain.PageResult[domain.User]{}, err
	}

	roleIDs := make([]string, 0)
	for _, user := range result.Items {
		roleIDs = append(roleIDs, user.RoleIDs...)
	}
	roles, err := s.roles.GetByIDs(ctx, roleIDs)
	if err != nil {
		return domain.PageResult[domain.User]{}, err
	}
	roleNames := make(map[string]string, len(roles))
	for _, role := range roles {
		roleNames[role.ID] = role.Name
	}

	items := make([]domain.User, 0, len(result.Items))
	for _, user := range result.Items {
		public := user.Public()
		public.RoleNames = make([]string, 0, len(user.RoleIDs))
		for _, id := range user.RoleIDs {
			if name, ok := roleNames[id]; ok {
				public.RoleNames = append(public.RoleNames, name)
			}
		}
		items = append(items, public)
	}
	return domain.PageResult[domain.User]{Items: items, Total: result.Total}, nil
}

// GetUser 查询单个用户，供 gRPC 与其它内部调用使用。
func (s *Service) GetUser(ctx context.Context, id string) (domain.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	roles, err := s.roles.GetByIDs(ctx, user.RoleIDs)
	if err != nil {
		return domain.User{}, err
	}
	public := user.Public()
	public.RoleNames = make([]string, 0, len(roles))
	for _, role := range roles {
		public.RoleNames = append(public.RoleNames, role.Name)
	}
	return public, nil
}

// UpdateUserStatus 修改用户启用状态，非法状态直接拒绝。
func (s *Service) UpdateUserStatus(ctx context.Context, id string, status string) error {
	if status != domain.UserActive && status != domain.UserDisabled {
		return fmt.Errorf("%w: unsupported user status %q", domain.ErrInvalid, status)
	}
	return s.users.UpdateStatus(ctx, id, status)
}

// ListRoles 分页查询角色，并统计每个角色关联的用户数。
func (s *Service) ListRoles(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Role], error) {
	result, err := s.roles.List(ctx, query)
	if err != nil {
		return domain.PageResult[domain.Role]{}, err
	}
	roleIDs := make([]string, 0, len(result.Items))
	for _, role := range result.Items {
		roleIDs = append(roleIDs, role.ID)
	}
	counts, err := s.users.CountByRoles(ctx, roleIDs)
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

// ListPermissions 返回按声明顺序组装的权限树。
func (s *Service) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	nodes, err := s.permissions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return domain.BuildPermissionTree(nodes), nil
}

// ListTenants 分页查询租户。
func (s *Service) ListTenants(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Tenant], error) {
	return s.tenants.List(ctx, query)
}
