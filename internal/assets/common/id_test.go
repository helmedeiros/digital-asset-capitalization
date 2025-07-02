package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateID(t *testing.T) {
	t.Run("should generate non-empty ID", func(t *testing.T) {
		id := GenerateID("test-asset")
		assert.NotEmpty(t, id, "Generated ID should not be empty")
	})

	t.Run("should generate ID with correct length", func(t *testing.T) {
		id := GenerateID("test-asset")
		assert.Len(t, id, 16, "Generated ID should be 16 characters long")
	})

	t.Run("should generate hex-encoded ID", func(t *testing.T) {
		id := GenerateID("test-asset")

		// Check if all characters are valid hex characters
		for _, char := range id {
			assert.True(t, strings.ContainsRune("0123456789abcdef", char),
				"ID should contain only hex characters, found: %c", char)
		}
	})

	t.Run("should generate different IDs for same name called at different times", func(t *testing.T) {
		// Since the function uses time.Now(), consecutive calls should produce different IDs
		id1 := GenerateID("same-name")
		id2 := GenerateID("same-name")

		assert.NotEqual(t, id1, id2, "IDs generated at different times should be different")
	})

	t.Run("should generate different IDs for different names", func(t *testing.T) {
		id1 := GenerateID("asset-1")
		id2 := GenerateID("asset-2")

		assert.NotEqual(t, id1, id2, "IDs for different names should be different")
	})

	t.Run("should handle empty name", func(t *testing.T) {
		id := GenerateID("")
		assert.NotEmpty(t, id, "Should generate ID even for empty name")
		assert.Len(t, id, 16, "Generated ID should still be 16 characters long")
	})

	t.Run("should handle special characters in name", func(t *testing.T) {
		specialNames := []string{
			"test!@#$%^&*()",
			"test with spaces",
			"test\nwith\nnewlines",
			"test-with-unicode-🚀",
		}

		for _, name := range specialNames {
			id := GenerateID(name)
			require.NotEmpty(t, id, "Should generate ID for name: %s", name)
			require.Len(t, id, 16, "ID length should be correct for name: %s", name)
		}
	})

	t.Run("should handle very long names", func(t *testing.T) {
		longName := strings.Repeat("a", 1000)
		id := GenerateID(longName)
		assert.NotEmpty(t, id)
		assert.Len(t, id, 16)
	})
}
