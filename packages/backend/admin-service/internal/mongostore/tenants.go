package mongostore

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// TenantRepository 是租户的 MongoDB 实现。
type TenantRepository struct {
	coll *mongo.Collection
}

// List 支持按租户名或编码搜索并分页。
func (r *TenantRepository) List(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Tenant], error) {
	query = query.Normalize()
	filter := bson.D{}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: pattern}},
			bson.D{{Key: "code", Value: pattern}},
		}})
	}
	opts := options.Find().SetSkip(query.Offset()).SetLimit(query.Limit()).SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return domain.PageResult[domain.Tenant]{}, err
	}
	defer cursor.Close(ctx)
	items := []domain.Tenant{}
	if err := cursor.All(ctx, &items); err != nil {
		return domain.PageResult[domain.Tenant]{}, err
	}
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return domain.PageResult[domain.Tenant]{}, err
	}
	return domain.PageResult[domain.Tenant]{Items: items, Total: total}, nil
}

// GetByIDs 批量查询租户，供会话组装使用。
func (r *TenantRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.Tenant, error) {
	if len(ids) == 0 {
		return []domain.Tenant{}, nil
	}
	cursor, err := r.coll.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []domain.Tenant{}
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}
