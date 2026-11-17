package mongostore

import (
	"crypto/rand"
	"encoding/hex"
)

// newID 生成带业务前缀的 128 位随机 ID，保持跨集合可读性。
func newID(prefix string) (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}
