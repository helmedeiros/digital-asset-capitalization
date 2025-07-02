package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAsset(t *testing.T) {
	tests := []struct {
		name        string
		assetName   string
		description string
		why         string
		benefits    string
		how         string
		metrics     string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid asset",
			assetName:   "test-asset",
			description: "Test description",
			why:         "Test why",
			benefits:    "Test benefits",
			how:         "Test how",
			metrics:     "Test metrics",
			wantErr:     false,
		},
		{
			name:        "empty name",
			assetName:   "",
			description: "Test description",
			why:         "Test why",
			benefits:    "Test benefits",
			how:         "Test how",
			metrics:     "Test metrics",
			wantErr:     true,
			errMsg:      ErrEmptyName.Error(),
		},
		{
			name:        "empty description",
			assetName:   "test-asset",
			description: "",
			why:         "Test why",
			benefits:    "Test benefits",
			how:         "Test how",
			metrics:     "Test metrics",
			wantErr:     true,
			errMsg:      ErrEmptyDescription.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := NewAssetWithDetails(tt.assetName, tt.description, tt.why, tt.benefits, tt.how, tt.metrics)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.assetName, asset.Name)
			assert.Equal(t, tt.description, asset.Description)
			assert.Equal(t, tt.why, asset.Why)
			assert.Equal(t, tt.benefits, asset.Benefits)
			assert.Equal(t, tt.how, asset.How)
			assert.Equal(t, tt.metrics, asset.Metrics)
			assert.NotEmpty(t, asset.ID, "Expected non-empty ID")
			assert.Equal(t, 1, asset.Version)
		})
	}
}

func TestUpdateDescription(t *testing.T) {
	asset, err := NewAsset("test-asset", "Initial description")
	require.NoError(t, err)

	tests := []struct {
		name        string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid description",
			description: "Updated description",
			wantErr:     false,
		},
		{
			name:        "empty description",
			description: "",
			wantErr:     true,
			errMsg:      ErrEmptyDescription.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := asset.UpdateDescription(tt.description)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.description, asset.Description)
			assert.Equal(t, 2, asset.Version)
		})
	}
}

func TestUpdateDocumentation(t *testing.T) {
	asset, err := NewAsset("test-asset", "Test description")
	require.NoError(t, err)

	// Store initial time
	initialTime := asset.LastDocUpdateAt

	// Wait a bit to ensure time difference
	time.Sleep(time.Millisecond)

	// Update documentation
	asset.UpdateDocumentation()

	// Verify update
	assert.True(t, asset.LastDocUpdateAt.After(initialTime), "LastDocUpdateAt should be after initial time")
	assert.Equal(t, 2, asset.Version)
}

func TestTaskCountOperations(t *testing.T) {
	asset, err := NewAsset("test-asset", "Test description")
	require.NoError(t, err)

	// Test increment
	asset.IncrementTaskCount()
	assert.Equal(t, 1, asset.AssociatedTaskCount)
	assert.Equal(t, 2, asset.Version)

	// Test decrement
	asset.DecrementTaskCount()
	assert.Equal(t, 0, asset.AssociatedTaskCount)
	assert.Equal(t, 3, asset.Version)

	// Test decrement below zero
	asset.DecrementTaskCount()
	assert.Equal(t, 0, asset.AssociatedTaskCount)
	assert.Equal(t, 3, asset.Version)
}

func TestGenerateID(t *testing.T) {
	// Test that IDs are unique
	id1 := generateID("test-asset")
	id2 := generateID("test-asset")
	assert.NotEqual(t, id1, id2, "Generated IDs should be unique")

	// Test ID length
	assert.Len(t, id1, 16, "Expected ID length 16")
}

func TestDateStarted(t *testing.T) {
	asset, err := NewAsset("test-asset", "Test description")
	require.NoError(t, err)

	// Initially dateStarted should be zero
	assert.True(t, asset.DateStarted.IsZero(), "DateStarted should be zero initially")

	// Set dateStarted
	now := time.Now()
	asset.SetDateStarted(now)

	// Verify dateStarted was set correctly
	assert.Equal(t, now, asset.DateStarted)
	assert.Equal(t, 2, asset.Version)
}

func TestAsset_GetTaskCount(t *testing.T) {
	asset, err := NewAsset("Test Asset", "Test Description")
	require.NoError(t, err)

	t.Run("should return initial task count of 0", func(t *testing.T) {
		count := asset.GetTaskCount()
		assert.Equal(t, 0, count, "Initial task count should be 0")
	})

	t.Run("should return correct count after increment", func(t *testing.T) {
		// Reset asset state
		asset.AssociatedTaskCount = 0

		// Increment a few times
		err := asset.IncrementTaskCount()
		require.NoError(t, err)
		err = asset.IncrementTaskCount()
		require.NoError(t, err)

		count := asset.GetTaskCount()
		assert.Equal(t, 2, count, "Task count should be 2 after two increments")
	})

	t.Run("should return correct count after decrement", func(t *testing.T) {
		// Set initial count
		asset.AssociatedTaskCount = 5

		err := asset.DecrementTaskCount()
		require.NoError(t, err)

		count := asset.GetTaskCount()
		assert.Equal(t, 4, count, "Task count should be 4 after decrement from 5")
	})

	t.Run("should be thread-safe", func(t *testing.T) {
		asset.AssociatedTaskCount = 0

		// Test concurrent access
		done := make(chan bool, 2)

		go func() {
			for i := 0; i < 100; i++ {
				_ = asset.GetTaskCount()
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 50; i++ {
				_ = asset.IncrementTaskCount()
			}
			done <- true
		}()

		// Wait for both goroutines
		<-done
		<-done

		// Should not panic and should return a valid count
		count := asset.GetTaskCount()
		assert.GreaterOrEqual(t, count, 0, "Task count should be non-negative")
	})
}

func TestAsset_GetVersion(t *testing.T) {
	asset, err := NewAsset("Test Asset", "Test Description")
	require.NoError(t, err)

	t.Run("should return initial version of 1", func(t *testing.T) {
		version := asset.GetVersion()
		assert.Equal(t, 1, version, "Initial version should be 1")
	})

	t.Run("should return correct version after updates", func(t *testing.T) {
		initialVersion := asset.GetVersion()

		// Perform operations that increment version
		err := asset.UpdateDescription("New description")
		require.NoError(t, err)

		err = asset.UpdateDocumentation()
		require.NoError(t, err)

		err = asset.IncrementTaskCount()
		require.NoError(t, err)

		newVersion := asset.GetVersion()
		assert.Equal(t, initialVersion+3, newVersion, "Version should increment by 3 after 3 operations")
	})

	t.Run("should be thread-safe", func(t *testing.T) {
		// Test concurrent access to version
		done := make(chan bool, 2)

		go func() {
			for i := 0; i < 100; i++ {
				_ = asset.GetVersion()
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 10; i++ {
				_ = asset.UpdateDescription("Concurrent update")
			}
			done <- true
		}()

		// Wait for both goroutines
		<-done
		<-done

		// Should not panic and should return a valid version
		version := asset.GetVersion()
		assert.Greater(t, version, 0, "Version should be positive")
	})
}

func TestAsset_UnmarshalJSON(t *testing.T) {
	t.Run("should unmarshal valid JSON correctly", func(t *testing.T) {
		jsonData := `{
			"id": "test-id-123",
			"name": "Test Asset",
			"description": "Test Description",
			"associated_task_count": 5,
			"version": 2,
			"platform": "web",
			"status": "active",
			"is_rolled_out_100": true,
			"keywords": ["api", "service"],
			"doc_link": "https://example.com/doc",
			"why": "Test why",
			"benefits": "Test benefits",
			"how": "Test how",
			"metrics": "Test metrics",
			"created_at": "2024-01-15T10:30:00Z",
			"updated_at": "2024-01-16T10:30:00Z",
			"last_doc_update_at": "2024-01-17T10:30:00Z",
			"launch_date": "2024-01-18T10:30:00Z",
			"date_started": "2024-01-14T10:30:00Z"
		}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(jsonData))
		require.NoError(t, err, "Should unmarshal valid JSON without error")

		// Verify all fields were set correctly
		assert.Equal(t, "test-id-123", asset.ID)
		assert.Equal(t, "Test Asset", asset.Name)
		assert.Equal(t, "Test Description", asset.Description)
		assert.Equal(t, 5, asset.AssociatedTaskCount)
		assert.Equal(t, 2, asset.Version)
		assert.Equal(t, "web", asset.Platform)
		assert.Equal(t, "active", asset.Status)
		assert.True(t, asset.IsRolledOut100)
		assert.Equal(t, []string{"api", "service"}, asset.Keywords)
		assert.Equal(t, "https://example.com/doc", asset.DocLink)
		assert.Equal(t, "Test why", asset.Why)
		assert.Equal(t, "Test benefits", asset.Benefits)
		assert.Equal(t, "Test how", asset.How)
		assert.Equal(t, "Test metrics", asset.Metrics)
	})

	t.Run("should handle minimal JSON", func(t *testing.T) {
		jsonData := `{
			"name": "Minimal Asset",
			"description": "Minimal Description"
		}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(jsonData))
		require.NoError(t, err, "Should unmarshal minimal JSON without error")

		assert.Equal(t, "Minimal Asset", asset.Name)
		assert.Equal(t, "Minimal Description", asset.Description)
		assert.Equal(t, 0, asset.AssociatedTaskCount) // Should default to 0
		assert.Equal(t, 0, asset.Version)             // Should default to 0
	})

	t.Run("should return error for invalid JSON", func(t *testing.T) {
		invalidJSON := `{"name": "Test", "description": "Test", "version": "not-a-number"}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(invalidJSON))
		assert.Error(t, err, "Should return error for invalid JSON")
	})

	t.Run("should return error for malformed JSON", func(t *testing.T) {
		malformedJSON := `{"name": "Test", "description": }`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(malformedJSON))
		assert.Error(t, err, "Should return error for malformed JSON")
	})

	t.Run("should handle empty JSON object", func(t *testing.T) {
		emptyJSON := `{}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(emptyJSON))
		require.NoError(t, err, "Should handle empty JSON object")

		// All fields should have zero values
		assert.Empty(t, asset.Name)
		assert.Empty(t, asset.Description)
		assert.Equal(t, 0, asset.AssociatedTaskCount)
		assert.Equal(t, 0, asset.Version)
	})

	t.Run("should handle arrays and null values", func(t *testing.T) {
		jsonData := `{
			"name": "Test Asset",
			"description": "Test Description",
			"keywords": null,
			"platform": null
		}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(jsonData))
		require.NoError(t, err, "Should handle null values")

		assert.Equal(t, "Test Asset", asset.Name)
		assert.Equal(t, "Test Description", asset.Description)
		assert.Nil(t, asset.Keywords)
		assert.Empty(t, asset.Platform)
	})
}
