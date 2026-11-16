package mongostore

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// DemoRepository 是示例业务数据的 MongoDB 实现。
type DemoRepository struct {
	overview *mongo.Collection
	records  *mongo.Collection
}

// NewDemoRepository 构造示例业务仓储。
func (s *Store) NewDemoRepository() *DemoRepository {
	return &DemoRepository{
		overview: s.DB.Collection("demo_overview"),
		records:  s.DB.Collection("demo_records"),
	}
}

// GetOverview 读取唯一的概览文档。
func (r *DemoRepository) GetOverview(ctx context.Context) (*domain.DemoOverview, error) {
	var overview domain.DemoOverview
	if err := r.overview.FindOne(ctx, bson.D{}).Decode(&overview); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &overview, nil
}

// ListRecords 按关键词搜索项目记录并分页。
func (r *DemoRepository) ListRecords(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.DemoRecord], error) {
	query = query.Normalize()
	filter := bson.D{}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: pattern}},
			bson.D{{Key: "owner", Value: pattern}},
			bson.D{{Key: "status", Value: pattern}},
		}})
	}
	opts := options.Find().SetSkip(query.Offset()).SetLimit(query.Limit()).SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor, err := r.records.Find(ctx, filter, opts)
	if err != nil {
		return domain.PageResult[domain.DemoRecord]{}, err
	}
	defer cursor.Close(ctx)
	items := []domain.DemoRecord{}
	if err := cursor.All(ctx, &items); err != nil {
		return domain.PageResult[domain.DemoRecord]{}, err
	}
	total, err := r.records.CountDocuments(ctx, filter)
	if err != nil {
		return domain.PageResult[domain.DemoRecord]{}, err
	}
	return domain.PageResult[domain.DemoRecord]{Items: items, Total: total}, nil
}
