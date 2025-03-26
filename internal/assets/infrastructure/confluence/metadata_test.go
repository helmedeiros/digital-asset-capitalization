package confluence

import (
	"reflect"
	"testing"
	"time"
)

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *PageMetadata
		wantErr  bool
	}{
		{
			name: "extract all metadata with API format labels",
			content: `<table>
				<tr><td><strong>Why are we doing this?</strong></td><td><p>Test description</p></td></tr>
				<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
				<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
				<tr><td><strong>Launch date</strong></td><td><p>March 4, 2022</p></td></tr>
			</table>
			{"metadata":{"labels":{"results":[{"name":"test-label"},{"name":"cap-asset-test-asset"}]}}}`,
			expected: &PageMetadata{
				Description: "Test description",
				Platform:    "Test Platform",
				Status:      "in development",
				LaunchDate:  time.Date(2022, 3, 4, 0, 0, 0, 0, time.UTC),
				Keywords:    []string{"test-label"},
				Identifier:  "cap-asset-test-asset",
			},
			wantErr: false,
		},
		{
			name: "extract all metadata with HTML format labels",
			content: `<table>
				<tr><td><strong>Why are we doing this?</strong></td><td><p>Test description</p></td></tr>
				<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
				<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
				<tr><td><strong>Launch date</strong></td><td><p>March 4, 2022</p></td></tr>
			</table>
			<div class="labels">{"label":"test-label"},{"label":"cap-asset-test-asset"}</div>`,
			expected: &PageMetadata{
				Description: "Test description",
				Platform:    "Test Platform",
				Status:      "in development",
				LaunchDate:  time.Date(2022, 3, 4, 0, 0, 0, 0, time.UTC),
				Keywords:    []string{"test-label"},
				Identifier:  "cap-asset-test-asset",
			},
			wantErr: false,
		},
		{
			name: "extract all metadata fields",
			content: `<table>
				<tr><td><strong>Why are we doing this?</strong></td><td><p>Test description</p></td></tr>
				<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
				<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
				<tr><td><strong>Launch date</strong></td><td><p>since 2022</p></td></tr>
			</table>
			<div class="labels">{"label":"test-label"},{"label":"cap-asset-test-asset"}</div>`,
			expected: &PageMetadata{
				Description: "Test description",
				Platform:    "Test Platform",
				Status:      "in development",
				LaunchDate:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				Keywords:    []string{"test-label"},
				Identifier:  "cap-asset-test-asset",
			},
			wantErr: false,
		},
		{
			name: "fallback to economic benefits",
			content: `<table>
				<tr><td><strong>Economic benefits</strong></td><td><p>Fallback description</p></td></tr>
			</table>
			<div class="labels">{"label":"test-label"},{"label":"cap-asset-test-asset"}</div>`,
			expected: &PageMetadata{
				Description: "Fallback description",
				Keywords:    []string{"test-label"},
				Identifier:  "cap-asset-test-asset",
			},
			wantErr: false,
		},
		{
			name: "extract status from macro",
			content: `<table>
				<tr><td><strong>Status</strong></td><td><p><ac:structured-macro ac:name="status"><ac:parameter ac:name="title">in development</ac:parameter></ac:structured-macro></p></td></tr>
			</table>
			<div class="labels">{"label":"test-label"},{"label":"cap-asset-test-asset"}</div>`,
			expected: &PageMetadata{
				Status:     "in development",
				Keywords:   []string{"test-label"},
				Identifier: "cap-asset-test-asset",
			},
			wantErr: false,
		},
		{
			name: "no asset identifier",
			content: `<table>
				<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
			</table>
			<div class="labels">{"label":"test-label"}</div>`,
			expected: &PageMetadata{
				Status:     "in development",
				Keywords:   []string{"test-label"},
				Identifier: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &Adapter{}
			got, err := adapter.extractMetadata(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if got.Description != tt.expected.Description {
				t.Errorf("Description = %v, want %v", got.Description, tt.expected.Description)
			}
			if got.Platform != tt.expected.Platform {
				t.Errorf("Platform = %v, want %v", got.Platform, tt.expected.Platform)
			}
			if got.Status != tt.expected.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.expected.Status)
			}
			if !got.LaunchDate.Equal(tt.expected.LaunchDate) {
				t.Errorf("LaunchDate = %v, want %v", got.LaunchDate, tt.expected.LaunchDate)
			}
			if got.IsRolledOut100 != tt.expected.IsRolledOut100 {
				t.Errorf("IsRolledOut100 = %v, want %v", got.IsRolledOut100, tt.expected.IsRolledOut100)
			}
			if !reflect.DeepEqual(got.Keywords, tt.expected.Keywords) {
				t.Errorf("Keywords = %v, want %v", got.Keywords, tt.expected.Keywords)
			}
			if got.Identifier != tt.expected.Identifier {
				t.Errorf("Identifier = %v, want %v", got.Identifier, tt.expected.Identifier)
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
