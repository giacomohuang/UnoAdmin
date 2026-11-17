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

// orgDoc 是组织节点的持久化结构，path 保存从根到自身的 ObjectId 链。
type orgDoc struct {
	ID            bson.ObjectID   `bson:"_id"`
	TenantID      bson.ObjectID   `bson:"tenantId"`
	LegalEntityID bson.ObjectID   `bson:"legalEntityId"`
	ParentID      bson.ObjectID   `bson:"parentId"`
	Name          string          `bson:"name"`
	Type          string          `bson:"type"`
	Path          []bson.ObjectID `bson:"path"`
	Sort          int             `bson:"sort"`
	Status        string          `bson:"status"`
	CreatedAt     time.Time       `bson:"createdAt"`
	UpdatedAt     time.Time       `bson:"updatedAt"`
}

// toDomain 把持久化结构转换为领域实体。
func (d orgDoc) toDomain() domain.OrgUnit {
	return domain.OrgUnit{
		ID: objectIDHex(d.ID), TenantID: objectIDHex(d.TenantID), LegalEntityID: objectIDHex(d.LegalEntityID),
		ParentID: objectIDHex(d.ParentID), Name: d.Name, Type: d.Type, Path: objectIDHexes(d.Path),
		Sort: d.Sort, Status: d.Status, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// orgDocFrom 把领域实体转换为持久化结构，ID 为空时生成新 ObjectId。
func orgDocFrom(unit domain.OrgUnit) (orgDoc, error) {
	id, err := parseObjectID(unit.ID)
	if err != nil {
		return orgDoc{}, err
	}
	if id.IsZero() {
		id = bson.NewObjectID()
	}
	tenantID, err := parseObjectID(unit.TenantID)
	if err != nil {
		return orgDoc{}, err
	}
	entityID, err := parseObjectID(unit.LegalEntityID)
	if err != nil {
		return orgDoc{}, err
	}
	parentID, err := parseObjectID(unit.ParentID)
	if err != nil {
		return orgDoc{}, err
	}
	path, err := parseObjectIDs(unit.Path)
	if err != nil {
		return orgDoc{}, err
	}
	return orgDoc{
		ID: id, TenantID: tenantID, LegalEntityID: entityID, ParentID: parentID,
		Name: unit.Name, Type: unit.Type, Path: path, Sort: unit.Sort, Status: unit.Status,
		CreatedAt: unit.CreatedAt, UpdatedAt: unit.UpdatedAt,
	}, nil
}

// OrgRepository 是组织节点的 MongoDB 实现。
type OrgRepository struct {
	coll *mongo.Collection
}

// ListAll 返回指定租户（可再限定经营主体）下的全部组织节点。
func (r *OrgRepository) ListAll(ctx context.Context, tenantID string, legalEntityID string) ([]domain.OrgUnit, error) {
	filter := bson.D{}
	if tenantID != "" {
		oid, err := parseObjectID(tenantID)
		if err != nil {
			return nil, err
		}
		filter = append(filter, bson.E{Key: "tenantId", Value: oid})
	}
	if legalEntityID != "" {
		oid, err := parseObjectID(legalEntityID)
		if err != nil {
			return nil, err
		}
		filter = append(filter, bson.E{Key: "legalEntityId", Value: oid})
	}
	opts := options.Find().SetSort(bson.D{{Key: "sort", Value: 1}, {Key: "_id", Value: 1}})
	return r.find(ctx, filter, opts)
}

// Get 按 ID 查询组织节点。
func (r *OrgRepository) Get(ctx context.Context, id string) (*domain.OrgUnit, error) {
	oid, err := parseObjectID(id)
	if err != nil || oid.IsZero() {
		return nil, domain.ErrNotFound
	}
	var doc orgDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	unit := doc.toDomain()
	return &unit, nil
}

// GetByIDs 批量查询组织节点，供账户授权展示。
func (r *OrgRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.OrgUnit, error) {
	oids, err := parseObjectIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(oids) == 0 {
		return []domain.OrgUnit{}, nil
	}
	return r.find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}})
}

// Create 写入组织节点。
func (r *OrgRepository) Create(ctx context.Context, unit domain.OrgUnit) (*domain.OrgUnit, error) {
	doc, err := orgDocFrom(unit)
	if err != nil {
		return nil, err
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return nil, err
	}
	created := doc.toDomain()
	return &created, nil
}

// Update 覆盖组织节点的可编辑字段。
func (r *OrgRepository) Update(ctx context.Context, unit domain.OrgUnit) (*domain.OrgUnit, error) {
	doc, err := orgDocFrom(unit)
	if err != nil {
		return nil, err
	}
	result, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: doc.ID}}, doc)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, domain.ErrNotFound
	}
	updated := doc.toDomain()
	return &updated, nil
}

// find 执行查询并转换为领域实体。
func (r *OrgRepository) find(ctx context.Context, filter bson.D, opts ...options.Lister[options.FindOptions]) ([]domain.OrgUnit, error) {
	cursor, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []orgDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	items := make([]domain.OrgUnit, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.toDomain())
	}
	return items, nil
}
