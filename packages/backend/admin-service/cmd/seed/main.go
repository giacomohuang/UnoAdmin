// Command seed 向 MongoDB 写入与前端 mock 数据一致的基础账号与权限。
package main

import (
	"context"
	"log"
	"os"

	"github.com/unoadmin/admin-service/internal/auth"
	"github.com/unoadmin/admin-service/internal/config"
	"github.com/unoadmin/admin-service/internal/mongostore"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	store, err := mongostore.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer store.Close(ctx)

	if err := store.EnsureIndexes(ctx); err != nil {
		log.Fatalf("ensure indexes: %v", err)
	}
	password := os.Getenv("SEED_PASSWORD")
	if err := mongostore.Seed(ctx, store, auth.NewBcryptHasher(cfg.BCryptCost), password); err != nil {
		log.Fatalf("seed database: %v", err)
	}
	log.Printf("seed complete: database=%s defaultPassword=%q", cfg.MongoDatabase, seedPasswordDisplay(password))
}

// seedPasswordDisplay 只在日志里显示实际使用的种子密码，便于本地登录。
func seedPasswordDisplay(password string) string {
	if password == "" {
		return "admin123"
	}
	return password
}
