package mongostore

import (
	"context"
	"errors"
	"regexp"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/unoadmin/admin-service/internal/domain"
)

// UserRepository 是用户的 MongoDB 实现。
type UserRepository struct {
	coll *mongo.Collection
}

// List 按租户与关键词过滤后分页，关键词匹配用户名、昵称与邮箱。
func (r *UserRepository) List(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.User], error) {
	query = query.Normalize()
	filter := bson.D{}
	if query.TenantID != "" {
		filter = append(filter, bson.E{Key: "tenantIds", Value: query.TenantID})
	}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "username", Value: pattern}},
			bson.D{{Key: "displayName", Value: pattern}},
			bson.D{{Key: "email", Value: pattern}},
		}})
	}

	opts := options.Find().SetSkip(query.Offset()).SetLimit(query.Limit()).SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return domain.PageResult[domain.User]{}, err
	}
	defer cursor.Close(ctx)

	items := []domain.User{}
	if err := cursor.All(ctx, &items); err != nil {
		return domain.PageResult[domain.User]{}, err
	}
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return domain.PageResult[domain.User]{}, err
	}
	return domain.PageResult[domain.User]{Items: items, Total: total}, nil
}

// GetByID 按 ID 查询用户。
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.findOne(ctx, bson.D{{Key: "_id", Value: id}})
}

// GetByUsername 按用户名查询用户， username 上有唯一索引。
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.findOne(ctx, bson.D{{Key: "username", Value: username}})
}

// UpdateStatus 更新用户状态，用户不存在时返回 ErrNotFound。
func (r *UserRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	result, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}}}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountByRoles 聚合统计每个角色关联的用户数。
func (r *UserRepository) CountByRoles(ctx context.Context, roleIDs []string) (map[string]int, error) {
	counts := map[string]int{}
	if len(roleIDs) == 0 {
		return counts, nil
	}
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$roleIds"}},
		{{Key: "$match", Value: bson.D{{Key: "roleIds", Value: bson.D{{Key: "$in", Value: roleIDs}}}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$roleIds"}, {Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	}
	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"n"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.ID] = row.Count
	}
	return counts, nil
}

// findOne 将 mongo.ErrNoDocuments 统一翻译为领域错误。
func (r *UserRepository) findOne(ctx context.Context, filter bson.D) (*domain.User, error) {
	var user domain.User
	if err := r.coll.FindOne(ctx, filter).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// primitiveRegex 构造不区分大小写的字面量正则，避免搜索词中的正则元字符被解释。
func primitiveRegex(search string) bson.Regex {
	return bson.Regex{Pattern: regexp.QuoteMeta(search), Options: "i"}
}
