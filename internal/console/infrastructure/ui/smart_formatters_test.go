package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSmartFormatter(t *testing.T) {
	sf := NewSmartFormatter()

	assert.NotNil(t, sf)
	assert.NotEmpty(t, sf.dateFormats)
	assert.NotNil(t, sf.numberRegex)
	assert.NotNil(t, sf.urlRegex)
	assert.NotNil(t, sf.emailRegex)
	assert.NotNil(t, sf.jiraKeyRegex)
	assert.NotNil(t, sf.moneyRegex)
	assert.Len(t, sf.dateFormats, 8)
}

func TestSmartFormatter_Format_NilAndEmpty(t *testing.T) {
	sf := NewSmartFormatter()

	// Test nil value
	result := sf.Format(nil)
	assert.Contains(t, result, "(null)")

	// Test empty string
	result = sf.Format("")
	assert.Contains(t, result, "(empty)")
}

func TestSmartFormatter_Format_WithContext(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		value    interface{}
		context  string
		expected string
	}{
		{"date context", "2023-12-25", "date", ""}, // Should format as date
		{"url context", "https://example.com", "url", ""},
		{"email context", "test@example.com", "email", ""},
		{"money context", "100.50", "money", ""},
		{"percent context", "85.5", "percent", ""},
		{"status context", "active", "status", ""},
		{"priority context", "high", "priority", ""},
		{"jira context", "ABC-123", "jira", ""},
		{"team context", "DevTeam", "team", ""},
		{"file context", "/path/to/file.go", "file", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.Format(tt.value, tt.context)
			// Just test that it doesn't panic and returns non-empty string
			assert.NotEmpty(t, result)
		})
	}
}

func TestSmartFormatter_AutoFormat(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		contains string // Check if result contains this string
	}{
		{"URL detection", "https://github.com/user/repo", "github.com"},
		{"Email detection", "user@domain.com", "user@domain.com"},
		{"JIRA key detection", "PROJ-123", "PROJ-123"},
		{"Money detection", "$100.50", "100.50"},
		{"Boolean true", "true", "true"},
		{"Boolean false", "false", "false"},
		{"Status active", "active", "true"},
		{"File path", "/path/to/file.txt", "/path/to/file.txt"},
		{"Number", "12345", "12345"},
		{"Percentage", "85%", "85"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.autoFormat(tt.input)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestSmartFormatter_FormatAsDate(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected bool // true if should format as date
	}{
		{"ISO date", "2023-12-25 10:30:00", true},
		{"ISO with timezone", "2023-12-25T10:30:00Z", true},
		{"Simple date", "2023-12-25", true},
		{"US date", "12/25/2023", true},
		{"Invalid date", "not-a-date", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatAsDate(tt.input)
			if tt.expected {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestSmartFormatter_FormatAsURL(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []string{
		"https://example.com",
		"http://test.org",
		"https://github.com/user/repo",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			result := sf.formatAsURL(url)
			assert.Contains(t, result, url)
		})
	}
}

func TestSmartFormatter_FormatAsEmail(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []string{
		"user@example.com",
		"test.email@domain.org",
		"name+tag@company.co.uk",
	}

	for _, email := range tests {
		t.Run(email, func(t *testing.T) {
			result := sf.formatAsEmail(email)
			assert.Contains(t, result, email)
		})
	}
}

func TestSmartFormatter_FormatAsMoney(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Dollar amount", "$100.50", "$100.50"},
		{"Euro amount", "€50.25", "€50.25"},
		{"Pound amount", "£75.00", "£75.00"},
		{"USD currency", "100 USD", "$100.00"},
		{"EUR currency", "50 EUR", "€50.00"},
		{"GBP currency", "75 GBP", "£75.00"},
		{"Plain number", "25.5", "$25.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatAsMoney(tt.input)
			assert.NotEmpty(t, result)
		})
	}
}

func TestSmartFormatter_FormatAsPercentage(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"High percentage", "85%", "85.0%"},
		{"Medium percentage", "65%", "65.0%"},
		{"Low percentage", "35%", "35.0%"},
		{"Very low percentage", "15%", "15.0%"},
		{"Perfect score", "100%", "100.0%"},
		{"Decimal percentage", "87.5%", "87.5%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatAsPercentage(tt.input)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestSmartFormatter_FormatAsBoolean(t *testing.T) {
	sf := NewSmartFormatter()

	trueValues := []string{"true", "yes", "y", "1", "on", "enabled", "active"}
	falseValues := []string{"false", "no", "n", "0", "off", "disabled", "inactive"}
	invalidValues := []string{"maybe", "unknown", ""}

	for _, val := range trueValues {
		t.Run("true_"+val, func(t *testing.T) {
			result := sf.formatAsBoolean(val)
			assert.Contains(t, result, "true")
		})
	}

	for _, val := range falseValues {
		t.Run("false_"+val, func(t *testing.T) {
			result := sf.formatAsBoolean(val)
			assert.Contains(t, result, "false")
		})
	}

	for _, val := range invalidValues {
		t.Run("invalid_"+val, func(t *testing.T) {
			result := sf.formatAsBoolean(val)
			assert.Empty(t, result)
		})
	}
}

func TestSmartFormatter_FormatAsStatus(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Active status", "active", "Active"},
		{"Inactive status", "inactive", "Inactive"},
		{"Completed status", "completed", "Completed"},
		{"Done status", "done", "Done"},
		{"Failed status", "failed", "Failed"},
		{"Pending status", "pending", "Pending"},
		{"In progress", "in progress", "In Progress"},
		{"Blocked status", "blocked", "Blocked"},
		{"Unknown status", "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatAsStatus(tt.input)
			if tt.expected != "" {
				assert.Contains(t, result, tt.expected)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestSmartFormatter_FormatAsPriority(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []string{
		"critical",
		"high",
		"medium",
		"low",
		"urgent",
		"normal",
		"minor",
	}

	for _, priority := range tests {
		t.Run(priority, func(t *testing.T) {
			result := sf.formatAsPriority(priority)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, priority)
		})
	}

	// Test unknown priority
	result := sf.formatAsPriority("unknown")
	assert.Contains(t, result, "unknown")
}

func TestSmartFormatter_FormatAsJiraKey(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name   string
		input  string
		isJira bool
	}{
		{"Valid JIRA key", "PROJ-123", true},
		{"Valid JIRA key 2", "ABC-456", true},
		{"Invalid format", "proj-123", false},
		{"No dash", "PROJ123", false},
		{"No letters", "123-456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatAsJiraKey(tt.input)
			assert.Contains(t, result, tt.input)
		})
	}
}

func TestSmartFormatter_FormatAsTeam(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal team", "DevTeam", "DevTeam"},
		{"Empty team", "", "(no team)"},
		{"None team", "none", "(no team)"},
		{"Unassigned team", "unassigned", "(no team)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatAsTeam(tt.input)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestSmartFormatter_FormatAsFilePath(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []string{
		"/path/to/file.go",
		"./relative/path.txt",
		"../parent/file.js",
		"C:\\Windows\\System32\\file.dll",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			result := sf.formatAsFilePath(path)
			assert.Contains(t, result, path)
		})
	}
}

func TestSmartFormatter_FormatAsNumber(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Small integer", "42", "42"},
		{"Large integer", "1000", "1000"},
		{"Very large", "1000000", "1000000"},
		{"Decimal", "3.14159", "3.14"},
		{"Negative", "-42", "-42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatAsNumber(tt.input)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestSmartFormatter_IsFilePath(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Absolute path", "/usr/bin/go", true},
		{"Relative path", "./config.json", true},
		{"Parent path", "../package.json", true},
		{"File with extension", "path/to/file.txt", true},
		{"Directory", "/home/user", true}, // Contains / character
		{"Not a path", "just text", false},
		{"URL", "https://example.com", true}, // Contains / and .
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.isFilePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSmartFormatter_FormatDuration(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{"Milliseconds", 500 * time.Millisecond, "500ms"},
		{"Seconds", 5 * time.Second, "5.0s"},
		{"Minutes", 5 * time.Minute, "5.0m"},
		{"Hours", 2 * time.Hour, "2.0h"},
		{"Days", 25 * time.Hour, "1.0d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.FormatDuration(tt.duration)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestSmartFormatter_FormatSize(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		bytes    int64
		contains string
	}{
		{"Bytes", 512, "512 B"},
		{"Kilobytes", 2048, "2.0 KB"},
		{"Megabytes", 5 * 1024 * 1024, "5.0 MB"},
		{"Gigabytes", 3 * 1024 * 1024 * 1024, "3.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.FormatSize(tt.bytes)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestSmartFormatter_FormatList(t *testing.T) {
	sf := NewSmartFormatter()

	items := []interface{}{"item1", "item2", "item3"}

	tests := []struct {
		name     string
		listType string
		contains string
	}{
		{"Numbered list", "numbered", "1."},
		{"Bullet list", "bullet", "•"},
		{"Comma list", "comma", ","},
		{"Default list", "", "•"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.FormatList(items, tt.listType)
			assert.Contains(t, result, tt.contains)
		})
	}

	// Test empty list
	result := sf.FormatList([]interface{}{}, "bullet")
	assert.Contains(t, result, "(empty)")
}

func TestSmartFormatter_FormatKeyValue(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		key      string
		value    interface{}
		contains string
	}{
		{"Simple key-value", "name", "TestAsset", "Name"},
		{"Snake case key", "created_at", "2023-12-25", "Created"},
		{"Status value", "status", "active", "Status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.FormatKeyValue(tt.key, tt.value)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestSmartFormatter_FormatKeyName(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Snake case", "created_at", "Created"},
		{"ID special case", "asset_id", "Asset Id"},
		{"URL special case", "doc_url", "Doc Url"},
		{"Simple word", "name", "Name"},
		{"Multiple words", "task_count", "Tasks"},
		{"Doc link", "doc_link", "Documentation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatKeyName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSmartFormatter_FormatWithContext(t *testing.T) {
	sf := NewSmartFormatter()

	tests := []struct {
		name    string
		str     string
		context string
		isEmpty bool
	}{
		{"Date context", "2023-12-25", "date", false},
		{"URL context", "https://test.com", "url", false},
		{"Email context", "test@example.com", "email", false},
		{"Money context", "100", "money", false},
		{"Percent context", "85", "percent", false},
		{"Status context", "active", "status", false},
		{"Priority context", "high", "priority", false},
		{"JIRA context", "ABC-123", "jira", false},
		{"Team context", "DevTeam", "team", false},
		{"File context", "/path/file.go", "file", false},
		{"Unknown context", "value", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sf.formatWithContext(tt.str, tt.context)
			if tt.isEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
			}
		})
	}
}
