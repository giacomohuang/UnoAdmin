package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/unoadmin/admin-service/internal/domain"
)

// principalContextKey 用于在请求上下文中传递已校验的调用主体。
type principalContextKey struct{}

// authService 是中间件需要的令牌校验能力。
type authService interface {
	Verify(ctx context.Context, raw string) (*domain.Principal, error)
}

// cors 处理跨域预检与响应头，允许前端携带 Authorization 与 X-Tenant-Id。
func cors(origins []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, origin := range origins {
		allowed[origin] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-Id")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authenticate 校验 Bearer 令牌并把主体写入请求上下文。
func authenticate(service authService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, domain.ErrUnauthorized)
			return
		}
		principal, err := service.Verify(r.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeError(w, err)
			return
		}

		// X-Tenant-Id 必须与令牌中的租户一致，避免过期租户头导致越权查询。
		if requested := r.Header.Get("X-Tenant-Id"); requested != "" && requested != principal.TenantID {
			writeError(w, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalContextKey{}, principal)))
	})
}

// requirePermission 在已认证主体上执行权限码校验。
func requirePermission(code string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := r.Context().Value(principalContextKey{}).(*domain.Principal)
		if !ok {
			writeError(w, domain.ErrUnauthorized)
			return
		}
		if !domain.Can(principal.Permissions, code) {
			writeError(w, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// currentPrincipal 从请求上下文读取调用主体。
func currentPrincipal(r *http.Request) *domain.Principal {
	principal, _ := r.Context().Value(principalContextKey{}).(*domain.Principal)
	return principal
}

// guard 组合认证与权限校验中间件。
func guard(service authService, permission string, next http.Handler) http.Handler {
	return authenticate(service, requirePermission(permission, next))
}
