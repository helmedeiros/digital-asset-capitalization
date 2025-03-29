package confluence

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dateFormats = []string{
	"since 2006",
	"January 2, 2006",
	"January 2nd, 2006",
	"January 2rd, 2006",
	"January 2th, 2006",
	"January 2st, 2006",
	"2006-01-02",
	"02/01/2006",
	"May 2, 2006",
	"March 4, 2006",
	"Q1 2006",
	"Q2 2006",
	"Q3 2006",
	"Q4 2006",
}

// PageMetadata represents the metadata extracted from a Confluence page
type PageMetadata struct {
	Description    string
	Why            string
	Benefits       string
	How            string
	Metrics        string
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
	metadata.Why = cleanHTML(extractTableValue(content, "Why are we doing this?"))
	metadata.Description = metadata.Why

	// Extract benefits from "Economic benefits" section
	metadata.Benefits = cleanHTML(extractTableValue(content, "Economic benefits"))

	// If Why is empty, use Benefits as Description
	if metadata.Description == "" {
		metadata.Description = metadata.Benefits
	}

	// Extract how from "How it works?" section
	metadata.How = cleanHTML(extractTableValue(content, "How it works?"))

	// Extract metrics from "How do we judge success?" section
	metadata.Metrics = cleanHTML(extractTableValue(content, "How do we judge success?"))

	// Extract platform from "Pod" section
	metadata.Platform = cleanHTML(extractTableValue(content, "Pod"))

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

	// Extract launch date
	launchDate := extractTableValue(content, "Launch date")

	var parsedDate time.Time
	if t, err := parseDate(launchDate); err == nil {
		parsedDate = t
	}

	metadata.LaunchDate = parsedDate

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
		header, // Simple text match
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

	// Find the start of the row containing our header
	rowStart := strings.LastIndex(content[:start], "<tr")
	if rowStart == -1 {
		return ""
	}

	// Find the end of the row
	rowEnd := strings.Index(content[start:], "</tr>")
	if rowEnd == -1 {
		return ""
	}
	rowEnd += start

	// Get the row content
	row := content[rowStart:rowEnd]

	// Check for proper closing tags
	tdCount := strings.Count(row, "<td")
	tdCloseCount := strings.Count(row, "</td")
	thCount := strings.Count(row, "<th")
	thCloseCount := strings.Count(row, "</th")

	// If the number of opening and closing tags don't match, the table is malformed
	if tdCount != tdCloseCount || thCount != thCloseCount {
		return ""
	}

	// Find the first cell in this row (could be td or th)
	firstCell := -1
	firstTd := strings.Index(row, "<td")
	firstTh := strings.Index(row, "<th")

	// Determine which tag appears first (if both exist)
	if firstTd != -1 && firstTh != -1 {
		if firstTd < firstTh {
			firstCell = firstTd
		} else {
			firstCell = firstTh
		}
	} else if firstTd != -1 {
		firstCell = firstTd
	} else if firstTh != -1 {
		firstCell = firstTh
	}

	if firstCell == -1 {
		return ""
	}

	// Find the second cell
	secondCell := -1
	// Look for both td and th after the first cell
	secondTd := strings.Index(row[firstCell+3:], "<td")
	secondTh := strings.Index(row[firstCell+3:], "<th")

	// Determine which tag appears first (if both exist)
	if secondTd != -1 && secondTh != -1 {
		if secondTd < secondTh {
			secondCell = secondTd
		} else {
			secondCell = secondTh
		}
	} else if secondTd != -1 {
		secondCell = secondTd
	} else if secondTh != -1 {
		secondCell = secondTh
	}

	if secondCell == -1 {
		return ""
	}
	secondCell += firstCell + 3

	// Find the end of the second cell (could be td or th)
	valueEnd := -1
	endTd := strings.Index(row[secondCell:], "</td>")
	endTh := strings.Index(row[secondCell:], "</th>")

	// Determine which end tag appears first (if both exist)
	if endTd != -1 && endTh != -1 {
		if endTd < endTh {
			valueEnd = endTd
		} else {
			valueEnd = endTh
		}
	} else if endTd != -1 {
		valueEnd = endTd
	} else if endTh != -1 {
		valueEnd = endTh
	}

	if valueEnd == -1 {
		return ""
	}
	valueEnd += secondCell

	// Find the actual content start (after the opening tag)
	valueStart := strings.Index(row[secondCell:valueEnd], ">")
	if valueStart == -1 {
		return ""
	}
	valueStart += secondCell + 1

	return strings.TrimSpace(row[valueStart:valueEnd])
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

func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func parseDate(dateStr string) (time.Time, error) {
	// Clean HTML from the date string first
	dateStr = cleanHTML(dateStr)

	// Remove any HTML time tag and extract datetime attribute if present
	if strings.Contains(dateStr, "<time") {
		re := regexp.MustCompile(`datetime="([^"]+)"`)
		if matches := re.FindStringSubmatch(dateStr); len(matches) > 1 {
			dateStr = matches[1]
		}
	}

	dateStr = strings.TrimSpace(dateStr)

	// Handle "since YYYY" format case-insensitively
	if strings.HasPrefix(strings.ToLower(dateStr), "since") {
		yearStr := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(dateStr), "since"))
		if year, err := strconv.Atoi(yearStr); err == nil {
			return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), nil
		}
	}

	// Handle quarter format (e.g., "Q1 2024")
	if strings.HasPrefix(dateStr, "Q") && len(dateStr) == 7 {
		quarter := mustParseInt(string(dateStr[1]))
		yearStr := dateStr[3:]
		if year, err := strconv.Atoi(yearStr); err == nil && quarter >= 1 && quarter <= 4 {
			month := time.Month((quarter-1)*3 + 1)
			return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC), nil
		}
	}

	// Remove ordinal indicators before parsing
	dateStr = regexp.MustCompile(`(\d+)(st|nd|rd|th)`).ReplaceAllString(dateStr, "$1")

	// Try all date formats
	for _, format := range dateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", dateStr)
}
