package id

import (
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// HashIDGenerator implements the IDGenerator interface using cap-asset-* ID generation
type HashIDGenerator struct{}

// NewHashIDGenerator creates a new hash-based ID generator
func NewHashIDGenerator() ports.IDGenerator {
	return &HashIDGenerator{}
}

// GenerateID creates a unique ID based on the given name in cap-asset-* format
func (g *HashIDGenerator) GenerateID(name string) string {
	// Convert name to lowercase and replace spaces with hyphens
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove special characters that aren't suitable for labels
	id = strings.ReplaceAll(id, "(", "")
	id = strings.ReplaceAll(id, ")", "")
	id = strings.ReplaceAll(id, "&", "and")
	id = strings.ReplaceAll(id, "/", "-")
	id = strings.ReplaceAll(id, ".", "-")

	return fmt.Sprintf("cap-asset-%s", id)
}
