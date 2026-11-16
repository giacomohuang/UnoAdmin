package redisstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/redisstore"
)

func newStore(t *testing.T) *redisstore.RefreshTokenStore {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return redisstore.NewRefreshTokenStore(client)
}

func TestRefreshTokenStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	session := domain.RefreshSession{UserID: "user-001", TenantID: "tenant-main"}

	if err := store.Save(ctx, "token-a", session, time.Hour); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Get(ctx, "token-a")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.UserID != session.UserID || got.TenantID != session.TenantID {
		t.Fatalf("Get() = %+v, want %+v", got, session)
	}
}

func TestRefreshTokenStoreMissingAndDelete(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	if _, err := store.Get(ctx, "missing"); err != domain.ErrNotFound {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}
	if err := store.Save(ctx, "token-b", domain.RefreshSession{UserID: "user-001"}, time.Hour); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := store.Delete(ctx, "token-b"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(ctx, "token-b"); err != domain.ErrNotFound {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}
}
