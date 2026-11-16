package token_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/testutil"
	"github.com/unoadmin/admin-service/internal/token"
)

func TestIssueThenVerifyRoundTrip(t *testing.T) {
	provider := token.NewJWT("test-secret", time.Hour, testutil.NewBlacklist())
	claims := domain.Claims{
		UserID:   "user-001",
		TenantID: "tenant-main",
	}

	raw, err := provider.Issue(context.Background(), claims)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if raw == "" {
		t.Fatal("Issue() returned an empty token")
	}

	got, err := provider.Verify(context.Background(), raw)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if got.UserID != claims.UserID || got.TenantID != claims.TenantID {
		t.Fatalf("Verify() = %+v, want user/tenant %q/%q", got, claims.UserID, claims.TenantID)
	}
	if got.TokenID == "" {
		t.Fatal("Verify() returned an empty token ID")
	}
}

func TestIssuedTokenDoesNotCarryPermissions(t *testing.T) {
	provider := token.NewJWT("test-secret", time.Hour, testutil.NewBlacklist())
	raw, err := provider.Issue(context.Background(), domain.Claims{UserID: "user-001"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload error = %v", err)
	}
	if strings.Contains(string(payload), "perms") || strings.Contains(string(payload), "permissions") {
		t.Fatalf("token payload must not contain permissions, got %s", payload)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	provider := token.NewJWT("test-secret", time.Hour, testutil.NewBlacklist())
	raw, err := provider.Issue(context.Background(), domain.Claims{UserID: "user-001"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	tampered := raw[:len(raw)-2] + "xx"
	if _, err := provider.Verify(context.Background(), tampered); err == nil {
		t.Fatal("Verify() accepted a tampered token")
	}
}

func TestVerifyRejectsRevokedToken(t *testing.T) {
	blacklist := testutil.NewBlacklist()
	provider := token.NewJWT("test-secret", time.Hour, blacklist)
	raw, err := provider.Issue(context.Background(), domain.Claims{UserID: "user-001"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	claims, err := provider.Verify(context.Background(), raw)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if err := provider.Revoke(context.Background(), claims.TokenID, claims.ExpiresAt); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, err := provider.Verify(context.Background(), raw); err == nil {
		t.Fatal("Verify() accepted a revoked token")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	provider := token.NewJWT("test-secret", -time.Hour, testutil.NewBlacklist())
	raw, err := provider.Issue(context.Background(), domain.Claims{UserID: "user-001"})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if _, err := provider.Verify(context.Background(), raw); err == nil {
		t.Fatal("Verify() accepted an expired token")
	}
}
