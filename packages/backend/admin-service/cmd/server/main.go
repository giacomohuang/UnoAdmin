// Command server 是 admin-base 后端微服务的组装入口，职责类似前端的 main.ts。
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/unoadmin/admin-service/internal/admin"
	"github.com/unoadmin/admin-service/internal/auth"
	"github.com/unoadmin/admin-service/internal/config"
	"github.com/unoadmin/admin-service/internal/grpcapi"
	"github.com/unoadmin/admin-service/internal/httpapi"
	"github.com/unoadmin/admin-service/internal/mongostore"
	"github.com/unoadmin/admin-service/internal/redisstore"
	"github.com/unoadmin/admin-service/internal/token"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// 基础设施：MongoDB 存放业务数据，Redis 存放刷新令牌。
	store, err := mongostore.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer store.Close(ctx)
	if err := store.EnsureIndexes(ctx); err != nil {
		log.Fatalf("ensure indexes: %v", err)
	}

	redisClient, err := redisstore.Connect(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	// 领域服务：所有依赖在此处显式装配，业务层不感知具体基础设施。
	hasher := auth.NewBcryptHasher(cfg.BCryptCost)
	tokens := token.NewJWT(cfg.JWTSecret, cfg.TokenTTL, store.NewTokenBlacklist())
	refreshStore := redisstore.NewRefreshTokenStore(redisClient)
	authService := auth.NewService(
		store.NewAccountRepository(),
		store.NewRoleRepository(),
		store.NewTenantRepository(),
		store.NewLegalEntityRepository(),
		tokens,
		hasher,
		refreshStore,
		cfg.RefreshTTL,
	)
	adminService := admin.NewService(
		store.NewAccountRepository(),
		store.NewRoleRepository(),
		store.NewPermissionRepository(),
		store.NewTenantRepository(),
		store.NewLegalEntityRepository(),
		store.NewOrgRepository(),
		hasher,
	)
	// REST 服务：供前端 SPA 调用。
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewServer(authService, adminService, cfg.AllowedOrigins),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// gRPC 服务：供其它微服务调用，使用共享密钥做服务间鉴权。
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcapi.ServiceTokenInterceptor(cfg.ServiceToken)))
	grpcapi.NewServer(authService, adminService).Register(grpcServer)
	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen gRPC: %v", err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

	log.Printf("admin-service ready: http=%s grpc=%s", cfg.HTTPAddr, cfg.GRPCAddr)

	// 优雅停机：收到信号后停止接收请求并释放连接。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()
	log.Println("admin-service stopped")
}
