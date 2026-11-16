package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

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
func (s *Server) handleListUsers(c *gin.Context) {
	query := parseListQuery(c)
	// 租户边界以令牌为准，忽略前端传入的租户条件。
	if principal, ok := currentPrincipal(c); ok {
		query.TenantID = principal.TenantID
	}
	result, err := s.admin.ListUsers(c.Request.Context(), query)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleUpdateUserStatus 更新指定用户的启用状态。
func (s *Server) handleUpdateUserStatus(c *gin.Context) {
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	if err := s.admin.UpdateUserStatus(c.Request.Context(), c.Param("id"), req.Status); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleListRoles 分页返回角色及其用户数。
func (s *Server) handleListRoles(c *gin.Context) {
	result, err := s.admin.ListRoles(c.Request.Context(), parseListQuery(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleListPermissions 返回权限树。
func (s *Server) handleListPermissions(c *gin.Context) {
	result, err := s.admin.ListPermissions(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleListTenants 分页返回租户。
func (s *Server) handleListTenants(c *gin.Context) {
	result, err := s.admin.ListTenants(c.Request.Context(), parseListQuery(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// parseListQuery 解析与前端 ListQuery 一致的分页与搜索参数。
func parseListQuery(c *gin.Context) domain.ListQuery {
	return domain.ListQuery{
		Page:     positiveInt(c.Query("page"), 1),
		PageSize: positiveInt(c.Query("pageSize"), 20),
		Search:   c.Query("search"),
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
