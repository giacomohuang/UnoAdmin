package mongostore

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// tenantDoc 是租户的持久化结构，_id 使用 MongoDB 原生 ObjectId。
type tenantDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	Code      string        `bson:"code"`
	Mode      string        `bson:"mode"`
	Plan      string        `bson:"plan"`
	Contact   string        `bson:"contact"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// toDomain 把持久化结构转换为领域实体；列表类字段由服务层另行填充。
func (d tenantDoc) toDomain() domain.Tenant {
	return domain.Tenant{
		ID: objectIDHex(d.ID), Name: d.Name, Code: d.Code, Mode: d.Mode,
		Plan: d.Plan, Contact: d.Contact, Status: d.Status,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// tenantDocFrom 把领域实体转换为持久化结构，ID 为空时生成新 ObjectId。
func tenantDocFrom(tenant domain.Tenant) (tenantDoc, error) {
	id, err := parseObjectID(tenant.ID)
	if err != nil {
		return tenantDoc{}, err
	}
	if id.IsZero() {
		id = bson.NewObjectID()
	}
	return tenantDoc{
		ID: id, Name: tenant.Name, Code: tenant.Code, Mode: tenant.Mode,
		Plan: tenant.Plan, Contact: tenant.Contact, Status: tenant.Status,
		CreatedAt: tenant.CreatedAt, UpdatedAt: tenant.UpdatedAt,
	}, nil
}

// TenantRepository 是租户的 MongoDB 实现。
type TenantRepository struct {
	coll *mongo.Collection
}

// List 支持按租户名或编码搜索并分页。
func (r *TenantRepository) List(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Tenant], error) {
	query = query.Normalize()
	filter := tenantFilter(query)
	opts := options.Find().SetSkip(query.Offset()).SetLimit(query.Limit()).SetSort(bson.D{{Key: "_id", Value: 1}})
	items, err := r.find(ctx, filter, opts)
	if err != nil {
		return domain.PageResult[domain.Tenant]{}, err
	}
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return domain.PageResult[domain.Tenant]{}, err
	}
	return domain.PageResult[domain.Tenant]{Items: items, Total: total}, nil
}

// ListAll 返回全部租户，供平台管理员组装可切换租户列表。
func (r *TenantRepository) ListAll(ctx context.Context) ([]domain.Tenant, error) {
	return r.find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
}

// Get 按 ID 查询租户。
func (r *TenantRepository) Get(ctx context.Context, id string) (*domain.Tenant, error) {
	oid, err := parseObjectID(id)
	if err != nil || oid.IsZero() {
		return nil, domain.ErrNotFound
	}
	var doc tenantDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	tenant := doc.toDomain()
	return &tenant, nil
}

// GetByIDs 批量查询租户，供会话组装使用。
func (r *TenantRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.Tenant, error) {
	oids, err := parseObjectIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(oids) == 0 {
		return []domain.Tenant{}, nil
	}
	return r.find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}})
}

// Create 写入新租户，编码唯一索引冲突时返回 ErrConflict。
func (r *TenantRepository) Create(ctx context.Context, tenant domain.Tenant) (*domain.Tenant, error) {
	doc, err := tenantDocFrom(tenant)
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

// Update 覆盖租户的可编辑字段。
func (r *TenantRepository) Update(ctx context.Context, tenant domain.Tenant) (*domain.Tenant, error) {
	doc, err := tenantDocFrom(tenant)
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

// Count 返回租户总数，供平台工作台展示。
func (r *TenantRepository) Count(ctx context.Context) (int64, error) {
	return r.coll.CountDocuments(ctx, bson.D{})
}

// find 执行查询并转换为领域实体。
func (r *TenantRepository) find(ctx context.Context, filter bson.D, opts ...options.Lister[options.FindOptions]) ([]domain.Tenant, error) {
	cursor, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []tenantDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	items := make([]domain.Tenant, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.toDomain())
	}
	return items, nil
}

// tenantFilter 组装租户列表的过滤条件。
func tenantFilter(query domain.ListQuery) bson.D {
	filter := bson.D{}
	if query.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: query.Status})
	}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: pattern}},
			bson.D{{Key: "code", Value: pattern}},
			bson.D{{Key: "contact", Value: pattern}},
		}})
	}
	return filter
}
