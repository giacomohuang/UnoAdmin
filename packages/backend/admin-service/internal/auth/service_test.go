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

// fixtures 构造覆盖「集团租户 + 单体租户」的认证测试装配。
func fixtures() (*auth.Service, *testutil.TokenProvider, *testutil.RefreshStore) {
	accounts := testutil.NewAccountRepo(
		domain.Account{
			ID: "acc-platform", AccountName: "lin.wei", PasswordHash: "hash:secret",
			DisplayName: "林薇", Status: domain.UserActive, PlatformAdmin: true,
			DataScope: domain.DataScopeTenant, RoleIDs: []string{"role-platform"},
		},
		domain.Account{
			ID: "acc-group", AccountName: "chen.yu", PasswordHash: "hash:secret",
			DisplayName: "陈宇", Status: domain.UserActive, TenantID: "tenant-holding",
			DataScope: domain.DataScopeTenant, RoleIDs: []string{"role-group-admin"},
			LegalEntityIDs: []string{"le-hq"},
		},
		domain.Account{
			ID: "acc-east", AccountName: "zhao.ming", PasswordHash: "hash:secret",
			DisplayName: "赵明", Status: domain.UserActive, TenantID: "tenant-holding",
			DataScope: domain.DataScopeEntity, RoleIDs: []string{"role-entity-admin"},
			LegalEntityIDs: []string{"le-east"},
		},
		domain.Account{
			ID: "acc-disabled", AccountName: "disabled", PasswordHash: "hash:secret",
			DisplayName: "禁用账户", Status: domain.UserDisabled, TenantID: "tenant-holding",
		},
	)
	roles := testutil.NewRoleRepo(
		domain.Role{ID: "role-platform", Name: "平台运营管理员", PermissionCodes: []string{"*"}, DataScope: domain.DataScopeTenant},
		domain.Role{ID: "role-group-admin", Name: "集团管理员", PermissionCodes: []string{"account:view", "entity:view"}, DataScope: domain.DataScopeTenant},
		domain.Role{ID: "role-entity-admin", Name: "主体管理员", PermissionCodes: []string{"org:view"}, DataScope: domain.DataScopeEntity},
	)
	tenants := testutil.NewTenantRepo(
		domain.Tenant{ID: "tenant-holding", Name: "汉唐控股集团", Code: "HANTANG", Mode: domain.TenantModeGroup, Status: domain.TenantActive},
		domain.Tenant{ID: "tenant-mingyuan", Name: "明远科技", Code: "MINGYUAN", Mode: domain.TenantModeStandalone, Status: domain.TenantActive},
	)
	entities := testutil.NewLegalEntityRepo(
		domain.LegalEntity{ID: "le-hq", TenantID: "tenant-holding", Type: domain.LegalEntityTypeGroup, Name: "集团总部", Status: domain.TenantActive},
		domain.LegalEntity{ID: "le-east", TenantID: "tenant-holding", ParentID: "le-hq", Type: domain.LegalEntityTypeCompany, Name: "华东公司", Status: domain.TenantActive},
		domain.LegalEntity{ID: "le-south", TenantID: "tenant-holding", ParentID: "le-hq", Type: domain.LegalEntityTypeCompany, Name: "华南公司", Status: domain.TenantActive},
		domain.LegalEntity{ID: "le-mingyuan", TenantID: "tenant-mingyuan", Type: domain.LegalEntityTypeCompany, Name: "明远科技", Status: domain.TenantActive},
	)
	tokens := &testutil.TokenProvider{
		Claims: &domain.Claims{AccountID: "acc-group", TenantID: "tenant-holding", LegalEntityID: "le-hq"},
	}
	refresh := testutil.NewRefreshStore()
	svc := auth.NewService(accounts, roles, tenants, entities, tokens, testutil.PlainHasher{}, refresh, time.Hour)
	return svc, tokens, refresh
}

func TestSignInIssuesTokenWithTenantAndEntity(t *testing.T) {
	svc, tokens, refresh := fixtures()

	pair, session, err := svc.SignIn(context.Background(), "chen.yu", "secret")
	if err != nil {
		t.Fatalf("SignIn() error = %v", err)
	}
	if pair.AccessToken != "issued-token" || pair.RefreshToken == "" {
		t.Fatalf("pair = %+v, want issued tokens", pair)
	}
	if refresh.Sessions[pair.RefreshToken].AccountID != "acc-group" {
		t.Fatalf("refresh session = %+v, want acc-group", refresh.Sessions[pair.RefreshToken])
	}
	if session.Account.ID != "acc-group" || session.Account.RoleNames[0] != "集团管理员" {
		t.Fatalf("session account = %+v, want group admin with role name", session.Account)
	}
	if len(session.Tenants) != 1 || session.Tenants[0].ID != "tenant-holding" {
		t.Fatalf("tenants = %+v, want only the home tenant", session.Tenants)
	}
	if session.CurrentTenantID != "tenant-holding" {
		t.Fatalf("currentTenantId = %q, want tenant-holding", session.CurrentTenantID)
	}
	// 集团级数据范围可看到集团下全部经营主体。
	if len(session.LegalEntities) != 3 {
		t.Fatalf("legal entities = %+v, want three visible entities", session.LegalEntities)
	}
	if session.CurrentLegalEntityID != "le-hq" {
		t.Fatalf("currentLegalEntityId = %q, want the authorized le-hq", session.CurrentLegalEntityID)
	}
	if session.DataScope != domain.DataScopeTenant {
		t.Fatalf("dataScope = %q, want tenant", session.DataScope)
	}
	if len(tokens.Issued) != 1 || tokens.Issued[0].LegalEntityID != "le-hq" {
		t.Fatalf("issued claims = %+v, want entity le-hq", tokens.Issued)
	}
}

func TestSignInRejectsWrongPasswordAndDisabledAccount(t *testing.T) {
	svc, _, _ := fixtures()

	if _, _, err := svc.SignIn(context.Background(), "chen.yu", "wrong"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("wrong password error = %v, want ErrUnauthorized", err)
	}
	if _, _, err := svc.SignIn(context.Background(), "disabled", "secret"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("disabled account error = %v, want ErrForbidden", err)
	}
}

func TestEntityScopedAccountOnlySeesAuthorizedEntities(t *testing.T) {
	svc, _, _ := fixtures()

	_, session, err := svc.SignIn(context.Background(), "zhao.ming", "secret")
	if err != nil {
		t.Fatalf("SignIn() error = %v", err)
	}
	if len(session.LegalEntities) != 1 || session.LegalEntities[0].ID != "le-east" {
		t.Fatalf("legal entities = %+v, want only le-east", session.LegalEntities)
	}
	if session.CurrentLegalEntityID != "le-east" {
		t.Fatalf("currentLegalEntityId = %q, want le-east", session.CurrentLegalEntityID)
	}
}

func TestRestoreReloadsPermissionsAndKeepsContext(t *testing.T) {
	svc, tokens, _ := fixtures()
	tokens.Claims.LegalEntityID = "le-east"

	session, err := svc.Restore(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if session.Account.AccountName != "chen.yu" || len(session.Permissions) != 2 {
		t.Fatalf("restored session = %+v, want fresh group admin session", session)
	}
	// 令牌中的主体在可见范围内时应被沿用。
	if session.CurrentLegalEntityID != "le-east" {
		t.Fatalf("currentLegalEntityId = %q, want le-east from token", session.CurrentLegalEntityID)
	}
}

func TestVerifyLoadsScopeFromRepository(t *testing.T) {
	svc, tokens, _ := fixtures()
	tokens.Claims.AccountID = "acc-east"
	tokens.Claims.LegalEntityID = "le-east"

	principal, err := svc.Verify(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if principal.AccountID != "acc-east" || principal.TenantID != "tenant-holding" || principal.LegalEntityID != "le-east" {
		t.Fatalf("principal = %+v, want acc-east / tenant-holding / le-east", principal)
	}
	if principal.DataScope != domain.DataScopeEntity {
		t.Fatalf("dataScope = %q, want entity", principal.DataScope)
	}
	if len(principal.LegalEntityIDs) != 1 || principal.LegalEntityIDs[0] != "le-east" {
		t.Fatalf("legalEntityIds = %v, want [le-east]", principal.LegalEntityIDs)
	}
	if len(principal.Permissions) != 1 || principal.Permissions[0] != "org:view" {
		t.Fatalf("permissions = %v, want repository permissions", principal.Permissions)
	}
}

func TestRestoreRebuildsEntitiesForTokenTenant(t *testing.T) {
	svc, tokens, _ := fixtures()
	// 平台管理员在令牌中切到单体租户，主体列表必须按新租户重算。
	tokens.Claims = &domain.Claims{AccountID: "acc-platform", TenantID: "tenant-mingyuan", LegalEntityID: "le-mingyuan"}

	session, err := svc.Restore(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if session.CurrentTenantID != "tenant-mingyuan" {
		t.Fatalf("currentTenantId = %q, want tenant-mingyuan", session.CurrentTenantID)
	}
	if len(session.LegalEntities) != 1 || session.LegalEntities[0].ID != "le-mingyuan" {
		t.Fatalf("legal entities = %+v, want only the entities of the token tenant", session.LegalEntities)
	}
	if session.CurrentLegalEntityID != "le-mingyuan" {
		t.Fatalf("currentLegalEntityId = %q, want le-mingyuan", session.CurrentLegalEntityID)
	}
}

func TestRefreshRebuildsEntitiesForStoredTenant(t *testing.T) {
	svc, _, refresh := fixtures()
	refresh.Sessions["platform-refresh"] = domain.RefreshSession{AccountID: "acc-platform", TenantID: "tenant-mingyuan", LegalEntityID: "le-mingyuan"}

	_, session, err := svc.Refresh(context.Background(), "platform-refresh")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if session.CurrentTenantID != "tenant-mingyuan" || len(session.LegalEntities) != 1 {
		t.Fatalf("session = %+v, want mingyuan with one entity", session)
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

func TestRefreshRotatesStoredTokenAndKeepsEntity(t *testing.T) {
	svc, _, refresh := fixtures()
	refresh.Sessions["old-refresh"] = domain.RefreshSession{AccountID: "acc-group", TenantID: "tenant-holding", LegalEntityID: "le-east"}

	pair, session, err := svc.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if pair.AccessToken != "issued-token" || pair.RefreshToken == "" || pair.RefreshToken == "old-refresh" {
		t.Fatalf("pair = %+v, want rotated tokens", pair)
	}
	if _, exists := refresh.Sessions["old-refresh"]; exists {
		t.Fatal("Refresh() did not delete the old refresh token")
	}
	stored := refresh.Sessions[pair.RefreshToken]
	if stored.TenantID != "tenant-holding" || stored.LegalEntityID != "le-east" {
		t.Fatalf("stored session = %+v, want tenant and entity preserved", stored)
	}
	if session.CurrentLegalEntityID != "le-east" {
		t.Fatalf("session entity = %q, want le-east", session.CurrentLegalEntityID)
	}
}

func TestSwitchTenantEnforcesPurchaseBoundary(t *testing.T) {
	svc, _, _ := fixtures()

	// 普通租户账户不能切换到其它租户。
	if _, err := svc.SwitchTenant(context.Background(), "any-token", "", "tenant-mingyuan"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SwitchTenant(other tenant) error = %v, want ErrForbidden", err)
	}
	pair, err := svc.SwitchTenant(context.Background(), "any-token", "", "tenant-holding")
	if err != nil || pair.AccessToken == "" {
		t.Fatalf("SwitchTenant(home tenant) = %+v, %v, want a token", pair, err)
	}

	// 平台管理员可跨租户切换。
	svc, tokens, _ := fixtures()
	tokens.Claims.AccountID = "acc-platform"
	tokens.Claims.TenantID = "tenant-holding"
	if _, err := svc.SwitchTenant(context.Background(), "any-token", "", "tenant-mingyuan"); err != nil {
		t.Fatalf("platform SwitchTenant() error = %v", err)
	}
}

func TestSwitchLegalEntityEnforcesEntityScope(t *testing.T) {
	svc, tokens, _ := fixtures()
	tokens.Claims.AccountID = "acc-east"
	tokens.Claims.LegalEntityID = "le-east"

	if _, err := svc.SwitchLegalEntity(context.Background(), "any-token", "", "le-south"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SwitchLegalEntity(unauthorized) error = %v, want ErrForbidden", err)
	}
	pair, err := svc.SwitchLegalEntity(context.Background(), "any-token", "", "le-east")
	if err != nil || pair.AccessToken == "" {
		t.Fatalf("SwitchLegalEntity(authorized) = %+v, %v, want a token", pair, err)
	}
}
