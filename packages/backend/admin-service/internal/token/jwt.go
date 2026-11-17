// Package token 实现基于 JWT 的访问令牌签发、校验与吊销。
package token

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/unoadmin/admin-service/internal/domain"
)

// jwtClaims 是令牌中携带的自定义声明与标准声明。
// 权限不写入令牌，每次校验时从数据库重新加载，避免已撤销权限继续生效。
type jwtClaims struct {
	AccountID     string `json:"uid"`
	TenantID      string `json:"tid"`
	LegalEntityID string `json:"eid"`
	jwt.RegisteredClaims
}

// JWT 是基于 HMAC-SHA256 的 TokenProvider 实现。
type JWT struct {
	secret    []byte
	ttl       time.Duration
	blacklist domain.TokenBlacklist
}

// NewJWT 构造令牌服务；ttl 为访问令牌有效期。
func NewJWT(secret string, ttl time.Duration, blacklist domain.TokenBlacklist) *JWT {
	return &JWT{
		secret:    []byte(secret),
		ttl:       ttl,
		blacklist: blacklist,
	}
}

// Issue 为给定主体签发访问令牌，并生成唯一的 jti 供吊销使用。
func (p *JWT) Issue(_ context.Context, claims domain.Claims) (string, error) {
	tokenID, err := newTokenID()
	if err != nil {
		return "", err
	}
	now := time.Now()
	expiresAt := now.Add(p.ttl)
	data := jwtClaims{
		AccountID:     claims.AccountID,
		TenantID:      claims.TenantID,
		LegalEntityID: claims.LegalEntityID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Issuer:    "unoadmin-admin-service",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, data).SignedString(p.secret)
}

// Verify 校验签名与有效期，并检查令牌是否已被吊销。
func (p *JWT) Verify(ctx context.Context, raw string) (*domain.Claims, error) {
	data := &jwtClaims{}
	parsed, err := jwt.ParseWithClaims(raw, data, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return p.secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, domain.ErrUnauthorized
	}

	revoked, err := p.blacklist.IsRevoked(ctx, data.ID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, domain.ErrUnauthorized
	}

	return &domain.Claims{
		AccountID:     data.AccountID,
		TenantID:      data.TenantID,
		LegalEntityID: data.LegalEntityID,
		TokenID:       data.ID,
		IssuedAt:      data.IssuedAt.Time,
		ExpiresAt:     data.ExpiresAt.Time,
	}, nil
}

// Revoke 将令牌 ID 加入黑名单，过期时间用于 TTL 自动清理。
func (p *JWT) Revoke(ctx context.Context, tokenID string, expiresAt time.Time) error {
	return p.blacklist.Add(ctx, tokenID, expiresAt)
}

// newTokenID 生成 128 位随机令牌 ID。
func newTokenID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
