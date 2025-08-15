package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Smart formatter status constants to avoid goconst warnings
const (
	SmartStatusDone = "done"
)

// SmartFormatter provides intelligent formatting based on data content and context
type SmartFormatter struct {
	dateFormats  []string
	numberRegex  *regexp.Regexp
	urlRegex     *regexp.Regexp
	emailRegex   *regexp.Regexp
	jiraKeyRegex *regexp.Regexp
	moneyRegex   *regexp.Regexp
}

// NewSmartFormatter creates a new smart formatter with common patterns
func NewSmartFormatter() *SmartFormatter {
	return &SmartFormatter{
		dateFormats: []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02",
			"01/02/2006",
			"02-01-2006",
			"Jan 2, 2006",
			"January 2, 2006",
		},
		numberRegex:  regexp.MustCompile(`^-?\d+\.?\d*$`),
		urlRegex:     regexp.MustCompile(`^https?://[^\s]+$`),
		emailRegex:   regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		jiraKeyRegex: regexp.MustCompile(`^[A-Z]+-\d+$`),
		moneyRegex:   regexp.MustCompile(`^\$?\d+\.?\d*\s*(USD|EUR|GBP|\$|€|£)?$`),
	}
}

// Format intelligently formats a value based on its content and optional context
func (sf *SmartFormatter) Format(value interface{}, context ...string) string {
	if value == nil {
		return MutedText("(null)")
	}

	str := fmt.Sprintf("%v", value)
	if str == "" {
		return MutedText("(empty)")
	}

	// Check context hints first
	if len(context) > 0 {
		if formatted := sf.formatWithContext(str, context[0]); formatted != "" {
			return formatted
		}
	}

	// Auto-detect format based on content
	return sf.autoFormat(str)
}

// formatWithContext formats based on explicit context information
func (sf *SmartFormatter) formatWithContext(str, context string) string {
	context = strings.ToLower(context)

	switch {
	case strings.Contains(context, "date") || strings.Contains(context, "time"):
		return sf.formatAsDate(str)
	case strings.Contains(context, "url") || strings.Contains(context, "link"):
		return sf.formatAsURL(str)
	case strings.Contains(context, "email"):
		return sf.formatAsEmail(str)
	case strings.Contains(context, "money") || strings.Contains(context, "cost") || strings.Contains(context, "price"):
		return sf.formatAsMoney(str)
	case strings.Contains(context, "percent"):
		return sf.formatAsPercentage(str)
	case strings.Contains(context, "status"):
		return sf.formatAsStatus(str)
	case strings.Contains(context, "priority"):
		return sf.formatAsPriority(str)
	case strings.Contains(context, "jira") || strings.Contains(context, "key"):
		return sf.formatAsJiraKey(str)
	case strings.Contains(context, "team"):
		return sf.formatAsTeam(str)
	case strings.Contains(context, "file") || strings.Contains(context, "path"):
		return sf.formatAsFilePath(str)
	}

	return ""
}

// autoFormat automatically detects and formats based on content patterns
func (sf *SmartFormatter) autoFormat(str string) string {
	// URL detection
	if sf.urlRegex.MatchString(str) {
		return sf.formatAsURL(str)
	}

	// Email detection
	if sf.emailRegex.MatchString(str) {
		return sf.formatAsEmail(str)
	}

	// JIRA key detection
	if sf.jiraKeyRegex.MatchString(str) {
		return sf.formatAsJiraKey(str)
	}

	// Money detection
	if sf.moneyRegex.MatchString(str) {
		return sf.formatAsMoney(str)
	}

	// Date detection
	if dateStr := sf.formatAsDate(str); dateStr != "" {
		return dateStr
	}

	// Boolean-like values
	if boolStr := sf.formatAsBoolean(str); boolStr != "" {
		return boolStr
	}

	// Status-like values
	if statusStr := sf.formatAsStatus(str); statusStr != "" {
		return statusStr
	}

	// File path detection
	if sf.isFilePath(str) {
		return sf.formatAsFilePath(str)
	}

	// Number formatting
	if sf.numberRegex.MatchString(str) {
		return sf.formatAsNumber(str)
	}

	// Percentage detection
	if strings.HasSuffix(str, "%") {
		return sf.formatAsPercentage(str)
	}

	// Default formatting
	return str
}

// formatAsDate tries to parse and format as a date
func (sf *SmartFormatter) formatAsDate(str string) string {
	for _, format := range sf.dateFormats {
		if t, err := time.Parse(format, str); err == nil {
			// Format based on recency
			now := time.Now()
			diff := now.Sub(t)

			if diff < 24*time.Hour {
				return MutedText(t.Format("15:04"))
			} else if diff < 7*24*time.Hour {
				return MutedText(t.Format("Mon 15:04"))
			} else if diff < 365*24*time.Hour {
				return MutedText(t.Format("Jan 2"))
			}
			return MutedText(t.Format("Jan 2, 2006"))
		}
	}
	return ""
}

// formatAsURL formats URLs with link styling
func (sf *SmartFormatter) formatAsURL(str string) string {
	return Link(str)
}

// formatAsEmail formats email addresses
func (sf *SmartFormatter) formatAsEmail(str string) string {
	return Link(str)
}

// formatAsMoney formats monetary values
func (sf *SmartFormatter) formatAsMoney(str string) string {
	// Extract number and currency
	cleaned := strings.ReplaceAll(str, ",", "")

	if f, err := strconv.ParseFloat(strings.Trim(cleaned, "$€£USDECHRBGP "), 64); err == nil {
		// Determine currency
		currency := "$"
		if strings.Contains(str, "EUR") || strings.Contains(str, "€") {
			currency = "€"
		} else if strings.Contains(str, "GBP") || strings.Contains(str, "£") {
			currency = "£"
		}

		formatted := fmt.Sprintf("%s%.2f", currency, f)
		return Code(BoldText(formatted))
	}

	return Code(str)
}

// formatAsPercentage formats percentage values
func (sf *SmartFormatter) formatAsPercentage(str string) string {
	cleaned := strings.TrimSuffix(str, "%")
	if f, err := strconv.ParseFloat(cleaned, 64); err == nil {
		color := ColorInfo
		if f >= 80 {
			color = ColorSuccess
		} else if f < 50 {
			color = ColorError
		} else if f < 70 {
			color = ColorWarning
		}

		return Colorize(fmt.Sprintf("%.1f%%", f), color)
	}

	return Code(str)
}

// formatAsBoolean formats boolean-like values
func (sf *SmartFormatter) formatAsBoolean(str string) string {
	lower := strings.ToLower(str)

	switch lower {
	case "true", "yes", "y", "1", "on", "enabled", "active":
		return SuccessText("true")
	case "false", "no", "n", "0", "off", "disabled", "inactive":
		return ErrorText("false")
	}

	return ""
}

// formatAsStatus formats status values with appropriate colors and icons
func (sf *SmartFormatter) formatAsStatus(str string) string {
	lower := strings.ToLower(str)

	statusMap := map[string]struct {
		icon  string
		color Color
		text  string
	}{
		"active":        {"", ColorSuccess, "Active"},
		"inactive":      {"", ColorMuted, "Inactive"},
		"completed":     {"", ColorSuccess, "Completed"},
		SmartStatusDone: {"", ColorSuccess, "Done"},
		"failed":        {"", ColorError, "Failed"},
		"error":         {"", ColorError, "Error"},
		"pending":       {"", ColorWarning, "Pending"},
		"in progress":   {"", ColorInfo, "In Progress"},
		"blocked":       {"", ColorError, "Blocked"},
		"cancelled":     {"", ColorMuted, "Cancelled"},
		"draft":         {"", ColorMuted, "Draft"},
		"review":        {"", ColorWarning, "Review"},
		"approved":      {"", ColorSuccess, "Approved"},
		"rejected":      {"", ColorError, "Rejected"},
	}

	if status, exists := statusMap[lower]; exists {
		return Colorize(status.text, status.color)
	}

	// Check for partial matches
	for key, status := range statusMap {
		if strings.Contains(lower, key) {
			return Colorize(str, status.color)
		}
	}

	return ""
}

// formatAsPriority formats priority values
func (sf *SmartFormatter) formatAsPriority(str string) string {
	lower := strings.ToLower(str)

	priorityMap := map[string]Color{
		"critical": ColorError,
		"high":     ColorWarning,
		"medium":   ColorInfo,
		"low":      ColorMuted,
		"urgent":   ColorError,
		"normal":   ColorInfo,
		"minor":    ColorMuted,
	}

	if priority, exists := priorityMap[lower]; exists {
		return Colorize(str, priority)
	}

	return BoldText(str)
}

// formatAsJiraKey formats JIRA keys as clickable links
func (sf *SmartFormatter) formatAsJiraKey(str string) string {
	if sf.jiraKeyRegex.MatchString(str) {
		return Link(BoldText(str))
	}
	return BoldText(str)
}

// formatAsTeam formats team names
func (sf *SmartFormatter) formatAsTeam(str string) string {
	if str == "" || strings.ToLower(str) == "none" || strings.ToLower(str) == "unassigned" {
		return MutedText("(no team)")
	}
	return Colorize(str, ColorInfo)
}

// formatAsFilePath formats file paths
func (sf *SmartFormatter) formatAsFilePath(str string) string {
	return Code(str)
}

// formatAsNumber formats numeric values with appropriate styling
func (sf *SmartFormatter) formatAsNumber(str string) string {
	if f, err := strconv.ParseFloat(str, 64); err == nil {
		// Large numbers with thousands separator
		if f >= 1000 {
			return BoldText(fmt.Sprintf("%.0f", f))
		}
		// Decimal numbers
		if f != float64(int64(f)) {
			return fmt.Sprintf("%.2f", f)
		}
		// Integers
		return fmt.Sprintf("%.0f", f)
	}
	return str
}

// isFilePath checks if a string looks like a file path
func (sf *SmartFormatter) isFilePath(str string) bool {
	// Check for common file path patterns
	return strings.Contains(str, "/") &&
		(strings.Contains(str, ".") || strings.HasPrefix(str, "/") || strings.HasPrefix(str, "./") || strings.HasPrefix(str, "../"))
}

// FormatDuration formats time durations in a human-readable way
func (sf *SmartFormatter) FormatDuration(d time.Duration) string {
	if d < time.Second {
		return MutedText(fmt.Sprintf("%dms", d.Milliseconds()))
	} else if d < time.Minute {
		return MutedText(fmt.Sprintf("%.1fs", d.Seconds()))
	} else if d < time.Hour {
		return MutedText(fmt.Sprintf("%.1fm", d.Minutes()))
	} else if d < 24*time.Hour {
		return MutedText(fmt.Sprintf("%.1fh", d.Hours()))
	}
	return MutedText(fmt.Sprintf("%.1fd", d.Hours()/24))
}

// FormatSize formats byte sizes in human-readable units
func (sf *SmartFormatter) FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatList formats a list of items with appropriate styling
func (sf *SmartFormatter) FormatList(items []interface{}, listType string) string {
	if len(items) == 0 {
		return MutedText("(empty)")
	}

	var result strings.Builder

	for i, item := range items {
		itemStr := sf.Format(item)

		switch listType {
		case "numbered":
			result.WriteString(fmt.Sprintf("%s %s", InfoText(fmt.Sprintf("%d.", i+1)), itemStr))
		case "bullet":
			result.WriteString(fmt.Sprintf("%s %s", InfoText("•"), itemStr))
		case "comma":
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(itemStr)
		default: // default to bullet
			result.WriteString(fmt.Sprintf("%s %s", InfoText("•"), itemStr))
		}

		if listType != "comma" && i < len(items)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// FormatKeyValue formats a key-value pair with smart value formatting
func (sf *SmartFormatter) FormatKeyValue(key string, value interface{}) string {
	formattedKey := sf.formatKeyName(key)
	formattedValue := sf.Format(value, key) // Pass key as context

	return fmt.Sprintf("%s: %s", Colorize(formattedKey, ColorPrimary), formattedValue)
}

// formatKeyName formats a key name for display
func (sf *SmartFormatter) formatKeyName(key string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(key, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}

	result := strings.Join(parts, " ")

	// Special cases
	specialCases := map[string]string{
		"Id":              "ID",
		"Url":             "URL",
		"Api":             "API",
		"Ui":              "UI",
		"Uuid":            "UUID",
		"Json":            "JSON",
		"Xml":             "XML",
		"Html":            "HTML",
		"Css":             "CSS",
		"Js":              "JS",
		"Doc Link":        "Documentation",
		"Task Count":      "Tasks",
		"Created At":      "Created",
		"Updated At":      "Updated",
		"Last Doc Update": "Last Doc Update",
	}

	if special, exists := specialCases[result]; exists {
		return special
	}

	return result
}
