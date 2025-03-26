package confluence

import (
	"testing"
	"time"
)

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *PageMetadata
	}{
		{
			name: "extract all metadata fields",
			content: `
				<table>
					<tr><td><strong>Why are we doing this?</strong></td><td><p>Test description</p></td></tr>
					<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
					<tr><td><strong>Status</strong></td><td><p>in continuous development</p></td></tr>
					<tr><td><strong>Launch date</strong></td><td><p>since 2022</p></td></tr>
				</table>
				<div>100% of traffic</div>
				<div>"label":"test-keyword"</div>
			`,
			expected: &PageMetadata{
				Description:    "Test description",
				Platform:       "Test Platform",
				Status:         "in continuous development",
				LaunchDate:     time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				IsRolledOut100: true,
				Keywords:       []string{"test-keyword"},
			},
		},
		{
			name: "fallback to economic benefits",
			content: `
				<table>
					<tr><td><strong>Economic benefits</strong></td><td><p>Fallback description</p></td></tr>
				</table>
			`,
			expected: &PageMetadata{
				Description: "Fallback description",
			},
		},
		{
			name: "extract status from macro",
			content: `
				<table>
					<tr><td><strong>Status</strong></td><td><ac:structured-macro ac:name="status" ac:schema-version="1"><ac:parameter ac:name="title">in development</ac:parameter></ac:structured-macro></td></tr>
				</table>
			`,
			expected: &PageMetadata{
				Status: "in development",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &Adapter{}
			metadata, err := adapter.extractMetadata(tt.content)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if metadata.Description != tt.expected.Description {
				t.Errorf("Description = %v, want %v", metadata.Description, tt.expected.Description)
			}
			if metadata.Platform != tt.expected.Platform {
				t.Errorf("Platform = %v, want %v", metadata.Platform, tt.expected.Platform)
			}
			if metadata.Status != tt.expected.Status {
				t.Errorf("Status = %v, want %v", metadata.Status, tt.expected.Status)
			}
			if !metadata.LaunchDate.Equal(tt.expected.LaunchDate) {
				t.Errorf("LaunchDate = %v, want %v", metadata.LaunchDate, tt.expected.LaunchDate)
			}
			if metadata.IsRolledOut100 != tt.expected.IsRolledOut100 {
				t.Errorf("IsRolledOut100 = %v, want %v", metadata.IsRolledOut100, tt.expected.IsRolledOut100)
			}
			if len(metadata.Keywords) != len(tt.expected.Keywords) {
				t.Errorf("Keywords length = %v, want %v", len(metadata.Keywords), len(tt.expected.Keywords))
			}
		})
	}
}

func TestExtractTableValue(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		header   string
		expected string
	}{
		{
			name:     "extract value from standard table",
			content:  `<table><tr><td><strong>Test</strong></td><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "Value",
		},
		{
			name:     "extract value from highlighted table",
			content:  `<table><tr><td data-highlight-colour="#f4f5f7"><p><strong>Test</strong></td><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "Value",
		},
		{
			name:     "header not found",
			content:  `<table><tr><td><strong>Other</strong></td><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTableValue(tt.content, tt.header)
			if result != tt.expected {
				t.Errorf("extractTableValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove HTML tags",
			input:    "<p>Test <strong>content</strong></p>",
			expected: "Test content",
		},
		{
			name:     "decode HTML entities",
			input:    "Test &amp; content &nbsp; here",
			expected: "Test & content here",
		},
		{
			name:     "clean up whitespace",
			input:    "Test   content\n\there",
			expected: "Test content here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanHTML(tt.input)
			if result != tt.expected {
				t.Errorf("cleanHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractLabels(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "extract single label",
			content:  `"label":"test-label"`,
			expected: []string{"test-label"},
		},
		{
			name:     "extract multiple labels",
			content:  `"label":"label1" "label":"label2"`,
			expected: []string{"label1", "label2"},
		},
		{
			name:     "exclude global labels",
			content:  `"label":"global-label" "label":"test-label"`,
			expected: []string{"test-label"},
		},
		{
			name:     "no labels",
			content:  "no labels here",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLabels(tt.content)
			if len(result) != len(tt.expected) {
				t.Errorf("extractLabels() length = %v, want %v", len(result), len(tt.expected))
			}
			// Compare slices ignoring order
			expectedMap := make(map[string]bool)
			for _, label := range tt.expected {
				expectedMap[label] = true
			}
			for _, label := range result {
				if !expectedMap[label] {
					t.Errorf("unexpected label: %v", label)
				}
			}
		})
	}
}
