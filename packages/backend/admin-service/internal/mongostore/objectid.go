package mongostore

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/unoadmin/admin-service/internal/domain"
)

// parseObjectID 把接口层传入的十六进制 ID 转为 ObjectID。
//
// 空字符串代表「无引用」，统一落库为 NilObjectID，方便按 ref 字段过滤。
func parseObjectID(id string) (bson.ObjectID, error) {
	if id == "" {
		return bson.NilObjectID, nil
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.NilObjectID, domain.ErrInvalid
	}
	return oid, nil
}

// parseObjectIDs 批量转换 ID，保持原有顺序并丢弃空值。
func parseObjectIDs(ids []string) ([]bson.ObjectID, error) {
	result := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		oid, err := parseObjectID(id)
		if err != nil {
			return nil, err
		}
		if !oid.IsZero() {
			result = append(result, oid)
		}
	}
	return result, nil
}

// objectIDHex 把 ObjectID 转为前端与领域层使用的十六进制字符串。
func objectIDHex(id bson.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}

// objectIDHexes 批量转换 ObjectID，结果始终非 nil。
func objectIDHexes(ids []bson.ObjectID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if !id.IsZero() {
			result = append(result, id.Hex())
		}
	}
	return result
}
