package mongostore

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// roleDoc 是角色的持久化结构，租户与专属主体以 ObjectId 引用。
type roleDoc struct {
	ID              bson.ObjectID `bson:"_id"`
	TenantID        bson.ObjectID `bson:"tenantId"`
	LegalEntityID   bson.ObjectID `bson:"legalEntityId"`
	Code            string        `bson:"code"`
	Name            string        `bson:"name"`
	Description     string        `bson:"description"`
	Status          string        `bson:"status"`
	DataScope       string        `bson:"dataScope"`
	PermissionCodes []string      `bson:"permissionCodes"`
	BuiltIn         bool          `bson:"builtIn"`
	CreatedAt       time.Time     `bson:"createdAt"`
	UpdatedAt       time.Time     `bson:"updatedAt"`
}

// roleRow 是角色聚合查询的行结构，附带账户数量。
//
// 直接平铺字段而不用内嵌结构：内嵌字段名不可导出时 bson 解码会跳过，
// 导致聚合结果全部为零值。
type roleRow struct {
	ID              bson.ObjectID `bson:"_id"`
	TenantID        bson.ObjectID `bson:"tenantId"`
	LegalEntityID   bson.ObjectID `bson:"legalEntityId"`
	Code            string        `bson:"code"`
	Name            string        `bson:"name"`
	Description     string        `bson:"description"`
	Status          string        `bson:"status"`
	DataScope       string        `bson:"dataScope"`
	PermissionCodes []string      `bson:"permissionCodes"`
	BuiltIn         bool          `bson:"builtIn"`
	CreatedAt       time.Time     `bson:"createdAt"`
	UpdatedAt       time.Time     `bson:"updatedAt"`
	UserCount       int           `bson:"userCount"`
}

// toDomain 把聚合行转换为领域实体。
func (r roleRow) toDomain() domain.Role {
	return domain.Role{
		ID: objectIDHex(r.ID), TenantID: objectIDHex(r.TenantID), LegalEntityID: objectIDHex(r.LegalEntityID),
		Code: r.Code, Name: r.Name, Description: r.Description, Status: r.Status,
		DataScope: r.DataScope, PermissionCodes: r.PermissionCodes, BuiltIn: r.BuiltIn,
		UserCount: r.UserCount, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// toDomain 把持久化结构转换为领域实体。
func (d roleDoc) toDomain() domain.Role {
	return domain.Role{
		ID: objectIDHex(d.ID), TenantID: objectIDHex(d.TenantID), LegalEntityID: objectIDHex(d.LegalEntityID),
		Code: d.Code, Name: d.Name, Description: d.Description, Status: d.Status,
		DataScope: d.DataScope, PermissionCodes: d.PermissionCodes, BuiltIn: d.BuiltIn,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// roleDocFrom 把领域实体转换为持久化结构，ID 为空时生成新 ObjectId。
func roleDocFrom(role domain.Role) (roleDoc, error) {
	id, err := parseObjectID(role.ID)
	if err != nil {
		return roleDoc{}, err
	}
	if id.IsZero() {
		id = bson.NewObjectID()
	}
	tenantID, err := parseObjectID(role.TenantID)
	if err != nil {
		return roleDoc{}, err
	}
	entityID, err := parseObjectID(role.LegalEntityID)
	if err != nil {
		return roleDoc{}, err
	}
	return roleDoc{
		ID: id, TenantID: tenantID, LegalEntityID: entityID, Code: role.Code,
		Name: role.Name, Description: role.Description, Status: role.Status,
		DataScope: role.DataScope, PermissionCodes: role.PermissionCodes, BuiltIn: role.BuiltIn,
		CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt,
	}, nil
}

// RoleRepository 是角色的 MongoDB 实现。
type RoleRepository struct {
	coll *mongo.Collection
}

// List 通过 $lookup 聚合账户数，并支持租户、状态与关键词过滤。
func (r *RoleRepository) List(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Role], error) {
	query = query.Normalize()
	match := bson.D{}
	if query.TenantID != "" {
		tenantID, err := parseObjectID(query.TenantID)
		if err != nil {
			return domain.PageResult[domain.Role]{}, err
		}
		match = append(match, bson.E{Key: "tenantId", Value: tenantID})
	}
	if query.Status != "" {
		match = append(match, bson.E{Key: "status", Value: query.Status})
	}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		match = append(match, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: pattern}},
			bson.D{{Key: "code", Value: pattern}},
			bson.D{{Key: "description", Value: pattern}},
			bson.D{{Key: "permissionCodes", Value: pattern}},
		}})
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "accounts"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "roleIds"},
			{Key: "as", Value: "matchedAccounts"},
		}}},
		{{Key: "$addFields", Value: bson.D{{Key: "userCount", Value: bson.D{{Key: "$size", Value: "$matchedAccounts"}}}}}},
		{{Key: "$project", Value: bson.D{{Key: "matchedAccounts", Value: 0}}}},
	}

	totalPipeline := append(mongo.Pipeline{}, pipeline...)
	totalPipeline = append(totalPipeline, bson.D{{Key: "$count", Value: "n"}})
	pagePipeline := append(mongo.Pipeline{}, pipeline...)
	pagePipeline = append(pagePipeline,
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
		bson.D{{Key: "$skip", Value: query.Offset()}},
		bson.D{{Key: "$limit", Value: query.Limit()}},
	)

	cursor, err := r.coll.Aggregate(ctx, pagePipeline)
	if err != nil {
		return domain.PageResult[domain.Role]{}, err
	}
	defer cursor.Close(ctx)
	var rows []roleRow
	if err := cursor.All(ctx, &rows); err != nil {
		return domain.PageResult[domain.Role]{}, err
	}
	items := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}

	total, err := r.count(ctx, totalPipeline)
	if err != nil {
		return domain.PageResult[domain.Role]{}, err
	}
	return domain.PageResult[domain.Role]{Items: items, Total: total}, nil
}

// GetByIDs 批量查询角色，供账户列表填充角色名。
func (r *RoleRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.Role, error) {
	oids, err := parseObjectIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(oids) == 0 {
		return []domain.Role{}, nil
	}
	cursor, err := r.coll.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var docs []roleDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	items := make([]domain.Role, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.toDomain())
	}
	return items, nil
}

// Create 写入角色，租户内编码冲突时返回 ErrConflict。
func (r *RoleRepository) Create(ctx context.Context, role domain.Role) (*domain.Role, error) {
	doc, err := roleDocFrom(role)
	if err != nil {
		return nil, err
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	created := doc.toDomain()
	return &created, nil
}

// Update 覆盖角色的可编辑字段。
func (r *RoleRepository) Update(ctx context.Context, role domain.Role) (*domain.Role, error) {
	doc, err := roleDocFrom(role)
	if err != nil {
		return nil, err
	}
	result, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: doc.ID}}, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, domain.ErrNotFound
	}
	updated := doc.toDomain()
	return &updated, nil
}

// count 执行计数聚合，空结果表示 0 条。
func (r *RoleRepository) count(ctx context.Context, pipeline mongo.Pipeline) (int64, error) {
	cursor, err := r.coll.Aggregate(ctx, pipeline, options.Aggregate())
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)
	var rows []struct {
		N int64 `bson:"n"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].N, nil
}
