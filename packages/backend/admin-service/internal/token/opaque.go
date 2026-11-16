package token

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateOpaqueToken 生成不透明随机令牌，用于刷新令牌。
func GenerateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
