// Package auth 实现登录、会话恢复、退出与租户切换等业务用例。
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/token"
)

// Service 编排认证相关的领域逻辑，不感知 HTTP 或 gRPC 协议。
type Service struct {
	users      domain.UserRepository
	roles      domain.RoleRepository
	tenants    domain.TenantRepository
	tokens     domain.TokenProvider
	hasher     domain.PasswordHasher
	refresh    domain.RefreshTokenStore
	refreshTTL time.Duration
}

// NewService 构造认证服务，refreshTTL 决定刷新令牌在 Redis 中的有效期。
func NewService(
	users domain.UserRepository,
	roles domain.RoleRepository,
	tenants domain.TenantRepository,
	tokens domain.TokenProvider,
	hasher domain.PasswordHasher,
	refresh domain.RefreshTokenStore,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		users: users, roles: roles, tenants: tenants,
		tokens: tokens, hasher: hasher, refresh: refresh, refreshTTL: refreshTTL,
	}
}

// SignIn 校验账号密码，签发访问令牌并把刷新令牌存入 Redis。
func (s *Service) SignIn(ctx context.Context, username string, password string) (domain.TokenPair, domain.AuthSession, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
	}
	if user.Status != domain.UserActive {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrForbidden
	}

	session, err := s.sessionFor(ctx, user)
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	return s.issuePair(ctx, user.ID, session.CurrentTenantID, session)
}

// issuePair 签发令牌对并把刷新令牌写入 Redis。
func (s *Service) issuePair(ctx context.Context, userID string, tenantID string, session domain.AuthSession) (domain.TokenPair, domain.AuthSession, error) {
	access, err := s.tokens.Issue(ctx, domain.Claims{UserID: userID, TenantID: tenantID})
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	refresh, err := token.GenerateOpaqueToken()
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	if err := s.refresh.Save(ctx, refresh, domain.RefreshSession{UserID: userID, TenantID: tenantID}, s.refreshTTL); err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	return domain.TokenPair{AccessToken: access, RefreshToken: refresh}, session, nil
}

// Restore 根据令牌重建会话；权限与租户每次都从仓储重新读取，避免令牌中的旧权限长期生效。
func (s *Service) Restore(ctx context.Context, raw string) (domain.AuthSession, error) {
	claims, err := s.tokens.Verify(ctx, raw)
	if err != nil {
		return domain.AuthSession{}, err
	}
	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return domain.AuthSession{}, err
	}
	if user.Status != domain.UserActive {
		return domain.AuthSession{}, domain.ErrUnauthorized
	}
	return s.sessionFor(ctx, user)
}

// Refresh 用 Redis 中的刷新令牌换取新令牌对，并轮换刷新令牌以防重放。
func (s *Service) Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, domain.AuthSession, error) {
	if refreshToken == "" {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
	}
	stored, err := s.refresh.Get(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
		}
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	if user.Status != domain.UserActive {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
	}
	session, err := s.sessionFor(ctx, user)
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}

	// 刷新时沿用 Redis 中记录的租户；若该租户已不可用则回退到用户默认租户。
	tenantID := stored.TenantID
	validTenant := false
	for _, candidate := range session.Tenants {
		if candidate.ID == tenantID {
			validTenant = true
			break
		}
	}
	if !validTenant {
		tenantID = session.CurrentTenantID
	}

	access, err := s.tokens.Issue(ctx, domain.Claims{UserID: user.ID, TenantID: tenantID})
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	nextRefresh, err := token.GenerateOpaqueToken()
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	if err := s.refresh.Delete(ctx, refreshToken); err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	if err := s.refresh.Save(ctx, nextRefresh, domain.RefreshSession{UserID: user.ID, TenantID: tenantID}, s.refreshTTL); err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	return domain.TokenPair{AccessToken: access, RefreshToken: nextRefresh}, session, nil
}

// SignOut 吊销当前访问令牌并删除其刷新令牌。
func (s *Service) SignOut(ctx context.Context, raw string, refreshToken string) error {
	claims, err := s.tokens.Verify(ctx, raw)
	if err != nil {
		return err
	}
	if err := s.tokens.Revoke(ctx, claims.TokenID, claims.ExpiresAt); err != nil {
		return err
	}
	if refreshToken == "" {
		return nil
	}
	return s.refresh.Delete(ctx, refreshToken)
}

// SwitchTenant 校验目标租户归属，签发新访问令牌并同步更新 Redis 中的刷新令牌。
func (s *Service) SwitchTenant(ctx context.Context, raw string, refreshToken string, tenantID string) (domain.TokenPair, error) {
	claims, err := s.tokens.Verify(ctx, raw)
	if err != nil {
		return domain.TokenPair{}, err
	}
	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return domain.TokenPair{}, err
	}
	available := false
	for _, id := range user.TenantIDs {
		if id == tenantID {
			available = true
			break
		}
	}
	if !available {
		return domain.TokenPair{}, domain.ErrForbidden
	}

	access, err := s.tokens.Issue(ctx, domain.Claims{UserID: user.ID, TenantID: tenantID})
	if err != nil {
		return domain.TokenPair{}, err
	}
	if refreshToken == "" {
		return domain.TokenPair{AccessToken: access}, nil
	}
	nextRefresh, err := token.GenerateOpaqueToken()
	if err != nil {
		return domain.TokenPair{}, err
	}
	if err := s.refresh.Delete(ctx, refreshToken); err != nil {
		return domain.TokenPair{}, err
	}
	if err := s.refresh.Save(ctx, nextRefresh, domain.RefreshSession{UserID: user.ID, TenantID: tenantID}, s.refreshTTL); err != nil {
		return domain.TokenPair{}, err
	}
	return domain.TokenPair{AccessToken: access, RefreshToken: nextRefresh}, nil
}

// Verify 校验令牌并从仓储实时加载调用主体的权限，供 HTTP 中间件与 gRPC 适配器使用。
func (s *Service) Verify(ctx context.Context, raw string) (*domain.Principal, error) {
	claims, err := s.tokens.Verify(ctx, raw)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user.Status != domain.UserActive {
		return nil, domain.ErrUnauthorized
	}
	roles, err := s.roles.GetByIDs(ctx, user.RoleIDs)
	if err != nil {
		return nil, err
	}

	roleNames := make([]string, 0, len(roles))
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}
	return &domain.Principal{
		UserID:      user.ID,
		TenantID:    claims.TenantID,
		Permissions: domain.EffectivePermissions(roles),
		RoleNames:   roleNames,
	}, nil
}

// sessionFor 组装会话响应，供登录与恢复共用。
func (s *Service) sessionFor(ctx context.Context, user *domain.User) (domain.AuthSession, error) {
	roles, err := s.roles.GetByIDs(ctx, user.RoleIDs)
	if err != nil {
		return domain.AuthSession{}, err
	}
	tenants, err := s.tenants.GetByIDs(ctx, user.TenantIDs)
	if err != nil {
		return domain.AuthSession{}, err
	}

	currentTenantID := ""
	if len(tenants) > 0 {
		currentTenantID = tenants[0].ID
	}

	publicUser := user.Public()
	publicUser.RoleNames = make([]string, 0, len(roles))
	for _, role := range roles {
		publicUser.RoleNames = append(publicUser.RoleNames, role.Name)
	}
	return domain.AuthSession{
		User:            publicUser,
		Permissions:     domain.EffectivePermissions(roles),
		Tenants:         tenants,
		CurrentTenantID: currentTenantID,
	}, nil
}
