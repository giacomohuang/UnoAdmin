package mongostore

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// seedPassword 是种子账号的默认密码，可通过命令行参数覆盖。
const seedPassword = "admin123"

// Seed 写入与前端 mock adapter 一致的基础数据；已存在的记录不会被覆盖。
func Seed(ctx context.Context, store *Store, hasher domain.PasswordHasher, password string) error {
	if password == "" {
		password = seedPassword
	}
	hash, err := hasher.Hash(password)
	if err != nil {
		return err
	}

	if err := seedMany(ctx, store.DB.Collection("tenants"), seedTenants()); err != nil {
		return err
	}
	if err := seedMany(ctx, store.DB.Collection("permissions"), seedPermissions()); err != nil {
		return err
	}
	if err := seedMany(ctx, store.DB.Collection("roles"), seedRoles()); err != nil {
		return err
	}
	if err := seedMany(ctx, store.DB.Collection("users"), seedUsers(hash)); err != nil {
		return err
	}
	if err := seedMany(ctx, store.DB.Collection("demo_overview"), seedDemoOverview()); err != nil {
		return err
	}
	return seedMany(ctx, store.DB.Collection("demo_records"), seedDemoRecords())
}

// seedMany 按 _id 幂等 upsert 一组文档。
func seedMany(ctx context.Context, coll *mongo.Collection, docs []bson.D) error {
	if len(docs) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(docs))
	for _, doc := range docs {
		var id any
		for _, element := range doc {
			if element.Key == "_id" {
				id = element.Value
				break
			}
		}
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.D{{Key: "_id", Value: id}}).
			SetUpdate(bson.D{{Key: "$setOnInsert", Value: doc}}).
			SetUpsert(true))
	}
	_, err := coll.BulkWrite(ctx, models, options.BulkWrite())
	return err
}

// seedTenants 复制前端 mock 数据中的租户。
func seedTenants() []bson.D {
	return []bson.D{
		{{Key: "_id", Value: "tenant-main"}, {Key: "name", Value: "总部空间"}, {Key: "code", Value: "HQ"}, {Key: "status", Value: domain.TenantActive}},
		{{Key: "_id", Value: "tenant-lab"}, {Key: "name", Value: "创新实验室"}, {Key: "code", Value: "LAB"}, {Key: "status", Value: domain.TenantActive}},
		{{Key: "_id", Value: "tenant-archive"}, {Key: "name", Value: "归档空间"}, {Key: "code", Value: "ARCHIVE"}, {Key: "status", Value: domain.TenantSuspended}},
	}
}

// seedRoles 复制前端 mock 数据中的角色及其权限码。
func seedRoles() []bson.D {
	return []bson.D{
		{{Key: "_id", Value: "role-admin"}, {Key: "name", Value: "平台管理员"}, {Key: "description", Value: "管理全部基础能力和业务模块"}, {Key: "permissionCodes", Value: []string{"*"}}},
		{{Key: "_id", Value: "role-operator"}, {Key: "name", Value: "运营经理"}, {Key: "description", Value: "查看业务数据并执行日常运营操作"}, {Key: "permissionCodes", Value: []string{"user:read", "role:read", "tenant:read", "demo:read"}}},
		{{Key: "_id", Value: "role-viewer"}, {Key: "name", Value: "只读访客"}, {Key: "description", Value: "仅查看被授权的页面"}, {Key: "permissionCodes", Value: []string{"demo:read"}}},
	}
}

// seedPermissions 复制前端 mock 数据中的权限树，以 parentId 与 sortOrder 表达层级。
func seedPermissions() []bson.D {
	return []bson.D{
		{{Key: "_id", Value: "permission-system"}, {Key: "name", Value: "系统管理"}, {Key: "code", Value: "system"}, {Key: "type", Value: "menu"}, {Key: "parentId", Value: ""}, {Key: "sortOrder", Value: 0}},
		{{Key: "_id", Value: "permission-user"}, {Key: "name", Value: "用户管理"}, {Key: "code", Value: "user:read"}, {Key: "type", Value: "page"}, {Key: "parentId", Value: "permission-system"}, {Key: "sortOrder", Value: 0}},
		{{Key: "_id", Value: "permission-user-write"}, {Key: "name", Value: "编辑用户"}, {Key: "code", Value: "user:write"}, {Key: "type", Value: "action"}, {Key: "parentId", Value: "permission-user"}, {Key: "sortOrder", Value: 0}},
		{{Key: "_id", Value: "permission-role"}, {Key: "name", Value: "角色管理"}, {Key: "code", Value: "role:read"}, {Key: "type", Value: "page"}, {Key: "parentId", Value: "permission-system"}, {Key: "sortOrder", Value: 1}},
		{{Key: "_id", Value: "permission-role-grant"}, {Key: "name", Value: "角色授权"}, {Key: "code", Value: "role:grant"}, {Key: "type", Value: "action"}, {Key: "parentId", Value: "permission-role"}, {Key: "sortOrder", Value: 0}},
		{{Key: "_id", Value: "permission-tenant"}, {Key: "name", Value: "租户管理"}, {Key: "code", Value: "tenant:read"}, {Key: "type", Value: "page"}, {Key: "parentId", Value: "permission-system"}, {Key: "sortOrder", Value: 2}},
		{{Key: "_id", Value: "permission-demo"}, {Key: "name", Value: "示例业务"}, {Key: "code", Value: "demo"}, {Key: "type", Value: "menu"}, {Key: "parentId", Value: ""}, {Key: "sortOrder", Value: 1}},
		{{Key: "_id", Value: "permission-demo-read"}, {Key: "name", Value: "示例看板"}, {Key: "code", Value: "demo:read"}, {Key: "type", Value: "page"}, {Key: "parentId", Value: "permission-demo"}, {Key: "sortOrder", Value: 0}},
	}
}

// seedUsers 复制前端 mock 数据中的用户，密码统一使用种子密码的哈希。
func seedUsers(hash string) []bson.D {
	return []bson.D{
		{{Key: "_id", Value: "user-001"}, {Key: "username", Value: "lin.wei"}, {Key: "passwordHash", Value: hash}, {Key: "displayName", Value: "林薇"}, {Key: "email", Value: "lin.wei@example.com"}, {Key: "status", Value: domain.UserActive}, {Key: "roleIds", Value: []string{"role-admin"}}, {Key: "tenantIds", Value: []string{"tenant-main"}}},
		{{Key: "_id", Value: "user-002"}, {Key: "username", Value: "chen.yu"}, {Key: "passwordHash", Value: hash}, {Key: "displayName", Value: "陈宇"}, {Key: "email", Value: "chen.yu@example.com"}, {Key: "status", Value: domain.UserActive}, {Key: "roleIds", Value: []string{"role-operator"}}, {Key: "tenantIds", Value: []string{"tenant-main"}}},
		{{Key: "_id", Value: "user-003"}, {Key: "username", Value: "zhao.ming"}, {Key: "passwordHash", Value: hash}, {Key: "displayName", Value: "赵明"}, {Key: "email", Value: "zhao.ming@example.com"}, {Key: "status", Value: domain.UserDisabled}, {Key: "roleIds", Value: []string{"role-viewer"}}, {Key: "tenantIds", Value: []string{"tenant-main"}}},
	}
}
