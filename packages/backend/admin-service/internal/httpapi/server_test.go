package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	return domain.TokenPair{AccessToken: "new-token", RefreshToken: "new-refresh"}, nil
}

// fakeAdmin 是管理用例的测试替身，记录最近一次查询参数。
type fakeAdmin struct {
	lastQuery domain.ListQuery
	err       error
}

func (f *fakeAdmin) ListUsers(_ context.Context, query domain.ListQuery) (domain.PageResult[domain.User], error) {
	f.lastQuery = query
	return domain.PageResult[domain.User]{Items: []domain.User{{ID: "user-001", Username: "lin.wei"}}, Total: 1}, f.err
}

func (f *fakeAdmin) GetUser(context.Context, string) (domain.User, error) {
	return domain.User{ID: "user-001"}, nil
}

func (f *fakeAdmin) UpdateUserStatus(context.Context, string, string) error { return f.err }

func (f *fakeAdmin) ListRoles(context.Context, domain.ListQuery) (domain.PageResult[domain.Role], error) {
	return domain.PageResult[domain.Role]{}, f.err
}

func (f *fakeAdmin) ListPermissions(context.Context) ([]domain.Permission, error) {
	return []domain.Permission{}, f.err
}

func (f *fakeAdmin) ListTenants(context.Context, domain.ListQuery) (domain.PageResult[domain.Tenant], error) {
	return domain.PageResult[domain.Tenant]{}, f.err
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
		User: domain.User{ID: "user-001", Username: "lin.wei"}, Permissions: []string{"*"},
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
	if body["currentTenantId"] == nil {
		t.Fatalf("session fields are missing: %v", body)
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

func TestProtectedRouteRequiresBearer(t *testing.T) {
	// 受保护路由注册在同一个认证分组下，逐条确认没有被漏挂鉴权。
	for _, path := range []string{"/api/v1/users", "/api/v1/roles", "/api/v1/permissions", "/api/v1/tenants"} {
		rec := httptest.NewRecorder()
		handler(&fakeAuth{}, &fakeAdmin{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want 401", path, rec.Code)
		}
	}
}

func TestProtectedRouteRejectsMissingPermission(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{
		UserID: "user-001", TenantID: "tenant-main", Permissions: []string{"role:read"},
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler(auth, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestListUsersScopesByPrincipalTenant(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{
		UserID: "user-001", TenantID: "tenant-lab", Permissions: []string{"user:read"},
	}}
	admin := &fakeAdmin{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=2&pageSize=5&search=lin", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler(auth, admin).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if admin.lastQuery.TenantID != "tenant-lab" || admin.lastQuery.Page != 2 || admin.lastQuery.PageSize != 5 || admin.lastQuery.Search != "lin" {
		t.Fatalf("query = %+v, want tenant-lab / page 2 / size 5 / lin", admin.lastQuery)
	}
}

func TestTenantHeaderMismatchIsForbidden(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{
		UserID: "user-001", TenantID: "tenant-main", Permissions: []string{"user:read"},
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("X-Tenant-Id", "tenant-lab")
	rec := httptest.NewRecorder()
	handler(auth, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestUpdateStatusMapsInvalidError(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{UserID: "user-001", Permissions: []string{"user:write"}}}
	admin := &fakeAdmin{err: domain.ErrInvalid}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/user-001/status", strings.NewReader(`{"status":"unknown"}`))
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler(auth, admin).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler(&fakeAuth{}, &fakeAdmin{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("missing CORS header: %v", rec.Header())
	}
}

func TestWriteErrorFallsBackToInternalError(t *testing.T) {
	auth := &fakeAuth{principal: &domain.Principal{UserID: "user-001", Permissions: []string{"user:read"}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	httpapi.NewServer(auth, &fakeAdmin{err: errors.New("boom")}, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
