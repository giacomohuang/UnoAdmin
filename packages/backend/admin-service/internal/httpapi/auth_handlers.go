package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/unoadmin/admin-service/internal/domain"
)

// authServiceAPI 定义认证处理器依赖的用例。
type authServiceAPI interface {
	SignIn(ctx context.Context, username string, password string) (domain.TokenPair, domain.AuthSession, error)
	Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, domain.AuthSession, error)
	Restore(ctx context.Context, raw string) (domain.AuthSession, error)
	SignOut(ctx context.Context, raw string, refreshToken string) error
	SwitchTenant(ctx context.Context, raw string, refreshToken string, tenantID string) (domain.TokenPair, error)
	SwitchLegalEntity(ctx context.Context, raw string, refreshToken string, entityID string) (domain.TokenPair, error)
	Verify(ctx context.Context, raw string) (*domain.Principal, error)
}

// loginRequest 是登录请求体。
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse 把令牌对与会话平铺为前端可直接消费的结构。
type loginResponse struct {
	domain.TokenPair
	domain.AuthSession
}

// switchTenantRequest 是切换租户请求体，携带刷新令牌以便同步轮换。
type switchTenantRequest struct {
	TenantID     string `json:"tenantId"`
	RefreshToken string `json:"refreshToken"`
}

// switchLegalEntityRequest 是切换经营主体请求体。
type switchLegalEntityRequest struct {
	LegalEntityID string `json:"legalEntityId"`
	RefreshToken  string `json:"refreshToken"`
}

// refreshRequest 是刷新访问令牌的请求体。
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// handleLogin 处理账号密码登录。
func (s *Server) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(c, domain.ErrInvalid)
		return
	}

	pair, session, err := s.auth.SignIn(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, loginResponse{TokenPair: pair, AuthSession: session})
}

// handleRefresh 用 Redis 中的刷新令牌换取新令牌对。
func (s *Server) handleRefresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	pair, session, err := s.auth.Refresh(c.Request.Context(), strings.TrimSpace(req.RefreshToken))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, loginResponse{TokenPair: pair, AuthSession: session})
}

// handleRestore 根据 Bearer 令牌重建会话。
func (s *Server) handleRestore(c *gin.Context) {
	session, err := s.auth.Restore(c.Request.Context(), bearerToken(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// handleLogout 吊销当前访问令牌并删除关联的刷新令牌。
func (s *Server) handleLogout(c *gin.Context) {
	var req refreshRequest
	// 请求体缺失不阻断注销，仍然吊销访问令牌。
	_ = c.ShouldBindJSON(&req)
	if err := s.auth.SignOut(c.Request.Context(), bearerToken(c), strings.TrimSpace(req.RefreshToken)); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleSwitchTenant 校验租户归属并返回携带新租户的新令牌。
func (s *Server) handleSwitchTenant(c *gin.Context) {
	var req switchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	if strings.TrimSpace(req.TenantID) == "" {
		writeError(c, domain.ErrInvalid)
		return
	}
	pair, err := s.auth.SwitchTenant(c.Request.Context(), bearerToken(c), strings.TrimSpace(req.RefreshToken), req.TenantID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, pair)
}

// handleSwitchLegalEntity 在租户内切换经营主体，并返回携带新主体的令牌。
func (s *Server) handleSwitchLegalEntity(c *gin.Context) {
	var req switchLegalEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, domain.ErrInvalid)
		return
	}
	if strings.TrimSpace(req.LegalEntityID) == "" {
		writeError(c, domain.ErrInvalid)
		return
	}
	pair, err := s.auth.SwitchLegalEntity(c.Request.Context(), bearerToken(c), strings.TrimSpace(req.RefreshToken), req.LegalEntityID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, pair)
}

// bearerToken 从请求头提取令牌，缺失时返回空串由服务层判定为未授权。
func bearerToken(c *gin.Context) string {
	return strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
}
