package mongostore

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// RoleRepository 是角色的 MongoDB 实现。
type RoleRepository struct {
	coll *mongo.Collection
}

// List 通过 $lookup 聚合用户数，并支持关键词搜索。
func (r *RoleRepository) List(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Role], error) {
	query = query.Normalize()
	pipeline := mongo.Pipeline{
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "roleIds"},
			{Key: "as", Value: "matchedUsers"},
		}}},
		{{Key: "$addFields", Value: bson.D{{Key: "userCount", Value: bson.D{{Key: "$size", Value: "$matchedUsers"}}}}}},
		{{Key: "$project", Value: bson.D{{Key: "matchedUsers", Value: 0}}}},
	}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: pattern}},
			bson.D{{Key: "description", Value: pattern}},
			bson.D{{Key: "permissionCodes", Value: pattern}},
		}}}}})
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
	items := []domain.Role{}
	if err := cursor.All(ctx, &items); err != nil {
		return domain.PageResult[domain.Role]{}, err
	}

	total, err := r.count(ctx, totalPipeline)
	if err != nil {
		return domain.PageResult[domain.Role]{}, err
	}
	return domain.PageResult[domain.Role]{Items: items, Total: total}, nil
}

// GetByIDs 批量查询角色，供用户列表填充角色名。
func (r *RoleRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.Role, error) {
	if len(ids) == 0 {
		return []domain.Role{}, nil
	}
	cursor, err := r.coll.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []domain.Role{}
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
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
