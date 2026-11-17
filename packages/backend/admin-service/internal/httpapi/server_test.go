package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unoadmin/admin-service/internal/admin"
	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/httpapi"
)

// fakeAuth 是认证用例的测试替身。
type fakeAuth struct {
	principal *domain.Principal
	verifyErr error
	session   domain.AuthSession
}

func (f *fakeAuth) Verify(context.Context, string) (*domain.Principal, error) {
	return f.principal, f.verifyErr
}

func (f *fakeAuth) SignIn(context.Context, string, string) (domain.TokenPair, domain.AuthSession, error) {
	return domain.TokenPair{AccessToken: "issued-token", RefreshToken: "issued-refresh"}, f.session, nil
}

func (f *fakeAuth) Refresh(context.Context, string) (domain.TokenPair, domain.AuthSession, error) {
	return domain.TokenPair{AccessToken: "new-token", RefreshToken: "new-refresh"}, f.session, nil
}

func (f *fakeAuth) Restore(context.Context, string) (domain.AuthSession, error) {
	return f.session, nil
}

func (f *fakeAuth) SignOut(context.Context, string, string) error { return nil }

func (f *fakeAuth) SwitchTenant(context.Context, string, string, string) (domain.TokenPair, error) {
	return domain.TokenPair{AccessToken: "tenant-token", RefreshToken: "tenant-refresh"}, nil
}

func (f *fakeAuth) SwitchLegalEntity(context.Context, string, string, string) (domain.TokenPair, error) {
	return domain.TokenPair{AccessToken: "entity-token", RefreshToken: "entity-refresh"}, nil
}

// fakeAdmin 是管理用例的测试替身，记录最近一次查询参数与调用主体。
type fakeAdmin struct {
	lastQuery     domain.ListQuery
	lastPrincipal *domain.Principal
	err           error
}

func (f *fakeAdmin) ListAccounts(_ context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[domain.Account], error) {
	f.lastQuery, f.lastPrincipal = query, principal
	return domain.PageResult[domain.Account]{Items: []domain.Account{{ID: "acc-001", Username: "lin.wei"}}, Total: 1}, f.err
}

func (f *fakeAdmin) SaveAccount(context.Context, *domain.Principal, admin.AccountInput) (*domain.Account, error) {
	return &domain.Account{ID: "acc-001"}, f.err
}

func (f *fakeAdmin) UpdateAccountStatus(context.Context, *domain.Principal, string, string) error {
	return f.err
}

func (f *fakeAdmin) ListLegalEntities(context.Context, *domain.Principal, domain.ListQuery) ([]domain.LegalEntity, error) {
	return []domain.LegalEntity{}, f.err
}

func (f *fakeAdmin) SaveLegalEntity(context.Context, *domain.Principal, admin.LegalEntityInput) (*domain.LegalEntity, error) {
	return &domain.LegalEntity{ID: "le-001"}, f.err
}

func (f *fakeAdmin) ListOrgs(context.Context, *domain.Principal, string) ([]domain.OrgUnit, error) {
	return []domain.OrgUnit{}, f.err
}

func (f *fakeAdmin) SaveOrg(context.Context, *domain.Principal, admin.OrgInput) (*domain.OrgUnit, error) {
	return &domain.OrgUnit{ID: "org-001"}, f.err
}

func (f *fakeAdmin) ListRoles(_ context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[domain.Role], error) {
	f.lastQuery, f.lastPrincipal = query, principal
	return domain.PageResult[domain.Role]{}, f.err
}

func (f *fakeAdmin) SaveRole(context.Context, *domain.Principal, admin.RoleInput) (*domain.Role, error) {
	return &domain.Role{ID: "role-001"}, f.err
}

func (f *fakeAdmin) ListPermissions(context.Context) ([]domain.Permission, error) {
	return []domain.Permission{}, f.err
}

func (f *fakeAdmin) SavePermission(context.Context, *domain.Principal, admin.PermissionInput) (*domain.Permission, error) {
	return &domain.Permission{ID: "perm-001"}, f.err
}

func (f *fakeAdmin) ReorderPermissions(context.Context, *domain.Principal, admin.PermissionReorderInput) ([]domain.Permission, error) {
	return []domain.Permission{}, f.err
}

func (f *fakeAdmin) DeletePermission(context.Context, *domain.Principal, string) error {
	return f.err
}

func (f *fakeAdmin) ListTenants(_ context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[admin.TenantOverview], error) {
	f.lastQuery, f.lastPrincipal = query, principal
	return domain.PageResult[admin.TenantOverview]{}, f.err
}

func (f *fakeAdmin) SaveTenant(context.Context, *domain.Principal, admin.TenantInput) (*admin.TenantOverview, error) {
	return &admin.TenantOverview{Tenant: domain.Tenant{ID: "tenant-001"}}, f.err
}

func (f *fakeAdmin) Stats(context.Context, *domain.Principal) (map[string]int, error) {
	return map[string]int{"accounts": 1}, f.err
}

func handler(auth *fakeAuth, admin *fakeAdmin) http.Handler {
	return httpapi.NewServer(auth, admin, []string{"http://localhost:5173"})
}

func TestHealthzIsPublic(t *testing.T) {
	rec := httptest.NewRecorder()
	handler(&fakeAuth{}, &fakeAdmin{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestLoginReturnsTokenAndSession(t *testing.T) {
	auth := &fakeAuth{session: domain.AuthSession{
		Account:     domain.Account{ID: "acc-001", Username: "lin.wei"},
		Permissions: []string{"*"}, CurrentTenantID: "tenant-holding", CurrentLegalEntityID: "le-hq",
		LegalEntities: []domain.LegalEntity{{ID: "le-hq", Name: "集团总部"}},
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"lin.wei","password":"secret"}`))
	handler(auth, &fakeAdmin{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["token"] != "issued-token" {
		t.Fatalf("token = %v, want issued-token", body["token"])
	}
	if body["currentTenantId"] == nil || body["currentLegalEntityId"] == nil || body["legalEntities"] == nil {
		t.Fatalf("session context fields are missing: %v", body)
	}
}

func TestRefreshReturnsNewTokenPair(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refreshToken":"old-refresh"}`))
	handler(&fakeAuth{}, &fakeAdmin{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body domain.TokenPair
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.AccessToken != "new-token" || body.RefreshToken != "new-refresh" {
		t.Fatalf("pair = %+v, want new tokens", body)
	}
}

func TestSwitchLegalEntityReturnsScopedToken(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{AccountID: "acc-001", TenantID: "tenant-holding", LegalEntityID: "le-hq", Permissions: []string{"*"}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/switch-legal-entity", strings.NewReader(`{"legalEntityId":"le-east"}`))
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler(auth, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRouteRequiresBearer(t *testing.T) {
	// 受保护路由注册在同一个认证分组下，逐条确认没有被漏挂鉴权。
	for _, path := range []string{
		"/api/v1/accounts", "/api/v1/legal-entities", "/api/v1/orgs",
		"/api/v1/roles", "/api/v1/permissions", "/api/v1/tenants", "/api/v1/stats",
	} {
		rec := httptest.NewRecorder()
		handler(&fakeAuth{}, &fakeAdmin{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want 401", path, rec.Code)
		}
	}
}

func TestProtectedRouteRejectsMissingPermission(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{
		AccountID: "acc-001", TenantID: "tenant-holding", Permissions: []string{"role:view"},
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler(auth, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestListAccountsPassesPrincipalAndQuery(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{
		AccountID: "acc-001", TenantID: "tenant-holding", Permissions: []string{"account:view"},
	}}
	admin := &fakeAdmin{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts?page=2&pageSize=5&search=lin", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler(auth, admin).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if admin.lastQuery.Page != 2 || admin.lastQuery.PageSize != 5 || admin.lastQuery.Search != "lin" {
		t.Fatalf("query = %+v, want page 2 / size 5 / lin", admin.lastQuery)
	}
	if admin.lastPrincipal == nil || admin.lastPrincipal.TenantID != "tenant-holding" {
		t.Fatalf("principal = %+v, want tenant-holding", admin.lastPrincipal)
	}
}

func TestTenantHeaderMismatchIsForbidden(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{
		AccountID: "acc-001", TenantID: "tenant-holding", Permissions: []string{"account:view"},
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("X-Tenant-Id", "tenant-mingyuan")
	rec := httptest.NewRecorder()
	handler(auth, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestLegalEntityHeaderOutsideScopeIsForbidden(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{
		AccountID: "acc-east", TenantID: "tenant-holding", LegalEntityID: "le-east",
		DataScope: domain.DataScopeEntity, LegalEntityIDs: []string{"le-east"},
		Permissions: []string{"org:view"},
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs?legalEntityId=le-south", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("X-Legal-Entity-Id", "le-south")
	rec := httptest.NewRecorder()
	handler(auth, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestUpdateStatusMapsInvalidError(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{AccountID: "acc-001", Permissions: []string{"account:update"}}}
	admin := &fakeAdmin{err: domain.ErrInvalid}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/accounts/acc-001/status", strings.NewReader(`{"status":"unknown"}`))
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler(auth, admin).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/accounts", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler(&fakeAuth{}, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("missing CORS header: %v", rec.Header())
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "X-Legal-Entity-Id") {
		t.Fatalf("missing entity header in CORS: %v", rec.Header())
	}
}

func TestWriteErrorFallsBackToInternalError(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{AccountID: "acc-001", Permissions: []string{"account:view"}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	httpapi.NewServer(auth, &fakeAdmin{err: errors.New("boom")}, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
