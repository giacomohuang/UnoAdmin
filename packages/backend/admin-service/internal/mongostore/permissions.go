package mongostore

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// permissionDoc 是权限节点的持久化结构，parentId 以 ObjectId 引用父节点。
type permissionDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	Code      string        `bson:"code"`
	Type      string        `bson:"type"`
	Scope     string        `bson:"scope"`
	ParentID  bson.ObjectID `bson:"parentId"`
	SortOrder int           `bson:"sortOrder"`
}

// permissionDocFrom 把领域实体转换为持久化结构，ID 为空时生成新 ObjectId。
func permissionDocFrom(permission domain.Permission) (permissionDoc, error) {
	id, err := parseObjectID(permission.ID)
	if err != nil {
		return permissionDoc{}, err
	}
	if id.IsZero() {
		id = bson.NewObjectID()
	}
	parentID, err := parseObjectID(permission.ParentID)
	if err != nil {
		return permissionDoc{}, err
	}
	return permissionDoc{
		ID: id, Name: permission.Name, Code: permission.Code, Type: permission.Type,
		Scope: permission.Scope, ParentID: parentID, SortOrder: permission.SortOrder,
	}, nil
}

// toDomain 把持久化结构转换为领域实体。
func (d permissionDoc) toDomain() domain.Permission {
	return domain.Permission{
		ID: objectIDHex(d.ID), Name: d.Name, Code: d.Code, Type: d.Type,
		Scope: d.Scope, ParentID: objectIDHex(d.ParentID), SortOrder: d.SortOrder,
	}
}

// PermissionRepository 是权限节点的 MongoDB 实现。
type PermissionRepository struct {
	coll *mongo.Collection
}

// ListAll 返回所有权限节点，按 sortOrder 与 ID 排序后交由领域层组装成树。
func (r *PermissionRepository) ListAll(ctx context.Context) ([]domain.Permission, error) {
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "_id", Value: 1}})
	cursor, err := r.coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []permissionDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	nodes := make([]domain.Permission, 0, len(docs))
	for _, doc := range docs {
		nodes = append(nodes, doc.toDomain())
	}
	return nodes, nil
}

// Get 按 ID 查询权限节点。
func (r *PermissionRepository) Get(ctx context.Context, id string) (*domain.Permission, error) {
	oid, err := parseObjectID(id)
	if err != nil || oid.IsZero() {
		return nil, domain.ErrNotFound
	}
	var doc permissionDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	node := doc.toDomain()
	return &node, nil
}

// Create 写入权限节点，编码冲突时返回 ErrConflict。
func (r *PermissionRepository) Create(ctx context.Context, permission domain.Permission) (*domain.Permission, error) {
	doc, err := permissionDocFrom(permission)
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

// Update 覆盖权限节点。
func (r *PermissionRepository) Update(ctx context.Context, permission domain.Permission) (*domain.Permission, error) {
	doc, err := permissionDocFrom(permission)
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

// Delete 移除权限节点。
func (r *PermissionRepository) Delete(ctx context.Context, id string) error {
	oid, err := parseObjectID(id)
	if err != nil || oid.IsZero() {
		return domain.ErrNotFound
	}
	result, err := r.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: oid}})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Reorder 按给定顺序重写同级节点的 sortOrder。
//
// 拖拽一次会提交同一列的全部节点，sortOrder 直接使用数组下标，保证连续且稳定。
func (r *PermissionRepository) Reorder(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(ids))
	for index, id := range ids {
		oid, err := parseObjectID(id)
		if err != nil || oid.IsZero() {
			continue
		}
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.D{{Key: "_id", Value: oid}}).
			SetUpdate(bson.D{{Key: "$set", Value: bson.D{{Key: "sortOrder", Value: index}}}}))
	}
	if len(models) == 0 {
		return nil
	}
	_, err := r.coll.BulkWrite(ctx, models)
	return err
}
