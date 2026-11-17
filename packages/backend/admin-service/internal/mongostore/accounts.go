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

// accountDoc 是管理端账户的持久化结构，租户、主体、组织与角色均以 ObjectId 引用。
type accountDoc struct {
	ID             bson.ObjectID   `bson:"_id"`
	Username       string          `bson:"username"`
	PasswordHash   string          `bson:"passwordHash"`
	DisplayName    string          `bson:"displayName"`
	Email          string          `bson:"email"`
	Phone          string          `bson:"phone"`
	Status         string          `bson:"status"`
	TenantID       bson.ObjectID   `bson:"tenantId"`
	DataScope      string          `bson:"dataScope"`
	PlatformAdmin  bool            `bson:"platformAdmin"`
	LegalEntityIDs []bson.ObjectID `bson:"legalEntityIds"`
	OrgIDs         []bson.ObjectID `bson:"orgIds"`
	RoleIDs        []bson.ObjectID `bson:"roleIds"`
	CreatedAt      time.Time       `bson:"createdAt"`
	UpdatedAt      time.Time       `bson:"updatedAt"`
}

// toDomain 把持久化结构转换为领域实体。
func (d accountDoc) toDomain() domain.Account {
	return domain.Account{
		ID: objectIDHex(d.ID), Username: d.Username, PasswordHash: d.PasswordHash,
		DisplayName: d.DisplayName, Email: d.Email, Phone: d.Phone, Status: d.Status,
		TenantID: objectIDHex(d.TenantID), DataScope: d.DataScope, PlatformAdmin: d.PlatformAdmin,
		LegalEntityIDs: objectIDHexes(d.LegalEntityIDs), OrgIDs: objectIDHexes(d.OrgIDs),
		RoleIDs: objectIDHexes(d.RoleIDs), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// accountDocFrom 把领域实体转换为持久化结构，ID 为空时生成新 ObjectId。
func accountDocFrom(account domain.Account) (accountDoc, error) {
	id, err := parseObjectID(account.ID)
	if err != nil {
		return accountDoc{}, err
	}
	if id.IsZero() {
		id = bson.NewObjectID()
	}
	tenantID, err := parseObjectID(account.TenantID)
	if err != nil {
		return accountDoc{}, err
	}
	entityIDs, err := parseObjectIDs(account.LegalEntityIDs)
	if err != nil {
		return accountDoc{}, err
	}
	orgIDs, err := parseObjectIDs(account.OrgIDs)
	if err != nil {
		return accountDoc{}, err
	}
	roleIDs, err := parseObjectIDs(account.RoleIDs)
	if err != nil {
		return accountDoc{}, err
	}
	return accountDoc{
		ID: id, Username: account.Username, PasswordHash: account.PasswordHash,
		DisplayName: account.DisplayName, Email: account.Email, Phone: account.Phone, Status: account.Status,
		TenantID: tenantID, DataScope: account.DataScope, PlatformAdmin: account.PlatformAdmin,
		LegalEntityIDs: entityIDs, OrgIDs: orgIDs, RoleIDs: roleIDs,
		CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt,
	}, nil
}

// AccountRepository 是管理端账户的 MongoDB 实现。
type AccountRepository struct {
	coll *mongo.Collection
}

// List 按租户、经营主体、状态与关键词过滤后分页。
func (r *AccountRepository) List(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Account], error) {
	query = query.Normalize()
	filter := bson.D{}
	if query.TenantID != "" {
		tenantID, err := parseObjectID(query.TenantID)
		if err != nil {
			return domain.PageResult[domain.Account]{}, err
		}
		filter = append(filter, bson.E{Key: "tenantId", Value: tenantID})
	}
	if query.LegalEntityID != "" {
		entityID, err := parseObjectID(query.LegalEntityID)
		if err != nil {
			return domain.PageResult[domain.Account]{}, err
		}
		filter = append(filter, bson.E{Key: "legalEntityIds", Value: entityID})
	}
	if len(query.LegalEntityIDs) > 0 {
		entityIDs, err := parseObjectIDs(query.LegalEntityIDs)
		if err != nil {
			return domain.PageResult[domain.Account]{}, err
		}
		filter = append(filter, bson.E{Key: "legalEntityIds", Value: bson.D{{Key: "$in", Value: entityIDs}}})
	}
	if query.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: query.Status})
	}
	if query.Search != "" {
		pattern := primitiveRegex(query.Search)
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "username", Value: pattern}},
			bson.D{{Key: "displayName", Value: pattern}},
			bson.D{{Key: "email", Value: pattern}},
			bson.D{{Key: "phone", Value: pattern}},
		}})
	}

	opts := options.Find().SetSkip(query.Offset()).SetLimit(query.Limit()).SetSort(bson.D{{Key: "_id", Value: 1}})
	items, err := r.find(ctx, filter, opts)
	if err != nil {
		return domain.PageResult[domain.Account]{}, err
	}
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return domain.PageResult[domain.Account]{}, err
	}
	return domain.PageResult[domain.Account]{Items: items, Total: total}, nil
}

// GetByID 按 ID 查询账户。
func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	oid, err := parseObjectID(id)
	if err != nil || oid.IsZero() {
		return nil, domain.ErrNotFound
	}
	return r.findOne(ctx, bson.D{{Key: "_id", Value: oid}})
}

// GetByUsername 按用户名查询账户，username 上有唯一索引。
func (r *AccountRepository) GetByUsername(ctx context.Context, username string) (*domain.Account, error) {
	return r.findOne(ctx, bson.D{{Key: "username", Value: username}})
}

// Create 写入账户，用户名冲突时返回 ErrConflict。
func (r *AccountRepository) Create(ctx context.Context, account domain.Account) (*domain.Account, error) {
	doc, err := accountDocFrom(account)
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

// Update 覆盖账户的可编辑字段。
func (r *AccountRepository) Update(ctx context.Context, account domain.Account) (*domain.Account, error) {
	doc, err := accountDocFrom(account)
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

// UpdateStatus 更新账户状态，账户不存在时返回 ErrNotFound。
func (r *AccountRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	oid, err := parseObjectID(id)
	if err != nil || oid.IsZero() {
		return domain.ErrNotFound
	}
	result, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: oid}},
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

// CountByRoles 聚合统计每个角色关联的账户数，键为角色 ID 的十六进制形式。
func (r *AccountRepository) CountByRoles(ctx context.Context, roleIDs []string) (map[string]int, error) {
	counts := map[string]int{}
	if len(roleIDs) == 0 {
		return counts, nil
	}
	oids, err := parseObjectIDs(roleIDs)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$roleIds"}},
		{{Key: "$match", Value: bson.D{{Key: "roleIds", Value: bson.D{{Key: "$in", Value: oids}}}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$roleIds"}, {Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
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

// CountByTenant 统计指定租户下的账户数量。
func (r *AccountRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	if tenantID == "" {
		return 0, nil
	}
	oid, err := parseObjectID(tenantID)
	if err != nil {
		return 0, err
	}
	total, err := r.coll.CountDocuments(ctx, bson.D{{Key: "tenantId", Value: oid}})
	return int(total), err
}

// findOne 将 mongo.ErrNoDocuments 统一翻译为领域错误。
func (r *AccountRepository) findOne(ctx context.Context, filter bson.D) (*domain.Account, error) {
	var doc accountDoc
	if err := r.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	account := doc.toDomain()
	return &account, nil
}

// find 执行查询并转换为领域实体。
func (r *AccountRepository) find(ctx context.Context, filter bson.D, opts ...options.Lister[options.FindOptions]) ([]domain.Account, error) {
	cursor, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []accountDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	items := make([]domain.Account, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.toDomain())
	}
	return items, nil
}
