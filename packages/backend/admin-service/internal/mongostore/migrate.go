package mongostore

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// legacyRefField 描述需要随主键一起迁移的引用字段。
//
// array 为 true 表示该字段保存的是 ObjectId 数组（如 legalEntityIds、path）。
type legacyRefField struct {
	name  string
	array bool
}

// legacyCollections 列出需要迁移主键的集合及其引用字段。
//
// 只包含当前仓储层仍在读写的集合；token_blacklist 的 tokenId 不是文档主键，
// 因此不参与迁移。
var legacyCollections = []struct {
	name   string
	fields []legacyRefField
}{
	{name: "tenants", fields: nil},
	{name: "legal_entities", fields: []legacyRefField{{name: "tenantId"}, {name: "parentId"}}},
	{name: "orgs", fields: []legacyRefField{{name: "tenantId"}, {name: "legalEntityId"}, {name: "parentId"}, {name: "path", array: true}}},
	{name: "accounts", fields: []legacyRefField{{name: "tenantId"}, {name: "legalEntityIds", array: true}, {name: "orgIds", array: true}, {name: "roleIds", array: true}}},
	{name: "roles", fields: []legacyRefField{{name: "tenantId"}, {name: "legalEntityId"}}},
	{name: "permissions", fields: []legacyRefField{{name: "parentId"}}},
}

// migrateLegacyStringIDs 把历史字符串主键迁移为 ObjectId，并同步转换引用字段。
//
// 每个旧主键生成新的 ObjectId；迁移后的文档带 `_legacyId` 标记，
// 中途失败重跑时可以据此复用已生成的主键，避免产生重复记录。
func (s *Store) migrateLegacyStringIDs(ctx context.Context) error {
	mapping := map[string]bson.ObjectID{}
	for _, item := range legacyCollections {
		ids, err := s.legacyStringIDs(ctx, item.name)
		if err != nil {
			return err
		}
		for _, id := range ids {
			existing, found, err := s.migratedID(ctx, item.name, id)
			if err != nil {
				return err
			}
			if found {
				mapping[id] = existing
				continue
			}
			mapping[id] = bson.NewObjectID()
		}
	}
	if len(mapping) == 0 {
		return nil
	}

	// 迁移期间新旧文档可能短暂共存，先移除索引避免唯一约束冲突。
	for _, item := range legacyCollections {
		if err := s.DB.Collection(item.name).Indexes().DropAll(ctx); err != nil {
			return fmt.Errorf("drop indexes for %s: %w", item.name, err)
		}
	}

	for _, item := range legacyCollections {
		if err := s.migrateCollection(ctx, item.name, item.fields, mapping); err != nil {
			return err
		}
	}

	// 清理迁移标记，恢复干净的业务文档结构。
	for _, item := range legacyCollections {
		if _, err := s.DB.Collection(item.name).UpdateMany(ctx,
			bson.D{{Key: "_legacyId", Value: bson.D{{Key: "$exists", Value: true}}}},
			bson.D{{Key: "$unset", Value: bson.D{{Key: "_legacyId", Value: ""}}}},
		); err != nil {
			return fmt.Errorf("cleanup migration marker in %s: %w", item.name, err)
		}
	}
	return nil
}

// migratedID 查询某个旧主键是否已经迁移过，返回迁移后的 ObjectId。
func (s *Store) migratedID(ctx context.Context, collection string, legacyID string) (bson.ObjectID, bool, error) {
	var doc struct {
		ID bson.ObjectID `bson:"_id"`
	}
	err := s.DB.Collection(collection).FindOne(ctx, bson.D{{Key: "_legacyId", Value: legacyID}}).Decode(&doc)
	if err == nil {
		return doc.ID, true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return bson.NilObjectID, false, nil
	}
	return bson.NilObjectID, false, fmt.Errorf("lookup migrated %s %s: %w", collection, legacyID, err)
}

// legacyStringIDs 返回集合中仍为字符串类型的 _id。
func (s *Store) legacyStringIDs(ctx context.Context, collection string) ([]string, error) {
	cursor, err := s.DB.Collection(collection).Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$type", Value: "string"}}}})
	if err != nil {
		return nil, fmt.Errorf("scan legacy ids in %s: %w", collection, err)
	}
	defer cursor.Close(ctx)

	var docs []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode legacy ids in %s: %w", collection, err)
	}
	ids := make([]string, 0, len(docs))
	for _, doc := range docs {
		ids = append(ids, doc.ID)
	}
	return ids, nil
}

// migrateCollection 逐条转换集合中的字符串主键文档。
func (s *Store) migrateCollection(ctx context.Context, collection string, fields []legacyRefField, mapping map[string]bson.ObjectID) error {
	coll := s.DB.Collection(collection)
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$type", Value: "string"}}}}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("scan %s: %w", collection, err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return fmt.Errorf("decode %s: %w", collection, err)
	}

	for _, doc := range docs {
		oldID, _ := doc["_id"].(string)
		newID, ok := mapping[oldID]
		if !ok {
			continue
		}
		converted := bson.M{}
		for key, value := range doc {
			if key == "_id" {
				continue
			}
			converted[key] = value
		}
		converted["_id"] = newID
		// 标记旧主键，便于迁移中断后重跑时复用同一个 ObjectId。
		converted["_legacyId"] = oldID
		for _, field := range fields {
			if value, exists := doc[field.name]; exists {
				converted[field.name] = convertLegacyRef(value, field.array, mapping)
			}
		}
		if _, err := coll.InsertOne(ctx, converted); err != nil && !mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("insert migrated %s %s: %w", collection, oldID, err)
		}
	}

	if _, err := coll.DeleteMany(ctx, filter); err != nil {
		return fmt.Errorf("delete legacy %s: %w", collection, err)
	}
	return nil
}

// convertLegacyRef 把字符串引用转换为 ObjectId；指向不存在记录的引用按空值处理。
func convertLegacyRef(value any, array bool, mapping map[string]bson.ObjectID) any {
	if !array {
		raw, ok := value.(string)
		if !ok {
			return value
		}
		if oid, exists := mapping[raw]; exists {
			return oid
		}
		return bson.NilObjectID
	}

	items := toStringSlice(value)
	oids := make([]bson.ObjectID, 0, len(items))
	for _, item := range items {
		if oid, exists := mapping[item]; exists {
			oids = append(oids, oid)
		}
	}
	return oids
}

// toStringSlice 兼容驱动返回的 bson.A 与 []string。
func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case bson.A:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if raw, ok := item.(string); ok {
				result = append(result, raw)
			}
		}
		return result
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if raw, ok := item.(string); ok {
				result = append(result, raw)
			}
		}
		return result
	case []string:
		return typed
	default:
		return nil
	}
}
