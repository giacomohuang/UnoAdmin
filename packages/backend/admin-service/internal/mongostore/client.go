// Package mongostore 提供领域仓储的 MongoDB 实现。
package mongostore

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Store 持有 MongoDB 客户端与数据库句柄。
type Store struct {
	Client *mongo.Client
	DB     *mongo.Database
}

// Connect 建立连接并立即 Ping 确认数据库可用。
func Connect(ctx context.Context, uri string, database string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}
	return &Store{Client: client, DB: client.Database(database)}, nil
}

// Close 释放连接。
func (s *Store) Close(ctx context.Context) error {
	return s.Client.Disconnect(ctx)
}

// EnsureIndexes 创建唯一约束与令牌黑名单的 TTL 索引。
func (s *Store) EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{"users", []mongo.IndexModel{
			{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "roleIds", Value: 1}}},
			{Keys: bson.D{{Key: "tenantIds", Value: 1}}},
		}},
		{"roles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
		}},
		{"permissions", []mongo.IndexModel{
			{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "parentId", Value: 1}}},
		}},
		{"tenants", []mongo.IndexModel{
			{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		}},
		// expiresAt 到期后由 MongoDB 自动清理，避免黑名单无限增长。
		{"token_blacklist", []mongo.IndexModel{
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		}},
	}

	for _, item := range indexes {
		if _, err := s.DB.Collection(item.collection).Indexes().CreateMany(ctx, item.models); err != nil {
			return fmt.Errorf("create indexes for %s: %w", item.collection, err)
		}
	}
	return nil
}

// NewUserRepository 构造用户仓储。
func (s *Store) NewUserRepository() *UserRepository {
	return &UserRepository{coll: s.DB.Collection("users")}
}

// NewRoleRepository 构造角色仓储。
func (s *Store) NewRoleRepository() *RoleRepository {
	return &RoleRepository{coll: s.DB.Collection("roles")}
}

// NewPermissionRepository 构造权限仓储。
func (s *Store) NewPermissionRepository() *PermissionRepository {
	return &PermissionRepository{coll: s.DB.Collection("permissions")}
}

// NewTenantRepository 构造租户仓储。
func (s *Store) NewTenantRepository() *TenantRepository {
	return &TenantRepository{coll: s.DB.Collection("tenants")}
}

// NewTokenBlacklist 构造令牌黑名单。
func (s *Store) NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{coll: s.DB.Collection("token_blacklist")}
}
