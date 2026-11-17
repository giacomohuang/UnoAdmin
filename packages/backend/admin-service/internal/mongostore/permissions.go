package mongostore

import (
	"context"

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
