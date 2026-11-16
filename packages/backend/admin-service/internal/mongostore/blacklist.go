package mongostore

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// blacklistEntry 是令牌黑名单的持久化结构，expiresAt 上建有 TTL 索引。
type blacklistEntry struct {
	ID        string    `bson:"_id"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

// TokenBlacklist 是令牌黑名单的 MongoDB 实现。
type TokenBlacklist struct {
	coll *mongo.Collection
}

// IsRevoked 判断令牌 ID 是否已被吊销。
func (b *TokenBlacklist) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	err := b.coll.FindOne(ctx, bson.D{{Key: "_id", Value: tokenID}}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Add 写入吊销记录；重复调用幂等。
func (b *TokenBlacklist) Add(ctx context.Context, tokenID string, expiresAt time.Time) error {
	entry := blacklistEntry{ID: tokenID, ExpiresAt: expiresAt}
	_, err := b.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: tokenID}},
		bson.D{{Key: "$setOnInsert", Value: entry}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}
