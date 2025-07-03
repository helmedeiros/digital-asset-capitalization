package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsset(t *testing.T) {
	mother := NewAssetMother()

	t.Run("should create a valid asset", func(t *testing.T) {
		asset, err := NewAssetWithDetails("Test Asset", "Test Description", "Test Why", "Test Benefits", "Test How", "Test Metrics")
		require.NoError(t, err)

		assert.Equal(t, "Test Asset", asset.Name, "expected name to be 'Test Asset'")
		assert.Equal(t, "Test Description", asset.Description, "expected description to be 'Test Description'")
		assert.Equal(t, "Test Why", asset.Why, "expected why to be 'Test Why'")
		assert.Equal(t, "Test Benefits", asset.Benefits, "expected benefits to be 'Test Benefits'")
		assert.Equal(t, "Test How", asset.How, "expected how to be 'Test How'")
		assert.Equal(t, "Test Metrics", asset.Metrics, "expected metrics to be 'Test Metrics'")
		assert.NotEmpty(t, asset.ID, "expected ID to be set")
		assert.Len(t, asset.ID, 16, "expected ID to be 16 characters long")
		assert.False(t, asset.CreatedAt.IsZero(), "expected CreatedAt to be set")
		assert.False(t, asset.UpdatedAt.IsZero(), "expected UpdatedAt to be set")
		assert.False(t, asset.LastDocUpdateAt.IsZero(), "expected LastDocUpdateAt to be set")
		assert.Empty(t, asset.ContributionTypes, "expected ContributionTypes to be empty")
		assert.Equal(t, 0, asset.AssociatedTaskCount, "expected AssociatedTaskCount to be 0")
		assert.Equal(t, 1, asset.Version, "expected Version to be 1")
	})

	t.Run("should not create asset with empty name", func(t *testing.T) {
		asset, err := NewAsset("", "description")
		assert.ErrorIs(t, err, ErrEmptyName, "expected error to be ErrEmptyName")
		assert.Nil(t, asset, "expected asset to be nil")
	})

	t.Run("should not create asset with empty description", func(t *testing.T) {
		asset, err := NewAsset("name", "")
		assert.ErrorIs(t, err, ErrEmptyDescription, "expected error to be ErrEmptyDescription")
		assert.Nil(t, asset, "expected asset to be nil")
	})

	t.Run("should track creation and update timestamps", func(t *testing.T) {
		asset := mother.CreateValidAsset()
		createdAt := asset.CreatedAt
		updatedAt := asset.UpdatedAt
		version := asset.Version

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		err := asset.UpdateDescription("new description")
		require.NoError(t, err, "unexpected error updating description")

		assert.Equal(t, createdAt, asset.CreatedAt, "CreatedAt should not change")
		assert.NotEqual(t, updatedAt, asset.UpdatedAt, "UpdatedAt should change")
		assert.Equal(t, version+1, asset.Version, "Version should increment")
	})

	t.Run("should track last documentation update", func(t *testing.T) {
		asset := mother.CreateValidAsset()
		lastDocUpdateAt := asset.LastDocUpdateAt
		version := asset.Version

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		asset.UpdateDocumentation()

		assert.NotEqual(t, lastDocUpdateAt, asset.LastDocUpdateAt, "LastDocUpdateAt should change")
		assert.Equal(t, version+1, asset.Version, "Version should increment")
	})

	t.Run("should validate contribution types", func(t *testing.T) {
		asset := mother.CreateValidAsset()

		tests := []struct {
			name             string
			contributionType string
			expectedError    error
		}{
			{
				name:             "valid contribution type",
				contributionType: "development",
				expectedError:    nil,
			},
			{
				name:             "empty contribution type",
				contributionType: "",
				expectedError:    ErrEmptyContributionType,
			},
			{
				name:             "invalid contribution type",
				contributionType: "invalid",
				expectedError:    ErrInvalidContributionType,
			},
			{
				name:             "duplicate contribution type",
				contributionType: "development",
				expectedError:    ErrDuplicateContributionType,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := asset.AddContributionType(tt.contributionType)
				assert.ErrorIs(t, err, tt.expectedError, "unexpected error")
				if err == nil {
					assert.NotEmpty(t, asset.ContributionTypes, "expected contribution type to be added")
				}
			})
		}
	})

	t.Run("should track task count and version", func(t *testing.T) {
		asset := mother.CreateAssetWithTaskCount(3)
		version := asset.Version

		assert.Equal(t, 3, asset.AssociatedTaskCount, "expected task count to be 3")

		asset.DecrementTaskCount()
		assert.Equal(t, 2, asset.AssociatedTaskCount, "expected task count to be 2")
		assert.Equal(t, version+1, asset.Version, "Version should increment")

		asset.DecrementTaskCount()
		asset.DecrementTaskCount()
		asset.DecrementTaskCount() // Should not go below 0

		assert.Equal(t, 0, asset.AssociatedTaskCount, "expected task count to be 0")
		assert.Equal(t, version+3, asset.Version, "Version should increment")
	})

	t.Run("should generate unique IDs", func(t *testing.T) {
		asset1 := mother.CreateValidAsset()
		asset2 := mother.CreateValidAsset()

		assert.NotEqual(t, asset1.ID, asset2.ID, "expected different IDs for different assets")
	})
}

func TestAsset_MarshalJSON(t *testing.T) {
	t.Run("should marshal asset to valid JSON", func(t *testing.T) {
		asset, err := NewAsset("Test Asset", "Test Description")
		require.NoError(t, err)

		// Set some additional fields
		asset.Platform = "web"
		asset.Status = "active"
		asset.IsRolledOut100 = true
		asset.Keywords = []string{"api", "service"}
		asset.DocLink = "https://example.com/doc"

		jsonData, err := asset.MarshalJSON()
		require.NoError(t, err, "Should marshal without error")
		require.NotEmpty(t, jsonData, "JSON data should not be empty")

		// Verify it's valid JSON by unmarshaling back
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err, "Marshaled JSON should be valid")

		// Verify key fields are present
		assert.Equal(t, "Test Asset", unmarshaled["name"])
		assert.Equal(t, "Test Description", unmarshaled["description"])
		assert.Equal(t, "web", unmarshaled["platform"])
		assert.Equal(t, "active", unmarshaled["status"])
		assert.Equal(t, true, unmarshaled["is_rolled_out_100"])
		assert.Equal(t, "https://example.com/doc", unmarshaled["doc_link"])

		// Verify keywords array
		keywords, ok := unmarshaled["keywords"].([]interface{})
		require.True(t, ok, "Keywords should be an array")
		assert.Len(t, keywords, 2)
		assert.Equal(t, "api", keywords[0])
		assert.Equal(t, "service", keywords[1])
	})

	t.Run("should marshal asset with empty optional fields", func(t *testing.T) {
		asset, err := NewAsset("Minimal Asset", "Minimal Description")
		require.NoError(t, err)

		jsonData, err := asset.MarshalJSON()
		require.NoError(t, err, "Should marshal minimal asset without error")

		var unmarshaled map[string]interface{}
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err, "Should produce valid JSON")

		assert.Equal(t, "Minimal Asset", unmarshaled["name"])
		assert.Equal(t, "Minimal Description", unmarshaled["description"])
	})

	t.Run("should handle concurrent marshaling", func(t *testing.T) {
		asset, err := NewAsset("Concurrent Asset", "Concurrent Description")
		require.NoError(t, err)

		// Test concurrent marshaling
		done := make(chan bool, 2)
		var marshalErr error

		go func() {
			for i := 0; i < 100; i++ {
				_, err := asset.MarshalJSON()
				if err != nil {
					marshalErr = err
				}
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 50; i++ {
				asset.UpdateDescription("Updated description")
			}
			done <- true
		}()

		// Wait for both goroutines
		<-done
		<-done

		assert.NoError(t, marshalErr, "Concurrent marshaling should not cause errors")
	})

	t.Run("should marshal with unicode and special characters", func(t *testing.T) {
		asset, err := NewAsset("Asset with 🚀 unicode", "Description with special chars: !@#$%^&*()")
		require.NoError(t, err)

		asset.Keywords = []string{"unicode-🎯", "special-chars!@#"}
		asset.Platform = "mobile & web"

		jsonData, err := asset.MarshalJSON()
		require.NoError(t, err, "Should handle unicode and special characters")

		var unmarshaled map[string]interface{}
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err, "Should produce valid JSON with unicode")

		assert.Equal(t, "Asset with 🚀 unicode", unmarshaled["name"])
		assert.Equal(t, "Description with special chars: !@#$%^&*()", unmarshaled["description"])
		assert.Equal(t, "mobile & web", unmarshaled["platform"])
	})
}

func TestAsset_UnmarshalJSON(t *testing.T) {
	t.Run("should unmarshal complete JSON correctly", func(t *testing.T) {
		jsonData := `{
			"id": "test-id-123",
			"name": "Test Asset",
			"description": "Test Description",
			"why": "Test why",
			"benefits": "Test benefits",
			"how": "Test how",
			"metrics": "Test metrics",
			"created_at": "2024-01-15T10:30:00Z",
			"updated_at": "2024-01-16T10:30:00Z",
			"last_doc_update_at": "2024-01-17T10:30:00Z",
			"contribution_types": ["development", "documentation"],
			"associated_task_count": 5,
			"version": 2,
			"platform": "web",
			"status": "active",
			"launch_date": "2024-01-18T10:30:00Z",
			"is_rolled_out_100": true,
			"keywords": ["api", "service"],
			"doc_link": "https://example.com/doc"
		}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(jsonData))
		require.NoError(t, err, "Should unmarshal complete JSON without error")

		// Verify all fields
		assert.Equal(t, "test-id-123", asset.ID)
		assert.Equal(t, "Test Asset", asset.Name)
		assert.Equal(t, "Test Description", asset.Description)
		assert.Equal(t, "Test why", asset.Why)
		assert.Equal(t, "Test benefits", asset.Benefits)
		assert.Equal(t, "Test how", asset.How)
		assert.Equal(t, "Test metrics", asset.Metrics)
		assert.Equal(t, 5, asset.AssociatedTaskCount)
		assert.Equal(t, 2, asset.Version)
		assert.Equal(t, "web", asset.Platform)
		assert.Equal(t, "active", asset.Status)
		assert.True(t, asset.IsRolledOut100)
		assert.Equal(t, []string{"api", "service"}, asset.Keywords)
		assert.Equal(t, "https://example.com/doc", asset.DocLink)
		assert.Equal(t, []string{"development", "documentation"}, asset.ContributionTypes)
	})

	t.Run("should unmarshal minimal JSON", func(t *testing.T) {
		jsonData := `{
			"name": "Minimal Asset",
			"description": "Minimal Description"
		}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(jsonData))
		require.NoError(t, err, "Should unmarshal minimal JSON")

		assert.Equal(t, "Minimal Asset", asset.Name)
		assert.Equal(t, "Minimal Description", asset.Description)
		assert.Equal(t, 0, asset.AssociatedTaskCount)
		assert.Equal(t, 0, asset.Version)
		assert.Empty(t, asset.ContributionTypes)
	})

	t.Run("should handle null and empty values", func(t *testing.T) {
		jsonData := `{
			"name": "Test Asset",
			"description": "Test Description",
			"keywords": null,
			"contribution_types": [],
			"platform": "",
			"associated_task_count": 0
		}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(jsonData))
		require.NoError(t, err, "Should handle null and empty values")

		assert.Equal(t, "Test Asset", asset.Name)
		assert.Equal(t, "Test Description", asset.Description)
		assert.Nil(t, asset.Keywords)
		assert.Empty(t, asset.ContributionTypes)
		assert.Empty(t, asset.Platform)
		assert.Equal(t, 0, asset.AssociatedTaskCount)
	})

	t.Run("should return error for invalid JSON", func(t *testing.T) {
		invalidJSONs := []string{
			`{"name": "Test", "description": "Test", "version": "invalid"}`,
			`{"name": "Test", "description": "Test", "associated_task_count": "not-a-number"}`,
			`{"name": "Test", "description": "Test", "is_rolled_out_100": "not-a-boolean"}`,
			`{"name": "Test", "description": }`, // Malformed
			`{"name": "Test", "keywords": "should-be-array"}`,
		}

		for i, invalidJSON := range invalidJSONs {
			var asset Asset
			err := asset.UnmarshalJSON([]byte(invalidJSON))
			assert.Error(t, err, "Should return error for invalid JSON case %d", i+1)
		}
	})

	t.Run("should handle unicode and special characters", func(t *testing.T) {
		jsonData := `{
			"name": "Asset with 🚀 unicode",
			"description": "Description with special chars: !@#$%^&*()",
			"keywords": ["unicode-🎯", "special-chars!@#"],
			"platform": "mobile & web"
		}`

		var asset Asset
		err := asset.UnmarshalJSON([]byte(jsonData))
		require.NoError(t, err, "Should handle unicode and special characters")

		assert.Equal(t, "Asset with 🚀 unicode", asset.Name)
		assert.Equal(t, "Description with special chars: !@#$%^&*()", asset.Description)
		assert.Equal(t, "mobile & web", asset.Platform)
		assert.Equal(t, []string{"unicode-🎯", "special-chars!@#"}, asset.Keywords)
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
		assert.Nil(t, asset.Keywords)
		assert.Nil(t, asset.ContributionTypes)
	})

	t.Run("should be thread-safe", func(t *testing.T) {
		jsonData := `{"name": "Concurrent Asset", "description": "Concurrent Description"}`

		done := make(chan bool, 2)
		var unmarshalErr error

		go func() {
			for i := 0; i < 100; i++ {
				var asset Asset
				err := asset.UnmarshalJSON([]byte(jsonData))
				if err != nil {
					unmarshalErr = err
				}
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 50; i++ {
				var asset Asset
				_ = asset.UnmarshalJSON([]byte(jsonData))
			}
			done <- true
		}()

		<-done
		<-done

		assert.NoError(t, unmarshalErr, "Concurrent unmarshaling should not cause errors")
	})
}

func TestAssetMother(t *testing.T) {
	mother := NewAssetMother()

	t.Run("CreateAssetWithCustomValues should create asset with specified values", func(t *testing.T) {
		asset := mother.CreateAssetWithCustomValues("Custom Asset", "Custom Description")

		assert.NotNil(t, asset, "Asset should not be nil")
		assert.Equal(t, "Custom Asset", asset.Name, "Name should match")
		assert.Equal(t, "Custom Description", asset.Description, "Description should match")
		assert.NotEmpty(t, asset.ID, "ID should be set")
		assert.Equal(t, 1, asset.Version, "Version should be 1")
	})

	t.Run("CreateAssetWithContributionTypes should add contribution types", func(t *testing.T) {
		asset := mother.CreateAssetWithContributionTypes("development", "maintenance")

		assert.NotNil(t, asset, "Asset should not be nil")
		assert.Len(t, asset.ContributionTypes, 2, "Should have 2 contribution types")
		assert.Contains(t, asset.ContributionTypes, "development", "Should contain development")
		assert.Contains(t, asset.ContributionTypes, "maintenance", "Should contain maintenance")
	})

	t.Run("CreateAssetWithContributionTypes should handle empty input", func(t *testing.T) {
		asset := mother.CreateAssetWithContributionTypes()

		assert.NotNil(t, asset, "Asset should not be nil")
		assert.Empty(t, asset.ContributionTypes, "Should have no contribution types")
	})

	t.Run("CreateAssetWithCustomTimestamps should set custom timestamps", func(t *testing.T) {
		createdAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
		lastDocUpdateAt := time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC)

		asset := mother.CreateAssetWithCustomTimestamps(createdAt, updatedAt, lastDocUpdateAt)

		assert.NotNil(t, asset, "Asset should not be nil")
		assert.Equal(t, createdAt, asset.CreatedAt, "CreatedAt should match")
		assert.Equal(t, updatedAt, asset.UpdatedAt, "UpdatedAt should match")
		assert.Equal(t, lastDocUpdateAt, asset.LastDocUpdateAt, "LastDocUpdateAt should match")
	})

	t.Run("CreateAssetWithVersion should set custom version", func(t *testing.T) {
		asset := mother.CreateAssetWithVersion(5)

		assert.NotNil(t, asset, "Asset should not be nil")
		assert.Equal(t, 5, asset.Version, "Version should be 5")
	})
}
