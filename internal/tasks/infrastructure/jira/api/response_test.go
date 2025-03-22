package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchResult_UnmarshalJSON(t *testing.T) {
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
					"customfield_10100": [
						{
							"id": 1,
							"name": "Sprint 1",
							"state": "active",
							"startDate": "2024-03-20T10:00:00.000Z",
							"endDate": "2024-03-27T10:00:00.000Z",
							"boardId": 1,
							"goal": "Test sprint goal"
						}
					],
					"created": "2024-03-20T10:00:00.000Z",
					"updated": "2024-03-20T11:00:00.000Z",
					"description": {
						"type": "doc",
						"version": 1,
						"content": [
							{
								"type": "paragraph",
								"content": [
									{
										"type": "text",
										"text": "Test Description"
									}
								]
							}
						]
					}
				}
			}
		]
	}`

	var response SearchResult
	err := json.Unmarshal([]byte(jsonData), &response)
	require.NoError(t, err, "Should unmarshal JSON successfully")

	assert.Len(t, response.Issues, 1, "Should have one issue")
	issue := response.Issues[0]
	assert.Equal(t, "TEST-1", issue.Key, "Issue key should match")
	assert.Equal(t, "Test Issue", issue.Fields.Summary, "Issue summary should match")
	assert.Equal(t, "In Progress", issue.Fields.Status.Name, "Issue status should match")
	assert.Equal(t, "TEST", issue.Fields.Project.Key, "Project key should match")
	assert.Len(t, issue.Fields.Sprint, 1, "Should have one sprint")
	assert.Equal(t, "Sprint 1", issue.Fields.Sprint[0].Name, "Sprint name should match")
	assert.Equal(t, "active", issue.Fields.Sprint[0].State, "Sprint state should match")
	assert.Equal(t, "Test sprint goal", issue.Fields.Sprint[0].Goal, "Sprint goal should match")

	// Test description content
	assert.Len(t, issue.Fields.Description.Content, 1, "Should have one paragraph")
	assert.Equal(t, "paragraph", issue.Fields.Description.Content[0].Type, "Content type should be paragraph")
	assert.Len(t, issue.Fields.Description.Content[0].Content, 1, "Should have one text element")
	assert.Equal(t, "text", issue.Fields.Description.Content[0].Content[0].Type, "Text type should be text")
	assert.Equal(t, "Test Description", issue.Fields.Description.Content[0].Content[0].Text, "Text content should match")

	// Test timestamp parsing
	created, err := time.Parse(time.RFC3339, issue.Fields.Created)
	require.NoError(t, err, "Should parse created timestamp")
	assert.Equal(t, "2024-03-20T10:00:00Z", created.Format(time.RFC3339), "Created timestamp should match")

	updated, err := time.Parse(time.RFC3339, issue.Fields.Updated)
	require.NoError(t, err, "Should parse updated timestamp")
	assert.Equal(t, "2024-03-20T11:00:00Z", updated.Format(time.RFC3339), "Updated timestamp should match")
}
