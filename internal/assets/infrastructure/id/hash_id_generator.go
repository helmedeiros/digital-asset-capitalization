package id

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// HashIDGenerator implements the IDGenerator interface using hash-based ID generation
type HashIDGenerator struct{}

// NewHashIDGenerator creates a new hash-based ID generator
func NewHashIDGenerator() ports.IDGenerator {
	return &HashIDGenerator{}
}

// GenerateID creates a unique ID based on the given name and timestamp
func (g *HashIDGenerator) GenerateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
