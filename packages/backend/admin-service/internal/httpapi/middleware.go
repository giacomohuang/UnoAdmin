package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/unoadmin/admin-service/internal/domain"
)

// principalContextKey 是调用主体通过认证后写入 Gin 上下文的键。
const principalContextKey = "principal"

// authService 是中间件需要的令牌校验能力。
type authService interface {
	Verify(ctx context.Context, raw string) (*domain.Principal, error)
}

// cors 处理跨域预检与响应头，允许前端携带 Authorization、X-Tenant-Id 与 X-Legal-Entity-Id。
func cors(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-Id, X-Legal-Entity-Id")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			// 预检直接结束，不进入路由与鉴权。
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// authenticate 校验 Bearer 令牌并把主体写入 Gin 上下文。
func authenticate(service authService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			abortWithError(c, domain.ErrUnauthorized)
			return
		}
		principal, err := service.Verify(c.Request.Context(), strings.TrimPrefix(raw, "Bearer "))
		if err != nil {
			abortWithError(c, err)
			return
		}

		// X-Tenant-Id / X-Legal-Entity-Id 必须与令牌中的上下文一致，
		// 避免过期请求头导致跨租户或跨主体越权查询。
		if requested := c.GetHeader("X-Tenant-Id"); requested != "" && requested != principal.TenantID {
			abortWithError(c, domain.ErrForbidden)
			return
		}
		if requested := c.GetHeader("X-Legal-Entity-Id"); requested != "" &&
			requested != principal.LegalEntityID && !domain.CanAccessEntity(principal, requested) {
			abortWithError(c, domain.ErrForbidden)
			return
		}
		c.Set(principalContextKey, principal)
		c.Next()
	}
}

// requirePermission 在已认证主体上执行权限码校验。
func requirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := currentPrincipal(c)
		if !ok {
			abortWithError(c, domain.ErrUnauthorized)
			return
		}
		if !domain.Can(principal.Permissions, code) {
			abortWithError(c, domain.ErrForbidden)
			return
		}
		c.Next()
	}
}

// currentPrincipal 从 Gin 上下文读取调用主体。
func currentPrincipal(c *gin.Context) (*domain.Principal, bool) {
	value, ok := c.Get(principalContextKey)
	if !ok {
		return nil, false
	}
	principal, ok := value.(*domain.Principal)
	return principal, ok
}
