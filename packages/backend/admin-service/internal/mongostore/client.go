// Package mongostore 提供领域仓储的 MongoDB 实现。
package mongostore

import (
	"context"
	"fmt"
	"strings"
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

// EnsureIndexes 迁移历史主键、补齐旧字段，并创建唯一约束与 TTL 索引。
func (s *Store) EnsureIndexes(ctx context.Context) error {
	// 旧版本主键为字符串，先统一迁移为 ObjectId，再建立索引。
	if err := s.migrateLegacyStringIDs(ctx); err != nil {
		return err
	}
	// 旧版本没有租户内编码概念，先补齐字段再建唯一索引，避免历史数据阻塞迁移。
	if err := s.backfillLegacyDefaults(ctx); err != nil {
		return err
	}

	indexes := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{"tenants", []mongo.IndexModel{
			{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "mode", Value: 1}}},
		}},
		{"legal_entities", []mongo.IndexModel{
			// 主体编码只需在租户内唯一，不同租户可以复用同一编码。
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "parentId", Value: 1}}},
		}},
		{"orgs", []mongo.IndexModel{
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "legalEntityId", Value: 1}}},
			{Keys: bson.D{{Key: "legalEntityId", Value: 1}, {Key: "parentId", Value: 1}, {Key: "sort", Value: 1}}},
		}},
		{"accounts", []mongo.IndexModel{
			{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "tenantId", Value: 1}}},
			{Keys: bson.D{{Key: "legalEntityIds", Value: 1}}},
			{Keys: bson.D{{Key: "roleIds", Value: 1}}},
		}},
		{"roles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		}},
		{"permissions", []mongo.IndexModel{
			{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "parentId", Value: 1}}},
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

// NewAccountRepository 构造账户仓储。
func (s *Store) NewAccountRepository() *AccountRepository {
	return &AccountRepository{coll: s.DB.Collection("accounts")}
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

// NewLegalEntityRepository 构造经营主体仓储。
func (s *Store) NewLegalEntityRepository() *LegalEntityRepository {
	return &LegalEntityRepository{coll: s.DB.Collection("legal_entities")}
}

// NewOrgRepository 构造组织架构仓储。
func (s *Store) NewOrgRepository() *OrgRepository {
	return &OrgRepository{coll: s.DB.Collection("orgs")}
}

// NewTokenBlacklist 构造令牌黑名单。
func (s *Store) NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{coll: s.DB.Collection("token_blacklist")}
}

// backfillLegacyDefaults 为早期版本的集合补齐多租户字段。
//
// 迁移目标：
//   - tenants 缺少 mode 时按集团租户处理，避免列表出现空形态；
//   - roles 缺少 tenantId/code 时用名称生成编码，使租户内唯一索引可建立。
func (s *Store) backfillLegacyDefaults(ctx context.Context) error {
	if _, err := s.DB.Collection("tenants").UpdateMany(ctx,
		bson.D{{Key: "mode", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "mode", Value: "group"}}}},
	); err != nil {
		return fmt.Errorf("backfill tenant mode: %w", err)
	}

	// 旧版权限没有 scope 字段，补齐为租户级，避免前端按层级渲染时取到空值。
	if _, err := s.DB.Collection("permissions").UpdateMany(ctx,
		bson.D{{Key: "scope", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "scope", Value: "tenant"}}}},
	); err != nil {
		return fmt.Errorf("backfill permission scope: %w", err)
	}

	// 旧版角色没有租户/主体引用，统一补空 ObjectId，保证字段类型一致。
	if _, err := s.DB.Collection("roles").UpdateMany(ctx,
		bson.D{{Key: "tenantId", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "tenantId", Value: bson.NilObjectID}}}},
	); err != nil {
		return fmt.Errorf("backfill role tenant: %w", err)
	}
	if _, err := s.DB.Collection("roles").UpdateMany(ctx,
		bson.D{{Key: "legalEntityId", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "legalEntityId", Value: bson.NilObjectID}}}},
	); err != nil {
		return fmt.Errorf("backfill role legal entity: %w", err)
	}

	cursor, err := s.DB.Collection("roles").Find(ctx, bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "code", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "code", Value: ""}},
	}}})
	if err != nil {
		return fmt.Errorf("find legacy roles: %w", err)
	}
	defer cursor.Close(ctx)

	var legacy []struct {
		ID   bson.ObjectID `bson:"_id"`
		Name string        `bson:"name"`
	}
	if err := cursor.All(ctx, &legacy); err != nil {
		return fmt.Errorf("decode legacy roles: %w", err)
	}
	for _, role := range legacy {
		if _, err := s.DB.Collection("roles").UpdateOne(ctx,
			bson.D{{Key: "_id", Value: role.ID}},
			bson.D{{Key: "$set", Value: bson.D{
				{Key: "code", Value: legacyCode(role.Name, role.ID.Hex())},
			}}},
		); err != nil {
			return fmt.Errorf("backfill role %s: %w", role.ID.Hex(), err)
		}
	}
	return nil
}

// legacyCode 用名称生成可读编码，名称为空时回退到记录 ID。
func legacyCode(name string, id string) string {
	code := strings.ToUpper(strings.TrimSpace(name))
	if code == "" {
		return strings.ToUpper(id)
	}
	return code
}
