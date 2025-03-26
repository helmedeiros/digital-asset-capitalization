package common

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// GenerateID creates a unique ID based on the given name and timestamp
func GenerateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
