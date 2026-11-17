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

// legalEntityDoc 是经营主体的持久化结构，租户与上级主体均以 ObjectId 引用。
type legalEntityDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	TenantID  bson.ObjectID `bson:"tenantId"`
	ParentID  bson.ObjectID `bson:"parentId"`
	Type      string        `bson:"type"`
	Name      string        `bson:"name"`
	Code      string        `bson:"code"`
	Address   string        `bson:"address"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// toDomain 把持久化结构转换为领域实体。
func (d legalEntityDoc) toDomain() domain.LegalEntity {
	return domain.LegalEntity{
		ID: objectIDHex(d.ID), TenantID: objectIDHex(d.TenantID), ParentID: objectIDHex(d.ParentID),
		Type: d.Type, Name: d.Name, Code: d.Code, Address: d.Address, Status: d.Status,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// legalEntityDocFrom 把领域实体转换为持久化结构，ID 为空时生成新 ObjectId。
func legalEntityDocFrom(entity domain.LegalEntity) (legalEntityDoc, error) {
	id, err := parseObjectID(entity.ID)
	if err != nil {
		return legalEntityDoc{}, err
	}
	if id.IsZero() {
		id = bson.NewObjectID()
	}
	tenantID, err := parseObjectID(entity.TenantID)
	if err != nil {
		return legalEntityDoc{}, err
	}
	parentID, err := parseObjectID(entity.ParentID)
	if err != nil {
		return legalEntityDoc{}, err
	}
	return legalEntityDoc{
		ID: id, TenantID: tenantID, ParentID: parentID, Type: entity.Type,
		Name: entity.Name, Code: entity.Code, Address: entity.Address, Status: entity.Status,
		CreatedAt: entity.CreatedAt, UpdatedAt: entity.UpdatedAt,
	}, nil
}

// LegalEntityRepository 是经营主体的 MongoDB 实现。
type LegalEntityRepository struct {
	coll *mongo.Collection
}

// List 按租户、主体类型、状态与关键词过滤经营主体。
func (r *LegalEntityRepository) List(ctx context.Context, query domain.ListQuery) ([]domain.LegalEntity, error) {
	filter := bson.D{}
	if query.TenantID != "" {
		tenantID, err := parseObjectID(query.TenantID)
		if err != nil {
			return nil, err
		}
		filter = append(filter, bson.E{Key: "tenantId", Value: tenantID})
	}
	if query.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: query.Status})
	}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: pattern}},
			bson.D{{Key: "code", Value: pattern}},
			bson.D{{Key: "address", Value: pattern}},
		}})
	}
	opts := options.Find().SetSort(bson.D{{Key: "type", Value: 1}, {Key: "_id", Value: 1}})
	return r.find(ctx, filter, opts)
}

// Get 按 ID 查询经营主体。
func (r *LegalEntityRepository) Get(ctx context.Context, id string) (*domain.LegalEntity, error) {
	oid, err := parseObjectID(id)
	if err != nil || oid.IsZero() {
		return nil, domain.ErrNotFound
	}
	var doc legalEntityDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	entity := doc.toDomain()
	return &entity, nil
}

// GetByIDs 批量查询经营主体，供账户列表填充主体名称。
func (r *LegalEntityRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.LegalEntity, error) {
	oids, err := parseObjectIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(oids) == 0 {
		return []domain.LegalEntity{}, nil
	}
	return r.find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}})
}

// Create 写入经营主体，租户内编码唯一。
func (r *LegalEntityRepository) Create(ctx context.Context, entity domain.LegalEntity) (*domain.LegalEntity, error) {
	doc, err := legalEntityDocFrom(entity)
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

// Update 覆盖经营主体的可编辑字段。
func (r *LegalEntityRepository) Update(ctx context.Context, entity domain.LegalEntity) (*domain.LegalEntity, error) {
	doc, err := legalEntityDocFrom(entity)
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

// CountByTenant 聚合每个租户的经营主体数量，返回租户 ID（十六进制）到数量的映射。
func (r *LegalEntityRepository) CountByTenant(ctx context.Context, tenantID string) (map[string]int, error) {
	counts := map[string]int{}
	filter := bson.D{}
	if tenantID != "" {
		oid, err := parseObjectID(tenantID)
		if err != nil {
			return nil, err
		}
		filter = append(filter, bson.E{Key: "tenantId", Value: oid})
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$tenantId"}, {Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	}
	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		ID    bson.ObjectID `bson:"_id"`
		Count int           `bson:"n"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[objectIDHex(row.ID)] = row.Count
	}
	return counts, nil
}

// find 执行查询并转换为领域实体。
func (r *LegalEntityRepository) find(ctx context.Context, filter bson.D, opts ...options.Lister[options.FindOptions]) ([]domain.LegalEntity, error) {
	cursor, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []legalEntityDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	items := make([]domain.LegalEntity, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.toDomain())
	}
	return items, nil
}
