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
	public.POST("/auth/switch-legal-entity", s.handleSwitchLegalEntity)

	// 受保护路由：整组统一认证，再按路由声明权限码。
	secured := engine.Group("/api/v1", authenticate(s.auth))

	// 工作台指标。
	secured.GET("/stats", s.handleStats)

	// 租户（采购与计费边界）。
	secured.GET("/tenants", requirePermission("tenant:view"), s.handleListTenants)
	secured.POST("/tenants", requirePermission("tenant:edit"), s.handleCreateTenant)
	secured.PUT("/tenants/:id", requirePermission("tenant:edit"), s.handleUpdateTenant)

	// 经营主体（租户内隔离单元）。
	secured.GET("/legal-entities", requirePermission("entity:view"), s.handleListLegalEntities)
	secured.POST("/legal-entities", requirePermission("entity:edit"), s.handleCreateLegalEntity)
	secured.PUT("/legal-entities/:id", requirePermission("entity:edit"), s.handleUpdateLegalEntity)

	// 组织架构（经营主体内部结构）。
	secured.GET("/orgs", requirePermission("org:view"), s.handleListOrgs)
	secured.POST("/orgs", requirePermission("org:edit"), s.handleCreateOrg)
	secured.PUT("/orgs/:id", requirePermission("org:edit"), s.handleUpdateOrg)
	secured.DELETE("/orgs/:id", requirePermission("org:edit"), s.handleDeleteOrg)

	// 账户（管理端登录身份）。
	secured.GET("/accounts", requirePermission("account:view"), s.handleListAccounts)
	secured.PATCH("/accounts/:id/status", requirePermission("account:edit"), s.handleUpdateAccountStatus)
	secured.POST("/accounts", requirePermission("account:edit"), s.handleCreateAccount)
	secured.PUT("/accounts/:id", requirePermission("account:edit"), s.handleUpdateAccount)

	// 角色与权限。
	secured.GET("/roles", requirePermission("role:view"), s.handleListRoles)
	secured.POST("/roles", requirePermission("role:edit"), s.handleCreateRole)
	secured.PUT("/roles/:id", requirePermission("role:edit"), s.handleUpdateRole)
	secured.GET("/permissions", requirePermission("permission:view"), s.handleListPermissions)
	secured.POST("/permissions", requirePermission("permission:edit"), s.handleCreatePermission)
	secured.POST("/permissions/reorder", requirePermission("permission:edit"), s.handleReorderPermissions)
	secured.PUT("/permissions/:id", requirePermission("permission:edit"), s.handleUpdatePermission)
	secured.DELETE("/permissions/:id", requirePermission("permission:delete"), s.handleDeletePermission)

	return engine
}
