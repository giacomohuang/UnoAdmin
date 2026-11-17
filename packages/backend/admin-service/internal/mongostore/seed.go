package mongostore

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/unoadmin/admin-service/internal/domain"
)

// seedPassword 是种子账号的默认密码，可通过命令行参数覆盖。
const seedPassword = "admin123"

// seedWriter 按自然键（编码、用户名、权限码等）幂等写入种子数据。
//
// 主键统一由 MongoDB 的 ObjectId 生成器产生，不使用业务键派生，
// 因此首次插入即得到标准 ObjectId；重复执行时按自然键复用已有主键。
type seedWriter struct {
	ctx         context.Context
	db          *mongo.Database
	permissions map[string]bson.ObjectID
	roles       map[string]bson.ObjectID
}

// Seed 写入基础管理数据。
//
// 种子覆盖两种客户形态：集团统一采购（一个租户 + 集团总部 + 多个经营公司）
// 与单体公司采购（一个租户 + 唯一经营主体）。
func Seed(ctx context.Context, store *Store, hasher domain.PasswordHasher, password string) error {
	if password == "" {
		password = seedPassword
	}
	hash, err := hasher.Hash(password)
	if err != nil {
		return err
	}

	w := &seedWriter{
		ctx:         ctx,
		db:          store.DB,
		permissions: map[string]bson.ObjectID{},
		roles:       map[string]bson.ObjectID{},
	}
	return w.seed(hash)
}

// seed 依次写入租户、主体、组织、权限、角色与账户。
func (w *seedWriter) seed(hash string) error {
	now := time.Now().UTC()

	holdingID, err := w.ensureTenant("HANTANG", "汉唐控股集团", domain.TenantModeGroup, "enterprise", "集团信息中心", now)
	if err != nil {
		return err
	}
	mingyuanID, err := w.ensureTenant("MINGYUAN", "明远科技", domain.TenantModeStandalone, "standard", "综合办公室", now)
	if err != nil {
		return err
	}

	hqID, err := w.ensureLegalEntity(holdingID, "HT-HQ", bson.NilObjectID, domain.LegalEntityTypeGroup,
		"汉唐控股集团总部", "上海市浦东新区世纪大道 100 号", now)
	if err != nil {
		return err
	}
	eastID, err := w.ensureLegalEntity(holdingID, "HT-EAST", hqID, domain.LegalEntityTypeCompany,
		"汉唐华东实业有限公司", "杭州市滨江区江南大道 588 号", now)
	if err != nil {
		return err
	}
	southID, err := w.ensureLegalEntity(holdingID, "HT-SOUTH", hqID, domain.LegalEntityTypeCompany,
		"汉唐华南贸易有限公司", "广州市天河区珠江新城 8 号", now)
	if err != nil {
		return err
	}
	mingyuanEntityID, err := w.ensureLegalEntity(mingyuanID, "MY", bson.NilObjectID, domain.LegalEntityTypeCompany,
		"明远科技", "深圳市南山区科技园南区 12 栋", now)
	if err != nil {
		return err
	}

	// 集团总部组织树
	hqRoot, err := w.ensureOrg(holdingID, hqID, "集团总部", bson.NilObjectID, nil, "company", 0, now)
	if err != nil {
		return err
	}
	if _, err := w.ensureOrg(holdingID, hqID, "财务管理部", hqRoot.ID, hqRoot.Path, "department", 0, now); err != nil {
		return err
	}
	if _, err := w.ensureOrg(holdingID, hqID, "人力资源部", hqRoot.ID, hqRoot.Path, "department", 1, now); err != nil {
		return err
	}
	if _, err := w.ensureOrg(holdingID, hqID, "信息中心", hqRoot.ID, hqRoot.Path, "department", 2, now); err != nil {
		return err
	}

	// 华东公司组织树
	eastRoot, err := w.ensureOrg(holdingID, eastID, "华东公司", bson.NilObjectID, nil, "company", 0, now)
	if err != nil {
		return err
	}
	if _, err := w.ensureOrg(holdingID, eastID, "销售部", eastRoot.ID, eastRoot.Path, "department", 0, now); err != nil {
		return err
	}
	serviceNode, err := w.ensureOrg(holdingID, eastID, "售后服务部", eastRoot.ID, eastRoot.Path, "department", 1, now)
	if err != nil {
		return err
	}
	if _, err := w.ensureOrg(holdingID, eastID, "杭州服务站", serviceNode.ID, serviceNode.Path, "team", 0, now); err != nil {
		return err
	}

	// 华南公司与单体公司的根组织
	if _, err := w.ensureOrg(holdingID, southID, "华南公司", bson.NilObjectID, nil, "company", 0, now); err != nil {
		return err
	}
	mingyuanRoot, err := w.ensureOrg(mingyuanID, mingyuanEntityID, "明远科技", bson.NilObjectID, nil, "company", 0, now)
	if err != nil {
		return err
	}
	if _, err := w.ensureOrg(mingyuanID, mingyuanEntityID, "产品研发部", mingyuanRoot.ID, mingyuanRoot.Path, "department", 0, now); err != nil {
		return err
	}

	if err := w.syncPermissions(); err != nil {
		return err
	}
	if err := w.prunePermissions(); err != nil {
		return err
	}

	if err := w.syncRoles(holdingID, mingyuanID, eastID, now); err != nil {
		return err
	}


	return w.ensureAccounts(hash, holdingID, mingyuanID, hqID, eastID, mingyuanEntityID, eastRoot.ID, mingyuanRoot.ID, now)
}

// ensure 按自然键查询记录，不存在时用新生成的 ObjectId 插入。
func (w *seedWriter) ensure(collection string, filter bson.D, fields bson.D) (bson.ObjectID, error) {
	id := bson.NewObjectID()
	return w.ensureWithID(collection, filter, id, fields)
}

// ensureWithID 在调用方已知主键（例如组织需要把自身写入 path）时使用。
func (w *seedWriter) ensureWithID(collection string, filter bson.D, id bson.ObjectID, fields bson.D) (bson.ObjectID, error) {
	coll := w.db.Collection(collection)
	existing, found, err := w.findID(coll, filter)
	if err != nil {
		return bson.NilObjectID, err
	}
	if found {
		return existing, nil
	}

	doc := bson.D{{Key: "_id", Value: id}}
	doc = append(doc, fields...)
	if _, err := coll.InsertOne(w.ctx, doc); err != nil {
		// 并发写入或历史数据竞争时按自然键回查一次。
		if mongo.IsDuplicateKeyError(err) {
			if existing, found, findErr := w.findID(coll, filter); findErr == nil && found {
				return existing, nil
			}
		}
		return bson.NilObjectID, err
	}
	return id, nil
}

// upsert 覆盖代码托管的文档（权限目录、内置角色），不存在时插入。
func (w *seedWriter) upsert(collection string, filter bson.D, fields bson.D) (bson.ObjectID, error) {
	return w.upsertKeep(collection, filter, fields)
}

// upsertKeep 在更新时保留指定字段的现有值。
//
// 权限目录的 sortOrder 允许管理员拖拽调整，重新执行种子不能把顺序重置回代码声明顺序。
func (w *seedWriter) upsertKeep(collection string, filter bson.D, fields bson.D, keep ...string) (bson.ObjectID, error) {
	coll := w.db.Collection(collection)
	existing, found, err := w.findID(coll, filter)
	if err != nil {
		return bson.NilObjectID, err
	}
	if found {
		update := bson.D{}
		for _, element := range fields {
			if containsValue(keep, element.Key) {
				continue
			}
			update = append(update, element)
		}
		if _, err := coll.UpdateOne(w.ctx, bson.D{{Key: "_id", Value: existing}}, bson.D{{Key: "$set", Value: update}}); err != nil {
			return bson.NilObjectID, err
		}
		return existing, nil
	}
	return w.ensure(collection, filter, fields)
}

// findID 按过滤条件查询主键。
func (w *seedWriter) findID(coll *mongo.Collection, filter bson.D) (bson.ObjectID, bool, error) {
	var doc struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := coll.FindOne(w.ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return bson.NilObjectID, false, nil
		}
		return bson.NilObjectID, false, err
	}
	return doc.ID, true, nil
}

// ensureTenant 写入集团或单体租户。
func (w *seedWriter) ensureTenant(code string, name string, mode string, plan string, contact string, now time.Time) (bson.ObjectID, error) {
	return w.ensure("tenants", bson.D{{Key: "code", Value: code}}, bson.D{
		{Key: "name", Value: name}, {Key: "code", Value: code}, {Key: "mode", Value: mode},
		{Key: "plan", Value: plan}, {Key: "contact", Value: contact},
		{Key: "status", Value: domain.TenantActive}, {Key: "createdAt", Value: now}, {Key: "updatedAt", Value: now},
	})
}

// ensureLegalEntity 写入经营主体，编码在租户内唯一。
func (w *seedWriter) ensureLegalEntity(tenantID bson.ObjectID, code string, parentID bson.ObjectID, kind string, name string, address string, now time.Time) (bson.ObjectID, error) {
	return w.ensure("legal_entities", bson.D{{Key: "tenantId", Value: tenantID}, {Key: "code", Value: code}}, bson.D{
		{Key: "tenantId", Value: tenantID}, {Key: "parentId", Value: parentID}, {Key: "type", Value: kind},
		{Key: "name", Value: name}, {Key: "code", Value: code}, {Key: "address", Value: address},
		{Key: "status", Value: domain.TenantActive}, {Key: "createdAt", Value: now}, {Key: "updatedAt", Value: now},
	})
}

// orgNode 返回已写入的组织节点，供构建子节点路径。
type orgNode struct {
	ID   bson.ObjectID
	Path []bson.ObjectID
}

// ensureOrg 写入组织节点，path 保存从根到自身的 ObjectId 链。
func (w *seedWriter) ensureOrg(tenantID bson.ObjectID, entityID bson.ObjectID, name string, parentID bson.ObjectID, parentPath []bson.ObjectID, kind string, sort int, now time.Time) (orgNode, error) {
	id := bson.NewObjectID()
	path := append(append([]bson.ObjectID{}, parentPath...), id)
	existing, err := w.ensureWithID("orgs", bson.D{
		{Key: "legalEntityId", Value: entityID}, {Key: "name", Value: name}, {Key: "parentId", Value: parentID},
	}, id, bson.D{
		{Key: "tenantId", Value: tenantID}, {Key: "legalEntityId", Value: entityID}, {Key: "parentId", Value: parentID},
		{Key: "name", Value: name}, {Key: "type", Value: kind}, {Key: "path", Value: path}, {Key: "sort", Value: sort},
		{Key: "status", Value: domain.TenantActive}, {Key: "createdAt", Value: now}, {Key: "updatedAt", Value: now},
	})
	if err != nil {
		return orgNode{}, err
	}
	if existing == id {
		return orgNode{ID: id, Path: path}, nil
	}

	// 复用已有节点时，路径以库中记录为准。
	var doc struct {
		Path []bson.ObjectID `bson:"path"`
	}
	if err := w.db.Collection("orgs").FindOne(w.ctx, bson.D{{Key: "_id", Value: existing}}).Decode(&doc); err != nil {
		return orgNode{}, err
	}
	return orgNode{ID: existing, Path: doc.Path}, nil
}

// permissionNode 描述一条权限目录项，父节点通过权限码引用。
type permissionNode struct {
	code   string
	name   string
	kind   string
	scope  string
	parent string
	sort   int
}

// permissionCatalog 是平台内置的权限目录，按父节点在前排列。
var permissionCatalog = []permissionNode{
	{code: "dashboard:view", name: "工作台", kind: "menu", scope: domain.PermissionScopeTenant, parent: "", sort: 0},
	{code: "system", name: "系统", kind: "menu", scope: domain.PermissionScopeTenant, parent: "", sort: 1},

	{code: "tenant:view", name: "租户", kind: "page", scope: domain.PermissionScopePlatform, parent: "system", sort: 0},
	{code: "tenant:edit", name: "编辑租户", kind: "action", scope: domain.PermissionScopePlatform, parent: "tenant:view", sort: 0},

	{code: "entity:view", name: "经营主体", kind: "page", scope: domain.PermissionScopeTenant, parent: "system", sort: 1},
	{code: "entity:edit", name: "编辑主体", kind: "action", scope: domain.PermissionScopeTenant, parent: "entity:view", sort: 0},

	{code: "org:view", name: "组织架构", kind: "page", scope: domain.PermissionScopeEntity, parent: "system", sort: 2},
	{code: "org:edit", name: "编辑组织", kind: "action", scope: domain.PermissionScopeEntity, parent: "org:view", sort: 0},

	{code: "account:view", name: "账户", kind: "page", scope: domain.PermissionScopeTenant, parent: "system", sort: 3},
	{code: "account:edit", name: "编辑账户", kind: "action", scope: domain.PermissionScopeTenant, parent: "account:view", sort: 0},

	{code: "role:view", name: "角色", kind: "page", scope: domain.PermissionScopeTenant, parent: "system", sort: 4},
	{code: "role:edit", name: "编辑角色", kind: "action", scope: domain.PermissionScopeTenant, parent: "role:view", sort: 0},

	{code: "permission:view", name: "权限", kind: "page", scope: domain.PermissionScopeTenant, parent: "system", sort: 5},
	{code: "permission:edit", name: "编辑权限", kind: "action", scope: domain.PermissionScopeTenant, parent: "permission:view", sort: 0},
	{code: "permission:delete", name: "删除权限", kind: "action", scope: domain.PermissionScopeTenant, parent: "permission:view", sort: 1},
}

// syncPermissions 按权限码覆盖写入权限目录，并记录主键供角色授权引用。
func (w *seedWriter) syncPermissions() error {
	for _, node := range permissionCatalog {
		parentID := bson.NilObjectID
		if node.parent != "" {
			resolved, ok := w.permissions[node.parent]
			if !ok {
				return errors.New("permission parent must be seeded before child: " + node.parent)
			}
			parentID = resolved
		}
		id, err := w.upsertKeep("permissions", bson.D{{Key: "code", Value: node.code}}, bson.D{
			{Key: "name", Value: node.name}, {Key: "code", Value: node.code}, {Key: "type", Value: node.kind},
			{Key: "scope", Value: node.scope}, {Key: "parentId", Value: parentID}, {Key: "sortOrder", Value: node.sort},
		}, "sortOrder")
		if err != nil {
			return err
		}
		w.permissions[node.code] = id
	}
	return nil
}

// prunePermissions 删除已从代码目录中移除的权限节点。
//
// 权限定义由代码托管，旧版本遗留的菜单码不应继续出现在授权树中；
// 角色中残留的旧权限码在权限校验时会被忽略。
func (w *seedWriter) prunePermissions() error {
	ids := make([]bson.ObjectID, 0, len(w.permissions))
	for _, id := range w.permissions {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := w.db.Collection("permissions").DeleteMany(w.ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$nin", Value: ids}}}})
	return err
}


// syncRoles 覆盖写入内置角色，租户级角色用空 ObjectId 表示跨主体。
func (w *seedWriter) syncRoles(holdingID bson.ObjectID, mingyuanID bson.ObjectID, eastID bson.ObjectID, now time.Time) error {
	roles := []struct {
		code        string
		tenantID    bson.ObjectID
		entityID    bson.ObjectID
		name        string
		desc        string
		scope       string
		permissions []string
	}{
		{
			code: "PLATFORM_ADMIN", tenantID: bson.NilObjectID, entityID: bson.NilObjectID,
			name: "平台运营管理员", desc: "管理全部租户、主体与账户，可跨租户排障",
			scope: domain.DataScopeTenant, permissions: []string{"*"},
		},
		{
			code: "GROUP_ADMIN", tenantID: holdingID, entityID: bson.NilObjectID,
			name: "集团管理员", desc: "统一管理集团下全部经营主体的账户与角色",
			scope: domain.DataScopeTenant,
			permissions: []string{
				"dashboard:view", "tenant:view", "entity:view", "entity:edit",
				"org:view", "org:edit",
				"account:view", "account:edit",
				"role:view", "role:edit", "permission:view",
			},
		},
		{
			code: "ENTITY_ADMIN", tenantID: holdingID, entityID: eastID,
			name: "主体管理员", desc: "仅管理被授权经营主体内的组织与账户",
			scope: domain.DataScopeEntity,
			permissions: []string{
				"dashboard:view", "entity:view", "org:view", "org:edit",
				"account:view", "account:edit",
			},
		},
		{
			code: "AUDITOR", tenantID: holdingID, entityID: bson.NilObjectID,
			name: "只读审计员", desc: "只读查看主体、组织与账户",
			scope:       domain.DataScopeTenant,
			permissions: []string{"dashboard:view", "entity:view", "org:view", "account:view", "role:view", "permission:view"},
		},
		{
			code: "TENANT_ADMIN", tenantID: mingyuanID, entityID: bson.NilObjectID,
			name: "企业管理员", desc: "单体公司内拥有全部管理权限",
			scope: domain.DataScopeTenant,
			permissions: []string{
				"dashboard:view", "tenant:view", "entity:view", "org:view", "org:edit",
				"account:view", "account:edit",
				"role:view", "role:edit", "permission:view",
			},
		},
	}

	for _, role := range roles {
		id, err := w.upsert("roles", bson.D{{Key: "tenantId", Value: role.tenantID}, {Key: "code", Value: role.code}}, bson.D{
			{Key: "tenantId", Value: role.tenantID}, {Key: "legalEntityId", Value: role.entityID},
			{Key: "code", Value: role.code}, {Key: "name", Value: role.name}, {Key: "description", Value: role.desc},
			{Key: "status", Value: domain.TenantActive}, {Key: "dataScope", Value: role.scope},
			{Key: "permissionCodes", Value: role.permissions}, {Key: "builtIn", Value: true},
			{Key: "createdAt", Value: now}, {Key: "updatedAt", Value: now},
		})
		if err != nil {
			return err
		}
		w.roles[role.code] = id
	}
	return nil
}

// ensureAccounts 写入四个演示账户，密码统一使用种子密码的哈希。
func (w *seedWriter) ensureAccounts(hash string, holdingID bson.ObjectID, mingyuanID bson.ObjectID, hqID bson.ObjectID, eastID bson.ObjectID, mingyuanEntityID bson.ObjectID, eastOrgID bson.ObjectID, mingyuanOrgID bson.ObjectID, now time.Time) error {
	accounts := []struct {
		username      string
		displayName   string
		email         string
		phone         string
		tenantID      bson.ObjectID
		scope         string
		platformAdmin bool
		entityIDs     bson.A
		orgIDs        bson.A
		roleCode      string
	}{
		{
			username: "lin.wei", displayName: "林薇", email: "lin.wei@example.com", phone: "13800000001",
			tenantID: bson.NilObjectID, scope: domain.DataScopeTenant, platformAdmin: true,
			entityIDs: bson.A{}, orgIDs: bson.A{}, roleCode: "PLATFORM_ADMIN",
		},
		{
			username: "chen.yu", displayName: "陈宇", email: "chen.yu@example.com", phone: "13800000002",
			tenantID: holdingID, scope: domain.DataScopeTenant, platformAdmin: false,
			entityIDs: bson.A{hqID}, orgIDs: bson.A{}, roleCode: "GROUP_ADMIN",
		},
		{
			username: "zhao.ming", displayName: "赵明", email: "zhao.ming@example.com", phone: "13800000003",
			tenantID: holdingID, scope: domain.DataScopeEntity, platformAdmin: false,
			entityIDs: bson.A{eastID}, orgIDs: bson.A{eastOrgID}, roleCode: "ENTITY_ADMIN",
		},
		{
			username: "li.na", displayName: "李娜", email: "li.na@example.com", phone: "13800000004",
			tenantID: mingyuanID, scope: domain.DataScopeTenant, platformAdmin: false,
			entityIDs: bson.A{mingyuanEntityID}, orgIDs: bson.A{mingyuanOrgID}, roleCode: "TENANT_ADMIN",
		},
	}

	for _, account := range accounts {
		roleID, ok := w.roles[account.roleCode]
		if !ok {
			return errors.New("role must be seeded before account: " + account.roleCode)
		}
		if _, err := w.ensure("accounts", bson.D{{Key: "username", Value: account.username}}, bson.D{
			{Key: "username", Value: account.username}, {Key: "passwordHash", Value: hash},
			{Key: "displayName", Value: account.displayName}, {Key: "email", Value: account.email},
			{Key: "phone", Value: account.phone}, {Key: "status", Value: domain.UserActive},
			{Key: "tenantId", Value: account.tenantID}, {Key: "dataScope", Value: account.scope},
			{Key: "platformAdmin", Value: account.platformAdmin},
			{Key: "legalEntityIds", Value: account.entityIDs}, {Key: "orgIds", Value: account.orgIDs},
			{Key: "roleIds", Value: bson.A{roleID}},
			{Key: "createdAt", Value: now}, {Key: "updatedAt", Value: now},
		}); err != nil {
			return err
		}
	}
	return nil
}

// containsValue 判断字符串切片是否包含目标值。
func containsValue(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
