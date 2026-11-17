// Package auth 实现登录、会话恢复、退出以及租户/经营主体上下文切换等业务用例。
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
	accounts   domain.AccountRepository
	roles      domain.RoleRepository
	tenants    domain.TenantRepository
	entities   domain.LegalEntityRepository
	tokens     domain.TokenProvider
	hasher     domain.PasswordHasher
	refresh    domain.RefreshTokenStore
	refreshTTL time.Duration
}

// NewService 构造认证服务，refreshTTL 决定刷新令牌在 Redis 中的有效期。
func NewService(
	accounts domain.AccountRepository,
	roles domain.RoleRepository,
	tenants domain.TenantRepository,
	entities domain.LegalEntityRepository,
	tokens domain.TokenProvider,
	hasher domain.PasswordHasher,
	refresh domain.RefreshTokenStore,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		accounts: accounts, roles: roles, tenants: tenants, entities: entities,
		tokens: tokens, hasher: hasher, refresh: refresh, refreshTTL: refreshTTL,
	}
}

// SignIn 校验账号密码，签发访问令牌并把刷新令牌存入 Redis。
func (s *Service) SignIn(ctx context.Context, username string, password string) (domain.TokenPair, domain.AuthSession, error) {
	account, err := s.accounts.GetByUsername(ctx, username)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	if err := s.hasher.Compare(account.PasswordHash, password); err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
	}
	if account.Status != domain.UserActive {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrForbidden
	}

	session, err := s.sessionFor(ctx, account)
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	return s.issuePair(ctx, account.ID, session.CurrentTenantID, session.CurrentLegalEntityID, session)
}

// issuePair 签发令牌对并把刷新令牌写入 Redis。
func (s *Service) issuePair(ctx context.Context, accountID string, tenantID string, entityID string, session domain.AuthSession) (domain.TokenPair, domain.AuthSession, error) {
	access, err := s.tokens.Issue(ctx, domain.Claims{AccountID: accountID, TenantID: tenantID, LegalEntityID: entityID})
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	refresh, err := token.GenerateOpaqueToken()
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	refreshSession := domain.RefreshSession{AccountID: accountID, TenantID: tenantID, LegalEntityID: entityID}
	if err := s.refresh.Save(ctx, refresh, refreshSession, s.refreshTTL); err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	return domain.TokenPair{AccessToken: access, RefreshToken: refresh}, session, nil
}

// Restore 根据令牌重建会话；权限与主体范围每次都从仓储重新读取，避免旧权限长期生效。
func (s *Service) Restore(ctx context.Context, raw string) (domain.AuthSession, error) {
	claims, err := s.tokens.Verify(ctx, raw)
	if err != nil {
		return domain.AuthSession{}, err
	}
	account, err := s.accounts.GetByID(ctx, claims.AccountID)
	if err != nil {
		return domain.AuthSession{}, err
	}
	if account.Status != domain.UserActive {
		return domain.AuthSession{}, domain.ErrUnauthorized
	}
	if claims.TenantID == "" {
		return s.sessionFor(ctx, account)
	}
	return s.sessionForContext(ctx, account, claims.TenantID, claims.LegalEntityID)
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
	account, err := s.accounts.GetByID(ctx, stored.AccountID)
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	if account.Status != domain.UserActive {
		return domain.TokenPair{}, domain.AuthSession{}, domain.ErrUnauthorized
	}
	// 刷新时沿用 Redis 中记录的上下文；租户失效时回退到账户默认租户。
	session, err := s.sessionForContext(ctx, account, stored.TenantID, stored.LegalEntityID)
	if err != nil {
		return domain.TokenPair{}, domain.AuthSession{}, err
	}
	tenantID := session.CurrentTenantID
	entityID := session.CurrentLegalEntityID

	access, err := s.tokens.Issue(ctx, domain.Claims{AccountID: account.ID, TenantID: tenantID, LegalEntityID: entityID})
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
	refreshSession := domain.RefreshSession{AccountID: account.ID, TenantID: tenantID, LegalEntityID: entityID}
	if err := s.refresh.Save(ctx, nextRefresh, refreshSession, s.refreshTTL); err != nil {
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
//
// 集团统一采购的客户可在一个租户内切换经营主体；跨租户切换仅对平台管理员开放。
func (s *Service) SwitchTenant(ctx context.Context, raw string, refreshToken string, tenantID string) (domain.TokenPair, error) {
	claims, err := s.tokens.Verify(ctx, raw)
	if err != nil {
		return domain.TokenPair{}, err
	}
	account, err := s.accounts.GetByID(ctx, claims.AccountID)
	if err != nil {
		return domain.TokenPair{}, err
	}
	if !account.PlatformAdmin && account.TenantID != tenantID {
		return domain.TokenPair{}, domain.ErrForbidden
	}
	tenant, err := s.tenants.Get(ctx, tenantID)
	if err != nil {
		return domain.TokenPair{}, err
	}
	if tenant.Status != domain.TenantActive {
		return domain.TokenPair{}, domain.ErrForbidden
	}

	session, err := s.sessionForTenant(ctx, account, tenantID)
	if err != nil {
		return domain.TokenPair{}, err
	}
	return s.rotate(ctx, account.ID, tenantID, session.CurrentLegalEntityID, refreshToken)
}

// SwitchLegalEntity 在租户内切换经营主体，并签发携带新主体上下文的令牌。
func (s *Service) SwitchLegalEntity(ctx context.Context, raw string, refreshToken string, entityID string) (domain.TokenPair, error) {
	claims, err := s.tokens.Verify(ctx, raw)
	if err != nil {
		return domain.TokenPair{}, err
	}
	account, err := s.accounts.GetByID(ctx, claims.AccountID)
	if err != nil {
		return domain.TokenPair{}, err
	}
	entity, err := s.entities.Get(ctx, entityID)
	if err != nil {
		return domain.TokenPair{}, err
	}
	if entity.TenantID != claims.TenantID {
		return domain.TokenPair{}, domain.ErrForbidden
	}
	// 数据范围取账户配置与角色的更宽者，避免角色为租户级时被账户配置误拦。
	roles, err := s.roles.GetByIDs(ctx, account.RoleIDs)
	if err != nil {
		return domain.TokenPair{}, err
	}
	scope := account.DataScope
	for _, role := range roles {
		if role.DataScope != "" {
			scope = domain.WidenScope(scope, role.DataScope)
		}
	}
	principal := &domain.Principal{
		AccountID:      account.ID,
		TenantID:       claims.TenantID,
		DataScope:      scope,
		LegalEntityIDs: account.LegalEntityIDs,
		PlatformAdmin:  account.PlatformAdmin,
	}
	if !domain.CanAccessEntity(principal, entity.ID) {
		return domain.TokenPair{}, domain.ErrForbidden
	}
	return s.rotate(ctx, account.ID, claims.TenantID, entity.ID, refreshToken)
}

// rotate 签发新令牌对，并在提供刷新令牌时同步轮换。
func (s *Service) rotate(ctx context.Context, accountID string, tenantID string, entityID string, refreshToken string) (domain.TokenPair, error) {
	access, err := s.tokens.Issue(ctx, domain.Claims{AccountID: accountID, TenantID: tenantID, LegalEntityID: entityID})
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
	refreshSession := domain.RefreshSession{AccountID: accountID, TenantID: tenantID, LegalEntityID: entityID}
	if err := s.refresh.Save(ctx, nextRefresh, refreshSession, s.refreshTTL); err != nil {
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
	account, err := s.accounts.GetByID(ctx, claims.AccountID)
	if err != nil {
		return nil, err
	}
	if account.Status != domain.UserActive {
		return nil, domain.ErrUnauthorized
	}
	roles, err := s.roles.GetByIDs(ctx, account.RoleIDs)
	if err != nil {
		return nil, err
	}

	roleNames := make([]string, 0, len(roles))
	scope := account.DataScope
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
		if role.DataScope != "" {
			scope = domain.WidenScope(scope, role.DataScope)
		}
	}
	return &domain.Principal{
		AccountID:      account.ID,
		TenantID:       claims.TenantID,
		LegalEntityID:  claims.LegalEntityID,
		DataScope:      scope,
		LegalEntityIDs: account.LegalEntityIDs,
		Permissions:    domain.EffectivePermissions(roles),
		RoleNames:      roleNames,
		PlatformAdmin:  account.PlatformAdmin,
	}, nil
}

// sessionForContext 在指定租户下重建会话；租户不可用时回退到账户默认租户。
//
// 令牌与刷新令牌只携带上下文标识，主体列表必须按目标租户重新查询，
// 否则平台管理员切换租户后仍会拿到上一个租户的经营主体。
func (s *Service) sessionForContext(ctx context.Context, account *domain.Account, tenantID string, entityID string) (domain.AuthSession, error) {
	tenants, err := s.availableTenants(ctx, account)
	if err != nil {
		return domain.AuthSession{}, err
	}
	if tenantID == "" || !tenantExists(tenants, tenantID) {
		return s.sessionFor(ctx, account)
	}
	session, err := s.sessionForTenant(ctx, account, tenantID)
	if err != nil {
		return domain.AuthSession{}, err
	}
	if entityID != "" && entityExists(session.LegalEntities, entityID) {
		session.CurrentLegalEntityID = entityID
	}
	return session, nil
}

// sessionFor 组装会话响应，供登录与恢复共用。
func (s *Service) sessionFor(ctx context.Context, account *domain.Account) (domain.AuthSession, error) {
	tenantID := account.TenantID
	if tenantID == "" {
		// 平台管理员没有归属租户，优先进入已配置经营主体的可用租户，
		// 避免默认落在空租户上导致上下文切换器显示「未选择主体」。
		all, err := s.tenants.ListAll(ctx)
		if err != nil {
			return domain.AuthSession{}, err
		}
		for _, tenant := range all {
			if tenant.Status != domain.TenantActive {
				continue
			}
			if tenantID == "" {
				tenantID = tenant.ID
			}
			entities, err := s.entities.List(ctx, domain.ListQuery{TenantID: tenant.ID})
			if err != nil {
				return domain.AuthSession{}, err
			}
			if len(entities) > 0 {
				tenantID = tenant.ID
				break
			}
		}
	}
	return s.sessionForTenant(ctx, account, tenantID)
}

// sessionForTenant 组装指定租户下的会话，并挑选默认经营主体。
func (s *Service) sessionForTenant(ctx context.Context, account *domain.Account, tenantID string) (domain.AuthSession, error) {
	roles, err := s.roles.GetByIDs(ctx, account.RoleIDs)
	if err != nil {
		return domain.AuthSession{}, err
	}
	// 数据范围取账户配置与角色的更宽者，与 Verify/SwitchLegalEntity 保持一致。
	scope := account.DataScope
	for _, role := range roles {
		if role.DataScope != "" {
			scope = domain.WidenScope(scope, role.DataScope)
		}
	}
	tenants, err := s.availableTenants(ctx, account)
	if err != nil {
		return domain.AuthSession{}, err
	}
	entities, err := s.entities.List(ctx, domain.ListQuery{TenantID: tenantID})
	if err != nil {
		return domain.AuthSession{}, err
	}

	principal := &domain.Principal{
		AccountID:      account.ID,
		TenantID:       tenantID,
		DataScope:      scope,
		LegalEntityIDs: account.LegalEntityIDs,
		PlatformAdmin:  account.PlatformAdmin,
	}
	visible := domain.VisibleEntities(principal, entities)
	visible = domain.SortLegalEntities(visible)

	// 当前主体优先使用账户显式授权的第一个主体；
	// 集团级账户回退到集团总部主体。
	currentEntityID := ""
	for _, entity := range visible {
		if contains(account.LegalEntityIDs, entity.ID) {
			currentEntityID = entity.ID
			break
		}
	}
	if currentEntityID == "" && len(visible) > 0 {
		currentEntityID = visible[0].ID
	}

	publicAccount := account.Public()
	publicAccount.RoleNames = make([]string, 0, len(roles))
	for _, role := range roles {
		publicAccount.RoleNames = append(publicAccount.RoleNames, role.Name)
	}
	publicAccount.LegalEntityNames = make([]string, 0, len(visible))
	for _, entity := range visible {
		publicAccount.LegalEntityNames = append(publicAccount.LegalEntityNames, entity.Name)
	}

	return domain.AuthSession{
		Account:              publicAccount,
		Permissions:          domain.EffectivePermissions(roles),
		Tenants:              tenants,
		CurrentTenantID:      tenantID,
		LegalEntities:        visible,
		CurrentLegalEntityID: currentEntityID,
		DataScope:            scope,
	}, nil
}

// availableTenants 返回账户可切换的租户；平台管理员可见全部租户。
func (s *Service) availableTenants(ctx context.Context, account *domain.Account) ([]domain.Tenant, error) {
	if account.PlatformAdmin {
		return s.tenants.ListAll(ctx)
	}
	if account.TenantID == "" {
		return []domain.Tenant{}, nil
	}
	return s.tenants.GetByIDs(ctx, []string{account.TenantID})
}

// tenantExists 判断租户是否在会话可切换范围内。
func tenantExists(tenants []domain.Tenant, id string) bool {
	for _, tenant := range tenants {
		if tenant.ID == id {
			return true
		}
	}
	return false
}

// entityExists 判断经营主体是否在会话可切换范围内。
func entityExists(entities []domain.LegalEntity, id string) bool {
	for _, entity := range entities {
		if entity.ID == id {
			return true
		}
	}
	return false
}

// contains 判断字符串切片是否包含目标值。
func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
