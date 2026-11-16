package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/unoadmin/admin-service/internal/domain"
)

// adminServiceAPI 定义基础管理处理器依赖的用例。
type adminServiceAPI interface {
	ListUsers(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.User], error)
	GetUser(ctx context.Context, id string) (domain.User, error)
	UpdateUserStatus(ctx context.Context, id string, status string) error
	ListRoles(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Role], error)
	ListPermissions(ctx context.Context) ([]domain.Permission, error)
	ListTenants(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Tenant], error)
}

// statusRequest 是用户状态变更请求体。
type statusRequest struct {
	Status string `json:"status"`
}

// handleListUsers 分页返回当前租户下的用户。
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	query := parseListQuery(r)
	if principal := currentPrincipal(r); principal != nil {
		query.TenantID = principal.TenantID
	}
	result, err := s.admin.ListUsers(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleUpdateUserStatus 更新指定用户的启用状态。
func (s *Server) handleUpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	var req statusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	if err := s.admin.UpdateUserStatus(r.Context(), r.PathValue("id"), req.Status); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleListRoles 分页返回角色及其用户数。
func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	result, err := s.admin.ListRoles(r.Context(), parseListQuery(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleListPermissions 返回权限树。
func (s *Server) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	result, err := s.admin.ListPermissions(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleListTenants 分页返回租户。
func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	result, err := s.admin.ListTenants(r.Context(), parseListQuery(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// parseListQuery 解析与前端 ListQuery 一致的分页与搜索参数。
func parseListQuery(r *http.Request) domain.ListQuery {
	values := r.URL.Query()
	return domain.ListQuery{
		Page:     positiveInt(values.Get("page"), 1),
		PageSize: positiveInt(values.Get("pageSize"), 20),
		Search:   values.Get("search"),
	}
}

// positiveInt 解析正整数查询参数，失败时回退默认值。
func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
