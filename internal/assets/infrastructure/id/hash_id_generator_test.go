package id

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashIdGenerator_GenerateID(t *testing.T) {
	generator := NewHashIDGenerator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "generates ID for asset name",
			input:    "test-asset",
			expected: "cap-asset-test-asset",
		},
		{
			name:     "generates ID for empty string",
			input:    "",
			expected: "cap-asset-",
		},
		{
			name:     "generates ID with special characters",
			input:    "Test Asset (Special)",
			expected: "cap-asset-test-asset-special",
		},
		{
			name:     "handles complex name",
			input:    "Very/Long Asset Name & More",
			expected: "cap-asset-very-long-asset-name-and-more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := generator.GenerateID(tt.input)

			// Check that ID matches expected format
			assert.Equal(t, tt.expected, id)

			// Check that ID starts with cap-asset-
			assert.True(t, strings.HasPrefix(id, "cap-asset-"), "ID should start with cap-asset-")
		})
	}
}

func TestHashIdGenerator_GenerateID_Deterministic(t *testing.T) {
	generator := NewHashIDGenerator()

	// Same inputs should produce same IDs (deterministic)
	id1 := generator.GenerateID("test")
	id2 := generator.GenerateID("test")

	assert.Equal(t, id1, id2, "Same inputs should produce same IDs")
	assert.Equal(t, "cap-asset-test", id1)

	// Different inputs should produce different IDs
	id3 := generator.GenerateID("asset-1")
	id4 := generator.GenerateID("asset-2")

	assert.NotEqual(t, id3, id4, "Different inputs should produce different IDs")
	assert.Equal(t, "cap-asset-asset-1", id3)
	assert.Equal(t, "cap-asset-asset-2", id4)
}
