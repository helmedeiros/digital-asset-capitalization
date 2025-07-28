package id

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashIdGenerator_GenerateID(t *testing.T) {
	generator := NewHashIDGenerator()

	tests := []struct {
		name     string
		input    string
		expected int // expected length
	}{
		{
			name:     "generates ID for asset name",
			input:    "test-asset",
			expected: 16,
		},
		{
			name:     "generates ID for empty string",
			input:    "",
			expected: 16,
		},
		{
			name:     "generates ID for long name",
			input:    "very-long-asset-name-with-many-characters",
			expected: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := generator.GenerateID(tt.input)

			// Check that ID has expected length
			assert.Equal(t, tt.expected, len(id))

			// Check that ID is not empty
			assert.NotEmpty(t, id)

			// Check that ID is hexadecimal
			assert.Regexp(t, "^[a-f0-9]+$", id)
		})
	}
}

func TestHashIdGenerator_GenerateID_Uniqueness(t *testing.T) {
	generator := NewHashIDGenerator()

	// Generate multiple IDs for the same input
	// They should be different due to timestamp
	id1 := generator.GenerateID("test")
	id2 := generator.GenerateID("test")

	assert.NotEqual(t, id1, id2, "IDs should be unique even for same input due to timestamp")
}

func TestHashIdGenerator_GenerateID_Deterministic(t *testing.T) {
	generator := NewHashIDGenerator()

	// Different inputs should produce different IDs
	id1 := generator.GenerateID("asset-1")
	id2 := generator.GenerateID("asset-2")

	assert.NotEqual(t, id1, id2, "Different inputs should produce different IDs")
}
