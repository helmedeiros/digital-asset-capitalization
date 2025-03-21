package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchResponse_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"issues": [
			{
				"key": "TEST-1",
				"fields": {
					"summary": "Test Issue",
					"status": {
						"name": "In Progress"
					},
					"project": {
						"key": "TEST"
					},
					"sprint": {
						"name": "Sprint 1"
					},
					"created": "2024-03-20T10:00:00.000Z",
					"updated": "2024-03-20T11:00:00.000Z",
					"description": "Test Description"
				}
			}
		]
	}`

	var response SearchResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	require.NoError(t, err, "Should unmarshal JSON successfully")

	assert.Len(t, response.Issues, 1, "Should have one issue")
	issue := response.Issues[0]
	assert.Equal(t, "TEST-1", issue.Key, "Issue key should match")
	assert.Equal(t, "Test Issue", issue.Fields.Summary, "Issue summary should match")
	assert.Equal(t, "In Progress", issue.Fields.Status.Name, "Issue status should match")
	assert.Equal(t, "TEST", issue.Fields.Project.Key, "Project key should match")
	assert.Equal(t, "Sprint 1", issue.Fields.Sprint.Name, "Sprint name should match")
	assert.Equal(t, "Test Description", issue.Fields.Description, "Description should match")

	// Test timestamp parsing
	created, err := time.Parse(time.RFC3339, issue.Fields.Created)
	require.NoError(t, err, "Should parse created timestamp")
	assert.Equal(t, "2024-03-20T10:00:00Z", created.Format(time.RFC3339), "Created timestamp should match")

	updated, err := time.Parse(time.RFC3339, issue.Fields.Updated)
	require.NoError(t, err, "Should parse updated timestamp")
	assert.Equal(t, "2024-03-20T11:00:00Z", updated.Format(time.RFC3339), "Updated timestamp should match")
}
