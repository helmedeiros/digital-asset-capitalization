package api

import (
	"encoding/json"
	"strings"
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

func TestDescription_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		jsonData     string
		expectedText string
		expectError  bool
	}{
		{
			name:         "string description",
			jsonData:     `"This is a simple text description"`,
			expectedText: "This is a simple text description",
			expectError:  false,
		},
		{
			name: "complex ADF description",
			jsonData: `{
				"type": "doc",
				"version": 1,
				"content": [
					{
						"type": "paragraph",
						"content": [
							{
								"type": "text",
								"text": "This is complex content"
							}
						]
					}
				]
			}`,
			expectedText: "This is complex content",
			expectError:  false,
		},
		{
			name:         "empty string description",
			jsonData:     `""`,
			expectedText: "",
			expectError:  false,
		},
		{
			name: "empty complex description",
			jsonData: `{
				"type": "doc",
				"version": 1,
				"content": []
			}`,
			expectedText: "",
			expectError:  false,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var desc Description
			err := json.Unmarshal([]byte(tt.jsonData), &desc)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedText, desc.ExtractAllText())
		})
	}
}

func TestDescriptionContent_extractText(t *testing.T) {
	tests := []struct {
		name         string
		content      DescriptionContent
		expectedText string
	}{
		{
			name: "text content",
			content: DescriptionContent{
				Type: "text",
				Text: "Hello world",
			},
			expectedText: "Hello world",
		},
		{
			name: "paragraph with nested text",
			content: DescriptionContent{
				Type: "paragraph",
				Content: []DescriptionContent{
					{
						Type: "text",
						Text: "Paragraph text",
					},
				},
			},
			expectedText: "Paragraph text ",
		},
		{
			name: "nested content",
			content: DescriptionContent{
				Type: "unknown",
				Content: []DescriptionContent{
					{
						Type: "text",
						Text: "Nested text",
					},
				},
			},
			expectedText: "Nested text",
		},
		{
			name: "empty content",
			content: DescriptionContent{
				Type: "text",
				Text: "",
			},
			expectedText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result strings.Builder
			tt.content.extractText(&result)
			assert.Equal(t, tt.expectedText, result.String())
		})
	}
}

// fieldsWith constructs a Fields whose RawFields map matches what
// Fields.UnmarshalJSON would have populated, without going through full
// JSON unmarshaling. Lets the table tests assert on each custom-field
// getter directly.
func fieldsWith(raw map[string]interface{}) *Fields {
	return &Fields{RawFields: raw}
}

func TestFields_GetTPDBusinessUnits(t *testing.T) {
	const fieldID = "customfield_tpd"

	tests := []struct {
		name    string
		fieldID string
		fields  *Fields
		want    []string
	}{
		{
			name:    "empty field ID returns nil",
			fieldID: "",
			fields:  fieldsWith(map[string]interface{}{fieldID: []interface{}{}}),
			want:    nil,
		},
		{
			name:    "nil RawFields returns nil",
			fieldID: fieldID,
			fields:  &Fields{},
			want:    nil,
		},
		{
			name:    "missing key returns nil",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{"other": "value"}),
			want:    nil,
		},
		{
			name:    "nil value returns nil",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: nil}),
			want:    nil,
		},
		{
			name:    "wrong type returns nil",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: "scalar instead of array"}),
			want:    nil,
		},
		{
			name:    "extracts values from objects with value key",
			fieldID: fieldID,
			fields: fieldsWith(map[string]interface{}{
				fieldID: []interface{}{
					map[string]interface{}{"value": "Payments"},
					map[string]interface{}{"value": "Search"},
				},
			}),
			want: []string{"Payments", "Search"},
		},
		{
			name:    "skips non-object items and objects missing value",
			fieldID: fieldID,
			fields: fieldsWith(map[string]interface{}{
				fieldID: []interface{}{
					"scalar item",
					map[string]interface{}{"label": "no value key"},
					map[string]interface{}{"value": ""},
					map[string]interface{}{"value": "Mobility"},
				},
			}),
			want: []string{"Mobility"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.fields.GetTPDBusinessUnits(tt.fieldID))
		})
	}
}

func TestFields_GetEngineeringHours(t *testing.T) {
	const fieldID = "customfield_hours"

	floatPtr := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		fieldID string
		fields  *Fields
		want    *float64
	}{
		{
			name:    "empty field ID returns nil",
			fieldID: "",
			fields:  fieldsWith(map[string]interface{}{fieldID: 42.0}),
			want:    nil,
		},
		{
			name:    "nil RawFields returns nil",
			fieldID: fieldID,
			fields:  &Fields{},
			want:    nil,
		},
		{
			name:    "missing key returns nil",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{"other": 1.0}),
			want:    nil,
		},
		{
			name:    "nil value returns nil",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: nil}),
			want:    nil,
		},
		{
			name:    "float64 returns pointer to value",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: 7.5}),
			want:    floatPtr(7.5),
		},
		{
			name:    "int is converted to float64",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: 12}),
			want:    floatPtr(12),
		},
		{
			name:    "unsupported type returns nil",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: "12"}),
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.GetEngineeringHours(tt.fieldID)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tt.want, *got)
		})
	}
}

func TestFields_GetWorkStream(t *testing.T) {
	const fieldID = "customfield_workstream"

	tests := []struct {
		name    string
		fieldID string
		fields  *Fields
		want    string
	}{
		{
			name:    "empty field ID returns empty",
			fieldID: "",
			fields:  fieldsWith(map[string]interface{}{fieldID: map[string]interface{}{"value": "X"}}),
			want:    "",
		},
		{
			name:    "nil RawFields returns empty",
			fieldID: fieldID,
			fields:  &Fields{},
			want:    "",
		},
		{
			name:    "missing key returns empty",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{"other": map[string]interface{}{"value": "X"}}),
			want:    "",
		},
		{
			name:    "nil value returns empty",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: nil}),
			want:    "",
		},
		{
			name:    "wrong type returns empty",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: []interface{}{"X"}}),
			want:    "",
		},
		{
			name:    "object without value key returns empty",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: map[string]interface{}{"label": "X"}}),
			want:    "",
		},
		{
			name:    "returns value string",
			fieldID: fieldID,
			fields:  fieldsWith(map[string]interface{}{fieldID: map[string]interface{}{"value": "Pricing"}}),
			want:    "Pricing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.fields.GetWorkStream(tt.fieldID))
		})
	}
}
