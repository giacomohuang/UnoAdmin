package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/unoadmin/admin-service/internal/admin"
	"github.com/unoadmin/admin-service/internal/domain"
)

// adminServiceAPI 定义基础管理处理器依赖的用例。
type adminServiceAPI interface {
	ListTenants(ctx context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[admin.TenantOverview], error)
	SaveTenant(ctx context.Context, principal *domain.Principal, input admin.TenantInput) (*admin.TenantOverview, error)
	ListLegalEntities(ctx context.Context, principal *domain.Principal, query domain.ListQuery) ([]domain.LegalEntity, error)
	SaveLegalEntity(ctx context.Context, principal *domain.Principal, input admin.LegalEntityInput) (*domain.LegalEntity, error)
	ListOrgs(ctx context.Context, principal *domain.Principal, entityID string) ([]domain.OrgUnit, error)
	SaveOrg(ctx context.Context, principal *domain.Principal, input admin.OrgInput) (*domain.OrgUnit, error)
	ListAccounts(ctx context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[domain.Account], error)
	SaveAccount(ctx context.Context, principal *domain.Principal, input admin.AccountInput) (*domain.Account, error)
	UpdateAccountStatus(ctx context.Context, principal *domain.Principal, id string, status string) error
	ListRoles(ctx context.Context, principal *domain.Principal, query domain.ListQuery) (domain.PageResult[domain.Role], error)
	SaveRole(ctx context.Context, principal *domain.Principal, input admin.RoleInput) (*domain.Role, error)
	ListPermissions(ctx context.Context) ([]domain.Permission, error)
	Stats(ctx context.Context, principal *domain.Principal) (map[string]int, error)
}

// statusRequest 是账户状态变更请求体。
type statusRequest struct {
	Status string `json:"status"`
}

// handleStats 返回当前租户的规模指标。
func (s *Server) handleStats(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	result, err := s.admin.Stats(c.Request.Context(), principal)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleListTenants 分页返回租户；非平台管理员只会得到自己的租户。
func (s *Server) handleListTenants(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	result, err := s.admin.ListTenants(c.Request.Context(), principal, parseListQuery(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleCreateTenant 创建租户，单体租户会同步生成唯一的经营主体。
func (s *Server) handleCreateTenant(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.TenantInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	result, err := s.admin.SaveTenant(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// handleUpdateTenant 更新租户名称、形态、套餐与状态。
func (s *Server) handleUpdateTenant(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.TenantInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	input.ID = c.Param("id")
	result, err := s.admin.SaveTenant(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleListLegalEntities 返回被授权范围内的经营主体。
func (s *Server) handleListLegalEntities(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	query := parseListQuery(c)
	query.TenantID = c.Query("tenantId")
	result, err := s.admin.ListLegalEntities(c.Request.Context(), principal, query)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleCreateLegalEntity 在租户下新增经营主体。
func (s *Server) handleCreateLegalEntity(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.LegalEntityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	result, err := s.admin.SaveLegalEntity(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// handleUpdateLegalEntity 更新经营主体信息。
func (s *Server) handleUpdateLegalEntity(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.LegalEntityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	input.ID = c.Param("id")
	result, err := s.admin.SaveLegalEntity(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleListOrgs 返回指定经营主体的组织树。
func (s *Server) handleListOrgs(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	result, err := s.admin.ListOrgs(c.Request.Context(), principal, c.Query("legalEntityId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleCreateOrg 新增组织节点。
func (s *Server) handleCreateOrg(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.OrgInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	result, err := s.admin.SaveOrg(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// handleUpdateOrg 更新组织节点。
func (s *Server) handleUpdateOrg(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.OrgInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	input.ID = c.Param("id")
	result, err := s.admin.SaveOrg(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleListAccounts 分页返回账户。
func (s *Server) handleListAccounts(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	result, err := s.admin.ListAccounts(c.Request.Context(), principal, parseListQuery(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleCreateAccount 新增账户。
func (s *Server) handleCreateAccount(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.AccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	result, err := s.admin.SaveAccount(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// handleUpdateAccount 更新账户信息与授权范围。
func (s *Server) handleUpdateAccount(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.AccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	input.ID = c.Param("id")
	result, err := s.admin.SaveAccount(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleUpdateAccountStatus 更新指定账户的启用状态。
func (s *Server) handleUpdateAccountStatus(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	if err := s.admin.UpdateAccountStatus(c.Request.Context(), principal, c.Param("id"), req.Status); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleListRoles 分页返回角色及其账户数。
func (s *Server) handleListRoles(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	result, err := s.admin.ListRoles(c.Request.Context(), principal, parseListQuery(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleCreateRole 新增角色。
func (s *Server) handleCreateRole(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.RoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	result, err := s.admin.SaveRole(c.Request.Context(), principal, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// handleUpdateRole 更新角色及权限授权。
func (s *Server) handleUpdateRole(c *gin.Context) {
	principal, ok := currentPrincipal(c)
	if !ok {
		writeError(c, domain.ErrUnauthorized)
		return
	}
	var input admin.RoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	input.ID = c.Param("id")
	result, err := s.admin.SaveRole(c.Request.Context(), principal, input)
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

// parseListQuery 解析与前端 ListQuery 一致的分页与搜索参数。
func parseListQuery(c *gin.Context) domain.ListQuery {
	return domain.ListQuery{
		Page:     positiveInt(c.Query("page"), 1),
		PageSize: positiveInt(c.Query("pageSize"), 20),
		Search:   c.Query("search"),
		Status:   c.Query("status"),
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
