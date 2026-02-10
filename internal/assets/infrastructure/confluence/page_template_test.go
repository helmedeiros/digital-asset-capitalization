package confluence

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestGetStatusBadge(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedTitle  string
		expectedColour string
	}{
		{
			name:           "Live status",
			status:         "live",
			expectedTitle:  "Live",
			expectedColour: "Green",
		},
		{
			name:           "Production status",
			status:         "production",
			expectedTitle:  "Live",
			expectedColour: "Green",
		},
		{
			name:           "Active status",
			status:         "active",
			expectedTitle:  "Live",
			expectedColour: "Green",
		},
		{
			name:           "Beta status",
			status:         "beta",
			expectedTitle:  "Beta",
			expectedColour: "Blue",
		},
		{
			name:           "Pilot status",
			status:         "pilot",
			expectedTitle:  "Beta",
			expectedColour: "Blue",
		},
		{
			name:           "Development status",
			status:         "development",
			expectedTitle:  "Development",
			expectedColour: "Yellow",
		},
		{
			name:           "In development status",
			status:         "in development",
			expectedTitle:  "Development",
			expectedColour: "Yellow",
		},
		{
			name:           "WIP status",
			status:         "wip",
			expectedTitle:  "Development",
			expectedColour: "Yellow",
		},
		{
			name:           "Planning status",
			status:         "planning",
			expectedTitle:  "Planning",
			expectedColour: "Grey",
		},
		{
			name:           "Deprecated status",
			status:         "deprecated",
			expectedTitle:  "Deprecated",
			expectedColour: "Red",
		},
		{
			name:           "Retiring status",
			status:         "retiring",
			expectedTitle:  "Deprecated",
			expectedColour: "Red",
		},
		{
			name:           "Paused status",
			status:         "paused",
			expectedTitle:  "Paused",
			expectedColour: "Yellow",
		},
		{
			name:           "Unknown status",
			status:         "custom-status",
			expectedTitle:  "Custom-status",
			expectedColour: "Grey",
		},
		{
			name:           "Empty status",
			status:         "",
			expectedTitle:  "Unknown",
			expectedColour: "Grey",
		},
		{
			name:           "Case insensitive",
			status:         "LIVE",
			expectedTitle:  "Live",
			expectedColour: "Green",
		},
		{
			name:           "Status with whitespace",
			status:         "  live  ",
			expectedTitle:  "Live",
			expectedColour: "Green",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			badge := GetStatusBadge(tt.status)
			assert.Equal(t, tt.expectedTitle, badge.Title)
			assert.Equal(t, tt.expectedColour, badge.Colour)
		})
	}
}

func TestGeneratePageContent(t *testing.T) {
	launchDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	asset := &domain.Asset{
		ID:          "cap-asset-test-asset",
		Name:        "Test Asset",
		Description: "A test asset for unit testing",
		Why:         "We need this to test our code",
		Benefits:    "Improved code quality",
		How:         "By running automated tests",
		Metrics:     "Code coverage percentage",
		Platform:    "Testing Platform",
		Status:      "live",
		LaunchDate:  launchDate,
		OwningTeam:  "Platform Team",
	}

	content := GeneratePageContent(asset)

	// Verify main structure
	t.Run("Contains main title", func(t *testing.T) {
		assert.Contains(t, content, `<h1>Asset Capitalisation</h1>`)
	})

	t.Run("Contains section headers", func(t *testing.T) {
		assert.Contains(t, content, `<h2>Overview</h2>`)
		assert.Contains(t, content, `<h2>Value</h2>`)
		assert.Contains(t, content, `<h2>Asset Checklist</h2>`)
	})

	// Verify overview section elements
	t.Run("Contains overview table with grey headers", func(t *testing.T) {
		assert.Contains(t, content, `background-color: #f4f5f7`)
	})

	t.Run("Contains asset name", func(t *testing.T) {
		assert.Contains(t, content, "Test Asset")
		assert.Contains(t, content, `<strong>Asset</strong>`)
	})

	t.Run("Contains owning team", func(t *testing.T) {
		assert.Contains(t, content, "Platform Team")
		assert.Contains(t, content, `<strong>Asset owned by</strong>`)
	})

	t.Run("Contains platform as pod", func(t *testing.T) {
		assert.Contains(t, content, "Testing Platform")
		assert.Contains(t, content, `<strong>Pod</strong>`)
	})

	t.Run("Contains launch date in human format", func(t *testing.T) {
		assert.Contains(t, content, `Mar 15, 2024`)
		assert.Contains(t, content, `<strong>Launch date</strong>`)
	})

	t.Run("Contains status badge", func(t *testing.T) {
		assert.Contains(t, content, `ac:name="status"`)
		assert.Contains(t, content, `<ac:parameter ac:name="title">Live</ac:parameter>`)
		assert.Contains(t, content, `<ac:parameter ac:name="colour">Green</ac:parameter>`)
	})

	// Verify value section elements
	t.Run("Contains value table with green headers", func(t *testing.T) {
		assert.Contains(t, content, `background-color: #e3fcef`)
	})

	t.Run("Contains why field", func(t *testing.T) {
		assert.Contains(t, content, "We need this to test our code")
		assert.Contains(t, content, `<em>Why are we doing this?</em>`)
	})

	t.Run("Contains benefits field", func(t *testing.T) {
		assert.Contains(t, content, "Improved code quality")
		assert.Contains(t, content, `<em>Economic benefits</em>`)
	})

	t.Run("Contains how field", func(t *testing.T) {
		assert.Contains(t, content, "By running automated tests")
		assert.Contains(t, content, `<em>How it works?</em>`)
	})

	t.Run("Contains metrics field", func(t *testing.T) {
		assert.Contains(t, content, "Code coverage percentage")
		assert.Contains(t, content, `<em>How do we judge success?</em>`)
	})

	// Verify checklist section
	t.Run("Contains asset checklist", func(t *testing.T) {
		assert.Contains(t, content, `<ac:task-list>`)
		assert.Contains(t, content, `<ac:task-status>complete</ac:task-status>`)
		assert.Contains(t, content, "technically feasible")
	})
}

func TestGeneratePageContent_WithEmptyFields(t *testing.T) {
	asset := &domain.Asset{
		ID:   "cap-asset-empty",
		Name: "Empty Asset",
	}

	content := GeneratePageContent(asset)

	t.Run("Uses dash for empty owning team", func(t *testing.T) {
		assert.Contains(t, content, "<strong>Asset owned by</strong></p></th><td><p>-</p></td>")
	})

	t.Run("Uses dash for empty platform", func(t *testing.T) {
		assert.Contains(t, content, "<strong>Pod</strong></p></th><td><p>-</p></td>")
	})

	t.Run("Uses dash for empty launch date", func(t *testing.T) {
		assert.Contains(t, content, "<strong>Launch date</strong></p></th><td><p>-</p></td>")
	})

	t.Run("Uses unknown for empty status", func(t *testing.T) {
		assert.Contains(t, content, `<ac:parameter ac:name="title">Unknown</ac:parameter>`)
	})

	t.Run("Uses dash for empty why", func(t *testing.T) {
		assert.Contains(t, content, "<em>Why are we doing this?</em></p></th><td><p>-</p></td>")
	})
}

func TestGeneratePageContent_HTMLEscaping(t *testing.T) {
	asset := &domain.Asset{
		ID:   "cap-asset-html",
		Name: "Asset with <script>alert('xss')</script>",
		Why:  "We need this & that",
	}

	content := GeneratePageContent(asset)

	t.Run("Escapes HTML in name", func(t *testing.T) {
		assert.Contains(t, content, "&lt;script&gt;")
		assert.NotContains(t, content, "<script>alert")
	})

	t.Run("Escapes ampersand in content", func(t *testing.T) {
		assert.Contains(t, content, "&amp;")
	})
}

func TestFormatMultilineContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single line",
			input:    "Single line content",
			expected: "<p>Single line content</p>",
		},
		{
			name:     "Multiple lines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "<p>Line 1</p><p>Line 2</p><p>Line 3</p>",
		},
		{
			name:     "Empty lines filtered",
			input:    "Line 1\n\nLine 2",
			expected: "<p>Line 1</p><p>Line 2</p>",
		},
		{
			name:     "Dash placeholder",
			input:    "-",
			expected: "<p>-</p>",
		},
		{
			name:     "Empty content",
			input:    "",
			expected: "<p>-</p>",
		},
		{
			name:     "Only whitespace",
			input:    "   \n   \n   ",
			expected: "<p>-</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMultilineContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateStatusMacro(t *testing.T) {
	badge := StatusBadge{Title: "Live", Colour: "Green"}
	result := generateStatusMacro(badge)

	assert.Contains(t, result, `ac:name="status"`)
	assert.Contains(t, result, `<ac:parameter ac:name="title">Live</ac:parameter>`)
	assert.Contains(t, result, `<ac:parameter ac:name="colour">Green</ac:parameter>`)
}

func TestGenerateDateMacro(t *testing.T) {
	result := generateDateMacro("2024-03-15")
	assert.Equal(t, `<time datetime="2024-03-15" />`, result)
}

func TestGenerateTableRow(t *testing.T) {
	result := generateTableRow("Header", "Value")
	expected := `<tr><th><p><strong>Header</strong></p></th><td>Value</td></tr>`
	assert.Equal(t, expected, result)
}

func TestPageContentStructure(t *testing.T) {
	asset := &domain.Asset{
		ID:         "cap-asset-structure-test",
		Name:       "Structure Test",
		Platform:   "Test Pod",
		Status:     "live",
		LaunchDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OwningTeam: "Test Team",
		Why:        "Why content",
		Benefits:   "Benefits content",
		Metrics:    "Metrics content",
		How:        "How content",
	}

	content := GeneratePageContent(asset)

	t.Run("Has three sections with headers", func(t *testing.T) {
		assert.Contains(t, content, `<h1>Asset Capitalisation</h1>`, "Should have main H1 title")
		assert.Contains(t, content, `<h2>Overview</h2>`, "Should have Overview H2 header")
		assert.Contains(t, content, `<h2>Value</h2>`, "Should have Value H2 header")
		assert.Contains(t, content, `<h2>Asset Checklist</h2>`, "Should have Asset Checklist H2 header")
	})

	t.Run("Has two tables", func(t *testing.T) {
		count := strings.Count(content, `<table`)
		assert.Equal(t, 2, count, "Should have exactly 2 tables (overview and value)")
	})

	t.Run("Overview section comes before value section", func(t *testing.T) {
		overviewPos := strings.Index(content, `#f4f5f7`)
		valuePos := strings.Index(content, `#e3fcef`)
		assert.Less(t, overviewPos, valuePos, "Overview section should come before value section")
	})

	t.Run("Has checklist section with task list", func(t *testing.T) {
		assert.Contains(t, content, `<ac:task-list>`, "Should have task list")
		taskCount := strings.Count(content, `<ac:task>`)
		assert.Equal(t, 5, taskCount, "Should have exactly 5 checklist items")
	})
}

func TestGeneratePageContentWithTribe(t *testing.T) {
	asset := &domain.Asset{
		ID:         "cap-asset-tribe-test",
		Name:       "Tribe Test Asset",
		OwningTeam: "COP",
		Platform:   "Test Platform",
		Status:     "live",
		LaunchDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	t.Run("includes tribe when provided", func(t *testing.T) {
		content := GeneratePageContentWithTribe(asset, "Platform Tribe")

		assert.Contains(t, content, "Platform Tribe")
		assert.Contains(t, content, `<strong>Tribe</strong>`)
		// Should not contain dash for tribe
		assert.NotContains(t, content, `<strong>Tribe</strong></p></th><td><p>-</p></td>`)
	})

	t.Run("shows dash when tribe is empty", func(t *testing.T) {
		content := GeneratePageContentWithTribe(asset, "")

		assert.Contains(t, content, `<strong>Tribe</strong></p></th><td><p>-</p></td>`)
	})

	t.Run("GeneratePageContent uses empty tribe by default", func(t *testing.T) {
		content := GeneratePageContent(asset)

		// Should show dash for tribe since no tribe is passed
		assert.Contains(t, content, `<strong>Tribe</strong></p></th><td><p>-</p></td>`)
	})

	t.Run("escapes HTML in tribe name", func(t *testing.T) {
		content := GeneratePageContentWithTribe(asset, "Tribe <script>alert('xss')</script>")

		assert.Contains(t, content, "&lt;script&gt;")
		assert.NotContains(t, content, "<script>alert")
	})
}

func TestFormatKeywords(t *testing.T) {
	tests := []struct {
		name     string
		keywords []string
		expected string
	}{
		{
			name:     "empty keywords",
			keywords: []string{},
			expected: "<p>-</p>",
		},
		{
			name:     "nil keywords",
			keywords: nil,
			expected: "<p>-</p>",
		},
		{
			name:     "single keyword",
			keywords: []string{"payment"},
			expected: "<p>payment</p>",
		},
		{
			name:     "multiple keywords",
			keywords: []string{"payment", "checkout", "mobile"},
			expected: "<p>payment, checkout, mobile</p>",
		},
		{
			name:     "keywords with special characters",
			keywords: []string{"test & keyword", "another <keyword>"},
			expected: "<p>test &amp; keyword, another &lt;keyword&gt;</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatKeywords(tt.keywords)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePageContent_ContainsKeywords(t *testing.T) {
	asset := &domain.Asset{
		ID:       "cap-asset-keywords-test",
		Name:     "Keywords Test Asset",
		Keywords: []string{"payment", "checkout", "mobile"},
	}

	content := GeneratePageContent(asset)

	t.Run("Contains keywords row in value section", func(t *testing.T) {
		assert.Contains(t, content, `<em>Keywords</em>`)
	})

	t.Run("Contains keywords as comma-separated values", func(t *testing.T) {
		assert.Contains(t, content, "payment, checkout, mobile")
	})

	t.Run("Keywords row has green background header", func(t *testing.T) {
		// Verify keywords row is in the value table (green headers)
		assert.Contains(t, content, `background-color: #e3fcef`)
	})
}

func TestGeneratePageContent_EmptyKeywords(t *testing.T) {
	asset := &domain.Asset{
		ID:       "cap-asset-no-keywords",
		Name:     "No Keywords Asset",
		Keywords: nil,
	}

	content := GeneratePageContent(asset)

	t.Run("Shows dash for empty keywords", func(t *testing.T) {
		assert.Contains(t, content, `<em>Keywords</em></p></th><td><p>-</p></td>`)
	})
}

func TestGeneratePageContentFull(t *testing.T) {
	asset := &domain.Asset{
		ID:         "cap-asset-full-test",
		Name:       "Full Test Asset",
		OwningTeam: "COP",
		Platform:   "Legacy Platform",
		Status:     "live",
		LaunchDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	t.Run("shows company as Asset owned by", func(t *testing.T) {
		content := GeneratePageContentFull(asset, "Engineering", "TechCorp", "COP Team")

		// Company should appear as "Asset owned by"
		assert.Contains(t, content, `<strong>Asset owned by</strong></p></th><td><p>TechCorp</p></td>`)
	})

	t.Run("shows team name as Pod", func(t *testing.T) {
		content := GeneratePageContentFull(asset, "Engineering", "TechCorp", "COP Team")

		// Team name should appear as "Pod"
		assert.Contains(t, content, `<strong>Pod</strong></p></th><td><p>COP Team</p></td>`)
	})

	t.Run("shows tribe correctly", func(t *testing.T) {
		content := GeneratePageContentFull(asset, "Engineering", "TechCorp", "COP Team")

		assert.Contains(t, content, `<strong>Tribe</strong></p></th><td><p>Engineering</p></td>`)
	})

	t.Run("falls back to owning team when company is empty", func(t *testing.T) {
		content := GeneratePageContentFull(asset, "Engineering", "", "COP Team")

		// Should show owning team (COP) as "Asset owned by"
		assert.Contains(t, content, `<strong>Asset owned by</strong></p></th><td><p>COP</p></td>`)
	})

	t.Run("falls back to platform when team name is empty", func(t *testing.T) {
		content := GeneratePageContentFull(asset, "Engineering", "TechCorp", "")

		// Should show platform as "Pod"
		assert.Contains(t, content, `<strong>Pod</strong></p></th><td><p>Legacy Platform</p></td>`)
	})

	t.Run("shows dash when all fallbacks are empty", func(t *testing.T) {
		emptyAsset := &domain.Asset{
			ID:   "cap-asset-empty",
			Name: "Empty Asset",
		}
		content := GeneratePageContentFull(emptyAsset, "", "", "")

		// Should show dash for both
		assert.Contains(t, content, `<strong>Asset owned by</strong></p></th><td><p>-</p></td>`)
		assert.Contains(t, content, `<strong>Pod</strong></p></th><td><p>-</p></td>`)
	})

	t.Run("escapes HTML in company and team name", func(t *testing.T) {
		content := GeneratePageContentFull(asset, "Tribe", "Company <script>", "Team <script>")

		assert.Contains(t, content, "&lt;script&gt;")
		assert.NotContains(t, content, "<script>")
	})
}
