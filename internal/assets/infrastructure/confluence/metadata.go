package confluence

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PageMetadata represents the metadata extracted from a Confluence page
type PageMetadata struct {
	Description    string
	Platform       string
	Status         string
	LaunchDate     time.Time
	IsRolledOut100 bool
	Keywords       []string
	Identifier     string
}

// extractMetadata extracts metadata from the page content
func (a *Adapter) extractMetadata(content string) (*PageMetadata, error) {
	// Validate content structure
	if !strings.Contains(content, "<table") {
		return nil, fmt.Errorf("invalid content: no table found")
	}

	metadata := &PageMetadata{}

	// Extract description from "Why are we doing this?" section
	desc := extractTableValue(content, "Why are we doing this?")
	if desc != "" {
		metadata.Description = cleanHTML(desc)
	} else {
		// Fallback to "Economic benefits" section if "Why are we doing this?" is empty
		desc = extractTableValue(content, "Economic benefits")
		metadata.Description = cleanHTML(desc)
	}

	// Extract platform from "Pod" section
	platform := extractTableValue(content, "Pod")
	metadata.Platform = cleanHTML(platform)

	// Extract status from "Status" section
	status := extractTableValue(content, "Status")
	// Extract status from macro title
	if strings.Contains(status, "ac:parameter ac:name=\"title\"") {
		start := strings.Index(status, "ac:parameter ac:name=\"title\">") + len("ac:parameter ac:name=\"title\">")
		end := strings.Index(status[start:], "</ac:parameter>")
		if end != -1 {
			status = status[start : start+end]
		}
	}
	// Clean up status and correct common typos
	status = cleanHTML(status)
	status = strings.ReplaceAll(status, "continious", "continuous")
	metadata.Status = status

	// Extract launch date from "Launch date" section
	launchDate := extractTableValue(content, "Launch date")
	launchDate = cleanHTML(launchDate)

	// Try different date formats
	if strings.HasPrefix(strings.ToLower(launchDate), "since") {
		// Handle "since YYYY" format
		year := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(launchDate), "since"))
		if yearInt, err := strconv.Atoi(year); err == nil {
			metadata.LaunchDate = time.Date(yearInt, 1, 1, 0, 0, 0, 0, time.UTC)
		}
	} else {
		// Try parsing specific date formats
		formats := []string{
			"January 2, 2006",
			"January 2 2006",
			"Jan 2, 2006",
			"Jan 2 2006",
			"02/01/2006",
			"2006-01-02",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, launchDate); err == nil {
				metadata.LaunchDate = t
				break
			}
		}
	}

	// Extract keywords from labels
	metadata.Keywords = extractLabels(content)

	// Extract asset identifier from labels
	metadata.Identifier = extractAssetIdentifier(content)

	// Set rollout status based on content
	metadata.IsRolledOut100 = strings.Contains(content, "100% of traffic")

	return metadata, nil
}

// extractTableValue extracts a value from a table row with the given header
func extractTableValue(content, header string) string {
	// Look for the header in a table cell
	markers := []string{
		fmt.Sprintf("<strong>%s</strong>", header),
		fmt.Sprintf("<p><strong>%s</strong></p>", header),
		fmt.Sprintf("data-highlight-colour=\"#f4f5f7\"><p><strong>%s</strong>", header),
		fmt.Sprintf("data-highlight-colour=\"#e3fcef\"><p><strong>%s</strong>", header),
	}

	var start int = -1
	for _, marker := range markers {
		start = strings.Index(content, marker)
		if start != -1 {
			break
		}
	}
	if start == -1 {
		return ""
	}

	// Find the next table cell
	tdMarkers := []string{"<td><p>", "<td>"}
	tdStart := -1
	for _, marker := range tdMarkers {
		tdStart = strings.Index(content[start:], marker)
		if tdStart != -1 {
			tdStart += start + len(marker)
			break
		}
	}
	if tdStart == -1 {
		return ""
	}

	// Find the end of the table cell
	tdEnds := []string{"</p></td>", "</td>"}
	tdEnd := -1
	for _, marker := range tdEnds {
		tdEnd = strings.Index(content[tdStart:], marker)
		if tdEnd != -1 {
			tdEnd += tdStart
			break
		}
	}
	if tdEnd == -1 {
		return ""
	}

	return strings.TrimSpace(content[tdStart:tdEnd])
}

// cleanHTML removes HTML tags and decodes entities
func cleanHTML(text string) string {
	// Remove HTML tags
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", " ")
	text = strings.ReplaceAll(text, "<em>", "")
	text = strings.ReplaceAll(text, "</em>", "")
	text = strings.ReplaceAll(text, "<strong>", "")
	text = strings.ReplaceAll(text, "</strong>", "")
	text = strings.ReplaceAll(text, "<br />", " ")
	text = strings.ReplaceAll(text, "<br/>", " ")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")

	// Clean up whitespace
	text = strings.Join(strings.Fields(text), " ")
	return text
}

// extractLabels extracts labels from the metadata section
func extractLabels(content string) []string {
	// First try to find labels in the metadata.labels.results format
	labelPattern := `"name":"([^"]+)"`
	re := regexp.MustCompile(labelPattern)
	matches := re.FindAllStringSubmatch(content, -1)

	// Extract unique labels, excluding those prefixed with "global" or "cap-asset-"
	labels := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			label := match[1]
			if !strings.HasPrefix(label, "global") && !strings.HasPrefix(label, "cap-asset-") {
				labels[label] = true
			}
		}
	}

	// If no labels found, try the HTML format
	if len(labels) == 0 {
		labelPattern = `"label":"([^"]+)"`
		re = regexp.MustCompile(labelPattern)
		matches = re.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				label := match[1]
				if !strings.HasPrefix(label, "global") && !strings.HasPrefix(label, "cap-asset-") {
					labels[label] = true
				}
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(labels))
	for label := range labels {
		result = append(result, label)
	}

	return result
}

// extractAssetIdentifier extracts the asset identifier from labels
func extractAssetIdentifier(content string) string {
	// First try to find labels in the metadata.labels.results format
	labelPattern := `"name":"([^"]+)"`
	re := regexp.MustCompile(labelPattern)
	matches := re.FindAllStringSubmatch(content, -1)

	// Look for a label that matches the cap-asset-* pattern
	for _, match := range matches {
		if len(match) > 1 {
			label := match[1]
			if strings.HasPrefix(label, "cap-asset-") {
				return label
			}
		}
	}

	// If not found, try the HTML format
	labelPattern = `"label":"([^"]+)"`
	re = regexp.MustCompile(labelPattern)
	matches = re.FindAllStringSubmatch(content, -1)

	// Look for a label that matches the cap-asset-* pattern
	for _, match := range matches {
		if len(match) > 1 {
			label := match[1]
			if strings.HasPrefix(label, "cap-asset-") {
				return label
			}
		}
	}

	return ""
}
