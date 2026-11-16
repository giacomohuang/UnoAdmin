package admin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/unoadmin/admin-service/internal/admin"
	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/testutil"
)

func fixtures() (*admin.Service, *testutil.UserRepo) {
	users := testutil.NewUserRepo(
		domain.User{
			ID: "user-001", Username: "lin.wei", PasswordHash: "hash:secret",
			DisplayName: "林薇", Status: domain.UserActive,
			RoleIDs: []string{"role-admin"}, TenantIDs: []string{"tenant-main"},
		},
		domain.User{
			ID: "user-002", Username: "chen.yu", DisplayName: "陈宇",
			Status: domain.UserActive, RoleIDs: []string{"role-operator"},
			TenantIDs: []string{"tenant-lab"},
		},
	)
	roles := testutil.NewRoleRepo(
		domain.Role{ID: "role-admin", Name: "平台管理员", PermissionCodes: []string{"*"}},
		domain.Role{ID: "role-operator", Name: "运营经理", PermissionCodes: []string{"user:read"}},
	)
	permissions := testutil.NewPermissionRepo(
		domain.Permission{ID: "p-user", Name: "用户管理", Code: "user:read", Type: "page", ParentID: "p-system", SortOrder: 0},
		domain.Permission{ID: "p-system", Name: "系统管理", Code: "system", Type: "menu", SortOrder: 0},
	)
	tenants := testutil.NewTenantRepo(
		domain.Tenant{ID: "tenant-main", Name: "总部空间", Code: "HQ", Status: domain.TenantActive},
		domain.Tenant{ID: "tenant-lab", Name: "创新实验室", Code: "LAB", Status: domain.TenantActive},
	)
	return admin.NewService(users, roles, permissions, tenants), users
}

func TestListUsersHydratesRoleNamesAndHidesSecrets(t *testing.T) {
	svc, _ := fixtures()

	got, err := svc.ListUsers(context.Background(), domain.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if got.Total != 2 || len(got.Items) != 2 {
		t.Fatalf("page = %+v, want 2 users", got)
	}
	if got.Items[0].RoleNames[0] != "平台管理员" {
		t.Fatalf("roleNames = %v, want hydrated names", got.Items[0].RoleNames)
	}
	if got.Items[0].PasswordHash != "" {
		t.Fatal("ListUsers() leaked the password hash")
	}
}

func TestListUsersScopesByTenant(t *testing.T) {
	svc, _ := fixtures()

	got, err := svc.ListUsers(context.Background(), domain.ListQuery{Page: 1, PageSize: 10, TenantID: "tenant-lab"})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if got.Total != 1 || got.Items[0].Username != "chen.yu" {
		t.Fatalf("page = %+v, want only chen.yu", got)
	}
}

func TestUpdateUserStatusValidatesStatus(t *testing.T) {
	svc, users := fixtures()

	if err := svc.UpdateUserStatus(context.Background(), "user-001", domain.UserDisabled); err != nil {
		t.Fatalf("UpdateUserStatus() error = %v", err)
	}
	if users.Users["user-001"].Status != domain.UserDisabled {
		t.Fatal("UpdateUserStatus() did not persist the change")
	}
	if err := svc.UpdateUserStatus(context.Background(), "user-001", "unknown"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid status error = %v, want ErrInvalid", err)
	}
}

func TestListRolesComputesUserCount(t *testing.T) {
	svc, _ := fixtures()

	got, err := svc.ListRoles(context.Background(), domain.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListRoles() error = %v", err)
	}
	for _, role := range got.Items {
		if role.UserCount != 1 {
			t.Fatalf("role %s userCount = %d, want 1", role.ID, role.UserCount)
		}
	}
}

func TestListPermissionsBuildsTree(t *testing.T) {
	svc, _ := fixtures()

	got, err := svc.ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions() error = %v", err)
	}
	if len(got) != 1 || got[0].Code != "system" || len(got[0].Children) != 1 {
		t.Fatalf("permissions = %+v, want one system root with a child", got)
	}
}
