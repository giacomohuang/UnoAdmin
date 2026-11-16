// Package httpapi 暴露供前端 SPA 调用的 REST API。
package httpapi

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Server 持有 REST 层依赖的用例，处理器以方法形式挂载。
type Server struct {
	auth  authServiceAPI
	admin adminServiceAPI
}

// NewServer 构造并注册全部路由，返回可交给 http.Server 的 Gin 引擎。
func NewServer(auth authServiceAPI, admin adminServiceAPI, origins []string) *gin.Engine {
	// 默认关闭 Gin 的路由转储与调试日志；排查问题时用 GIN_MODE=debug 覆盖。
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{auth: auth, admin: admin}
	engine := gin.New()
	engine.Use(gin.Recovery(), cors(origins))

	// 健康检查不挂在 /api/v1 下，也不做鉴权。
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 公开路由：登录与会话管理。
	public := engine.Group("/api/v1")
	public.POST("/auth/login", s.handleLogin)
	// 会话路由自行校验 Bearer 令牌，因为注销需要令牌 ID 执行吊销。
	public.POST("/auth/restore", s.handleRestore)
	public.POST("/auth/refresh", s.handleRefresh)
	public.POST("/auth/logout", s.handleLogout)
	public.POST("/auth/switch-tenant", s.handleSwitchTenant)

	// 受保护路由：整组统一认证，再按路由声明权限码。
	secured := engine.Group("/api/v1", authenticate(s.auth))
	secured.GET("/users", requirePermission("user:read"), s.handleListUsers)
	secured.PATCH("/users/:id/status", requirePermission("user:write"), s.handleUpdateUserStatus)
	secured.GET("/roles", requirePermission("role:read"), s.handleListRoles)
	secured.GET("/permissions", requirePermission(""), s.handleListPermissions)
	secured.GET("/tenants", requirePermission("tenant:read"), s.handleListTenants)
	return engine
}
