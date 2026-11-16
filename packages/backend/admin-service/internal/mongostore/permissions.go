package mongostore

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

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

	nodes := []domain.Permission{}
	if err := cursor.All(ctx, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}
