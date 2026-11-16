// Package httpapi 暴露供前端 SPA 调用的 REST API。
package httpapi

import "net/http"

// Server 持有 REST 层依赖的用例，处理器以方法形式挂载。
type Server struct {
	auth  authServiceAPI
	admin adminServiceAPI
}

// NewServer 构造并注册全部路由，返回可直接交给 http.Server 的处理器。
func NewServer(auth authServiceAPI, admin adminServiceAPI, origins []string) http.Handler {
	s := &Server{auth: auth, admin: admin}
	mux := http.NewServeMux()

	// 公开路由：登录与健康检查。
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// 会话路由自行校验 Bearer 令牌，因为注销需要令牌 ID 执行吊销。
	mux.HandleFunc("POST /api/v1/auth/restore", s.handleRestore)
	mux.HandleFunc("POST /api/v1/auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/switch-tenant", s.handleSwitchTenant)

	// 受保护路由：权限码与前端模块声明保持一致。
	mux.Handle("GET /api/v1/users", guard(auth, "user:read", http.HandlerFunc(s.handleListUsers)))
	mux.Handle("PATCH /api/v1/users/{id}/status", guard(auth, "user:write", http.HandlerFunc(s.handleUpdateUserStatus)))
	mux.Handle("GET /api/v1/roles", guard(auth, "role:read", http.HandlerFunc(s.handleListRoles)))
	mux.Handle("GET /api/v1/permissions", guard(auth, "", http.HandlerFunc(s.handleListPermissions)))
	mux.Handle("GET /api/v1/tenants", guard(auth, "tenant:read", http.HandlerFunc(s.handleListTenants)))

	return cors(origins, mux)
}
