package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/unoadmin/admin-service/internal/auth"
	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/testutil"
)

func fixtures() (*auth.Service, *testutil.TokenProvider, *testutil.RefreshStore) {
	users := testutil.NewUserRepo(
		domain.User{
			ID: "user-001", Username: "lin.wei", PasswordHash: "hash:secret",
			DisplayName: "林薇", Email: "lin.wei@example.com", Status: domain.UserActive,
			RoleIDs: []string{"role-admin"}, TenantIDs: []string{"tenant-main"},
		},
		domain.User{
			ID: "user-002", Username: "disabled", PasswordHash: "hash:secret",
			DisplayName: "禁用用户", Status: domain.UserDisabled,
			RoleIDs: []string{"role-viewer"}, TenantIDs: []string{"tenant-main"},
		},
	)
	roles := testutil.NewRoleRepo(
		domain.Role{ID: "role-admin", Name: "平台管理员", PermissionCodes: []string{"*"}},
		domain.Role{ID: "role-viewer", Name: "只读访客", PermissionCodes: []string{"user:read"}},
	)
	tenants := testutil.NewTenantRepo(
		domain.Tenant{ID: "tenant-main", Name: "总部空间", Code: "HQ", Status: domain.TenantActive},
		domain.Tenant{ID: "tenant-lab", Name: "创新实验室", Code: "LAB", Status: domain.TenantActive},
	)
	tokens := &testutil.TokenProvider{
		Claims: &domain.Claims{UserID: "user-001", TenantID: "tenant-main"},
	}
	refresh := testutil.NewRefreshStore()
	svc := auth.NewService(users, roles, tenants, tokens, testutil.PlainHasher{}, refresh, time.Hour)
	return svc, tokens, refresh
}

func TestSignInIssuesTokenWithEffectivePermissions(t *testing.T) {
	svc, tokens, refresh := fixtures()

	pair, session, err := svc.SignIn(context.Background(), "lin.wei", "secret")
	if err != nil {
		t.Fatalf("SignIn() error = %v", err)
	}
	if pair.AccessToken != "issued-token" {
		t.Fatalf("access token = %q, want issued-token", pair.AccessToken)
	}
	if pair.RefreshToken == "" || refresh.Sessions[pair.RefreshToken].UserID != "user-001" {
		t.Fatalf("refresh token not saved correctly: %q", pair.RefreshToken)
	}
	if session.User.ID != "user-001" || session.User.RoleNames[0] != "平台管理员" {
		t.Fatalf("session user = %+v, want admin with role name", session.User)
	}
	if len(session.Permissions) != 1 || session.Permissions[0] != "*" {
		t.Fatalf("permissions = %v, want [ * ]", session.Permissions)
	}
	if len(session.Tenants) != 1 || session.Tenants[0].ID != "tenant-main" {
		t.Fatalf("tenants = %v, want tenant-main", session.Tenants)
	}
	if session.CurrentTenantID != "tenant-main" {
		t.Fatalf("currentTenantId = %q, want tenant-main", session.CurrentTenantID)
	}
	if len(tokens.Issued) != 1 || tokens.Issued[0].UserID != "user-001" {
		t.Fatalf("issued claims = %+v, want user-001", tokens.Issued)
	}
}

func TestSignInRejectsWrongPasswordAndDisabledUser(t *testing.T) {
	svc, _, _ := fixtures()

	if _, _, err := svc.SignIn(context.Background(), "lin.wei", "wrong"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("wrong password error = %v, want ErrUnauthorized", err)
	}
	if _, _, err := svc.SignIn(context.Background(), "disabled", "secret"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("disabled user error = %v, want ErrForbidden", err)
	}
}

func TestRestoreReloadsPermissionsFromRepository(t *testing.T) {
	svc, _, _ := fixtures()

	session, err := svc.Restore(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if session.User.Username != "lin.wei" || len(session.Permissions) != 1 {
		t.Fatalf("restored session = %+v, want fresh admin session", session)
	}
}

func TestVerifyLoadsPermissionsFromRepository(t *testing.T) {
	svc, _, _ := fixtures()

	principal, err := svc.Verify(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if principal.UserID != "user-001" || principal.TenantID != "tenant-main" {
		t.Fatalf("principal = %+v, want user-001 / tenant-main", principal)
	}
	if len(principal.Permissions) != 1 || principal.Permissions[0] != "*" {
		t.Fatalf("permissions = %v, want wildcard loaded from repository", principal.Permissions)
	}
	if len(principal.RoleNames) != 1 || principal.RoleNames[0] != "平台管理员" {
		t.Fatalf("roleNames = %v, want 平台管理员", principal.RoleNames)
	}
}

func TestSignOutRevokesCurrentToken(t *testing.T) {
	svc, tokens, _ := fixtures()
	tokens.Claims.TokenID = "token-001"
	tokens.Claims.ExpiresAt = time.Now().Add(time.Hour)

	if err := svc.SignOut(context.Background(), "any-token", "refresh-token"); err != nil {
		t.Fatalf("SignOut() error = %v", err)
	}
}

func TestRefreshRotatesStoredToken(t *testing.T) {
	svc, _, refresh := fixtures()
	refresh.Sessions["old-refresh"] = domain.RefreshSession{UserID: "user-001", TenantID: "tenant-main"}

	pair, session, err := svc.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if pair.AccessToken != "issued-token" || pair.RefreshToken == "" {
		t.Fatalf("pair = %+v, want new tokens", pair)
	}
	if pair.RefreshToken == "old-refresh" {
		t.Fatal("Refresh() must rotate the refresh token")
	}
	if _, exists := refresh.Sessions["old-refresh"]; exists {
		t.Fatal("Refresh() did not delete the old refresh token")
	}
	if refresh.Sessions[pair.RefreshToken].TenantID != "tenant-main" {
		t.Fatalf("stored session = %+v, want tenant-main", refresh.Sessions[pair.RefreshToken])
	}
	if session.User.ID != "user-001" {
		t.Fatalf("session user = %+v, want user-001", session.User)
	}
}

func TestRefreshRejectsUnknownToken(t *testing.T) {
	svc, _, _ := fixtures()
	if _, _, err := svc.Refresh(context.Background(), "missing"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Refresh() error = %v, want ErrUnauthorized", err)
	}
}

func TestSwitchTenantRejectsUnavailableTenant(t *testing.T) {
	svc, _, _ := fixtures()

	if _, err := svc.SwitchTenant(context.Background(), "any-token", "", "tenant-lab"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SwitchTenant() error = %v, want ErrForbidden", err)
	}
	pair, err := svc.SwitchTenant(context.Background(), "any-token", "", "tenant-main")
	if err != nil || pair.AccessToken == "" {
		t.Fatalf("SwitchTenant() = %+v, %v, want a token", pair, err)
	}
}
