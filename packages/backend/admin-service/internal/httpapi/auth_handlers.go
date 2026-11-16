package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/unoadmin/admin-service/internal/domain"
)

// authServiceAPI 定义认证处理器依赖的用例。
type authServiceAPI interface {
	SignIn(ctx context.Context, username string, password string) (domain.TokenPair, domain.AuthSession, error)
	Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, domain.AuthSession, error)
	Restore(ctx context.Context, raw string) (domain.AuthSession, error)
	SignOut(ctx context.Context, raw string, refreshToken string) error
	SwitchTenant(ctx context.Context, raw string, refreshToken string, tenantID string) (domain.TokenPair, error)
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

// refreshRequest 是刷新访问令牌的请求体。
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// handleLogin 处理账号密码登录。
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, domain.ErrInvalid)
		return
	}

	pair, session, err := s.auth.SignIn(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{TokenPair: pair, AuthSession: session})
}

// handleRefresh 用 Redis 中的刷新令牌换取新令牌对。
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	pair, session, err := s.auth.Refresh(r.Context(), strings.TrimSpace(req.RefreshToken))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{TokenPair: pair, AuthSession: session})
}

// handleRestore 根据 Bearer 令牌重建会话。
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	session, err := s.auth.Restore(r.Context(), bearerToken(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// handleLogout 吊销当前访问令牌并删除关联的刷新令牌。
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := s.auth.SignOut(r.Context(), bearerToken(r), strings.TrimSpace(req.RefreshToken)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleSwitchTenant 校验租户归属并返回携带新租户的新令牌。
func (s *Server) handleSwitchTenant(w http.ResponseWriter, r *http.Request) {
	var req switchTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.ErrInvalid)
		return
	}
	if strings.TrimSpace(req.TenantID) == "" {
		writeError(w, domain.ErrInvalid)
		return
	}
	pair, err := s.auth.SwitchTenant(r.Context(), bearerToken(r), strings.TrimSpace(req.RefreshToken), req.TenantID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pair)
}

// bearerToken 从请求头提取令牌，缺失时返回空串由服务层判定为未授权。
func bearerToken(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}
