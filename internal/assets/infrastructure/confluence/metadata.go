package confluence

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
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
	metadata := &PageMetadata{}

	// Check for invalid content
	if !strings.Contains(content, "<table") || !strings.Contains(content, "</table>") {
		return nil, fmt.Errorf("invalid content: no table found")
	}

	// Extract labels and identifier
	metadata.Keywords = extractLabels(content)
	metadata.Identifier = extractAssetIdentifier(content)

	// Extract table values
	metadata.Why = cleanHTML(extractTableValue(content, "Why are we doing this?"))
	metadata.Benefits = cleanHTML(extractTableValue(content, "Economic benefits"))
	metadata.How = cleanHTML(extractTableValue(content, "How it works?"))
	metadata.Metrics = cleanHTML(extractTableValue(content, "How do we judge success?"))
	metadata.Platform = cleanHTML(extractTableValue(content, "Pod"))
	metadata.Status = cleanHTML(extractTableValue(content, "Status"))

	// Extract launch date
	launchDate := extractTableValue(content, "Launch date")
	var parsedDate time.Time
	if t, err := parseDate(launchDate); err == nil {
		parsedDate = t
	}

	metadata.LaunchDate = parsedDate

	// Set description to Why field if available, otherwise use Economic benefits
	if metadata.Why != "" {
		metadata.Description = metadata.Why
	} else {
		metadata.Description = metadata.Benefits
	}

	// Set rollout status based on content
	metadata.IsRolledOut100 = strings.Contains(content, "100% of traffic")

	return metadata, nil
}

// extractTableValue extracts a value from a table row with the given header
func extractTableValue(content string, header string) string {
	// Replace Unicode entities with their HTML equivalents
	content = strings.ReplaceAll(content, `\u003c`, "<")
	content = strings.ReplaceAll(content, `\u003e`, ">")

	// Check for malformed tables
	if !strings.Contains(content, "<table") || !strings.Contains(content, "</table>") {
		return ""
	}

	// Check for proper table structure
	tdCount := strings.Count(content, "<td")
	tdCloseCount := strings.Count(content, "</td>")
	thCount := strings.Count(content, "<th")
	thCloseCount := strings.Count(content, "</th>")
	trCount := strings.Count(content, "<tr")
	trCloseCount := strings.Count(content, "</tr>")

	// If the number of opening and closing tags don't match, the table is malformed
	if tdCount != tdCloseCount || thCount != thCloseCount || trCount != trCloseCount {
		return ""
	}

	// Parse the HTML content
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return ""
	}

	// Find the first cell that matches the header
	var headerCell *html.Node
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
			text := extractText(n)
			if strings.TrimSpace(text) == header {
				headerCell = n
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	if headerCell == nil {
		return ""
	}

	// Find the next cell
	nextCell := headerCell.NextSibling
	for nextCell != nil && nextCell.Type != html.ElementNode {
		nextCell = nextCell.NextSibling
	}
	if nextCell == nil || (nextCell.Data != "td" && nextCell.Data != "th") {
		return ""
	}

	// Extract the value from the next cell's children
	var value string
	for child := nextCell.FirstChild; child != nil; child = child.NextSibling {
		value += renderNode(child)
	}
	value = strings.TrimSpace(value)

	// Check for empty paragraph tags with various formats
	emptyPTags := []string{
		"<p />",
		"<p/>",
		"<p></p>",
		"<p> </p>",
		"<p>\n</p>",
		"<p>\n\t</p>",
		"<p>\n        </p>",
		"<p>  </p>",
	}

	for _, emptyTag := range emptyPTags {
		if strings.TrimSpace(value) == emptyTag {
			return ""
		}
	}

	// If the value doesn't contain any HTML tags, return it as is
	if !strings.Contains(value, "<") && !strings.Contains(value, ">") {
		return value
	}

	return value
}

// extractText extracts text content from a node
func extractText(n *html.Node) string {
	var text string
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			text += n.Data
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return strings.TrimSpace(text)
}

// renderNode renders a node back to HTML
func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}

// cleanHTML removes HTML tags and decodes entities
func cleanHTML(input string) string {
	// First, handle unicode-encoded HTML entities
	unicodeReplacements := map[string]string{
		"\u0026":        "&",
		"\u003c":        "<",
		"\u003e":        ">",
		"\u003cp":       "<p",
		"\u003c/p":      "</p",
		"\u003cli":      "<li",
		"\u003c/li":     "</li",
		"\u003cul":      "<ul",
		"\u003c/ul":     "</ul",
		"\u003cbr":      "<br",
		"\u003c/br":     "</br",
		"\u003cdiv":     "<div",
		"\u003c/div":    "</div",
		"\u003cspan":    "<span",
		"\u003c/span":   "</span",
		"\u003ca":       "<a",
		"\u003c/a":      "</a",
		"\u003ccode":    "<code",
		"\u003c/code":   "</code",
		"\u003cstrong":  "<strong",
		"\u003c/strong": "</strong",
		"\u003cem":      "<em",
		"\u003c/em":     "</em",
		"\u003cp /":     "<p",
		"\u003cp/":      "<p",
		"\u0026ldquo;":  "\"",
		"\u0026rdquo;":  "\"",
		"\u0026lsquo;":  "'",
		"\u0026rsquo;":  "'",
	}

	for unicode, replacement := range unicodeReplacements {
		input = strings.ReplaceAll(input, unicode, replacement)
	}

	// Handle common HTML entities
	replacements := map[string]string{
		"&mdash;":  "—",
		"&euro;":   "€",
		"&hellip;": "...",
		"&amp;":    "&",
		"&nbsp;":   " ",
		"&quot;":   "\"",
		"&rsquo;":  "'",
		"&lsquo;":  "'",
		"&rdquo;":  "\"",
		"&ldquo;":  "\"",
		"&ndash;":  "–",
		"&lt;":     "<",
		"&gt;":     ">",
	}

	for entity, replacement := range replacements {
		input = strings.ReplaceAll(input, entity, replacement)
	}

	// Check for empty paragraph tags before parsing
	if input == "" || input == "<p />" || input == "<p/>" || input == "<p></p>" ||
		input == "\u003cp /\u003e" || input == "\u003cp/\u003e" || input == "\u003cp\u003e\u003c/p\u003e" ||
		input == "\n" || input == "\r\n" || input == "\r" {
		return ""
	}

	// Remove HTML tags
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return input
	}

	var textContent strings.Builder
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			textContent.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
		// Add space after block elements
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div" || n.Data == "li" || n.Data == "br") {
			textContent.WriteString(" ")
		}
	}
	extractText(doc)

	// Clean up whitespace
	result := textContent.String()
	result = strings.Join(strings.Fields(result), " ")
	result = strings.TrimSpace(result)

	// Return empty string if the result is just whitespace
	if result == "" {
		return ""
	}

	return result
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
	// First try to extract date from time tag
	if strings.Contains(dateStr, "<time") {
		// Handle Unicode-encoded time tags
		dateStr = strings.ReplaceAll(dateStr, "\u003ctime", "<time")
		dateStr = strings.ReplaceAll(dateStr, "\u003c/time", "</time")

		// Find the datetime attribute
		start := strings.Index(dateStr, "datetime=\"")
		if start != -1 {
			start += len("datetime=\"")
			end := strings.Index(dateStr[start:], "\"")
			if end != -1 {
				dateStr = dateStr[start : start+end]
				// Try to parse the date
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					return t, nil
				}
			}
		}
	}

	// Clean up the date string
	dateStr = strings.TrimSpace(dateStr)
	dateStr = cleanHTML(dateStr)

	// Try each date format
	for _, format := range dateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", dateStr)
}
