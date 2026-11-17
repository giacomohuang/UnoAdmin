package mongostore

import (
	"regexp"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// primitiveRegex 构造不区分大小写的字面量正则，避免搜索词中的正则元字符被解释。
func primitiveRegex(search string) bson.Regex {
	return bson.Regex{Pattern: regexp.QuoteMeta(search), Options: "i"}
}
