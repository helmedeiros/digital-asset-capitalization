package confluence

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// emptyPlaceholder is the HTML representation of an empty field
const emptyPlaceholder = `<p>-</p>`

// StatusBadge represents a Confluence status badge with color
type StatusBadge struct {
	Title  string
	Colour string
}

// GetStatusBadge maps asset status to Confluence badge configuration
func GetStatusBadge(status string) StatusBadge {
	status = strings.TrimSpace(strings.ToLower(status))

	switch status {
	case "live", "production", "active":
		return StatusBadge{Title: "Live", Colour: "Green"}
	case "beta", "pilot":
		return StatusBadge{Title: "Beta", Colour: "Blue"}
	case "in progress", "in-progress", "inprogress":
		return StatusBadge{Title: "In Progress", Colour: "Blue"}
	case "development", "in development", "wip":
		return StatusBadge{Title: "Development", Colour: "Yellow"}
	case "planning", "planned":
		return StatusBadge{Title: "Planning", Colour: "Grey"}
	case "deprecated", "retiring", "sunset":
		return StatusBadge{Title: "Deprecated", Colour: "Red"}
	case "paused", "on hold":
		return StatusBadge{Title: "Paused", Colour: "Yellow"}
	default:
		if status != "" {
			// Capitalize first letter for display
			return StatusBadge{Title: capitalizeFirst(status), Colour: "Grey"}
		}
		return StatusBadge{Title: "Unknown", Colour: "Grey"}
	}
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// GeneratePageContent creates Confluence storage format HTML from an asset
func GeneratePageContent(asset *domain.Asset) string {
	var sb strings.Builder

	// Main title
	sb.WriteString(`<h1>Asset Capitalisation</h1>`)

	// Overview section
	sb.WriteString(`<h2>Overview</h2>`)
	sb.WriteString(generateOverviewSection(asset))

	// Value section
	sb.WriteString(`<h2>Value</h2>`)
	sb.WriteString(generateValueSection(asset))

	// Asset Checklist section
	sb.WriteString(`<h2>Asset Checklist</h2>`)
	sb.WriteString(generateChecklistSection())

	return sb.String()
}

// generateOverviewSection creates the overview table with asset metadata
func generateOverviewSection(asset *domain.Asset) string {
	var sb strings.Builder

	// Overview table with grey header column
	sb.WriteString(`<table data-layout="default">`)
	sb.WriteString(`<colgroup><col style="width: 340.0px;" /><col style="width: 680.0px;" /></colgroup>`)
	sb.WriteString(`<tbody>`)

	// Asset name row
	sb.WriteString(generateOverviewTableRow("Asset", escapeHTML(asset.Name)))

	// Owner row
	ownerValue := asset.GetOwningTeam()
	if ownerValue == "" {
		ownerValue = "-"
	}
	sb.WriteString(generateOverviewTableRow("Asset owned by", escapeHTML(ownerValue)))

	// Tribe row (derived from team or empty)
	sb.WriteString(generateOverviewTableRow("Tribe", "-"))

	// Pod row
	podValue := asset.Platform
	if podValue == "" {
		podValue = "-"
	}
	sb.WriteString(generateOverviewTableRow("Pod", escapeHTML(podValue)))

	// Launch date row - format as human readable
	launchDateValue := "-"
	if !asset.LaunchDate.IsZero() {
		launchDateValue = formatHumanDate(asset.LaunchDate)
	}
	sb.WriteString(generateOverviewTableRow("Launch date", launchDateValue))

	// Retiring date row (new field, empty by default)
	sb.WriteString(generateOverviewTableRow("Retiring date", "-"))

	// Status row with badge
	statusBadge := GetStatusBadge(asset.Status)
	statusValue := generateStatusMacro(statusBadge)
	sb.WriteString(generateOverviewTableRow("Status", statusValue))

	sb.WriteString(`</tbody>`)
	sb.WriteString(`</table>`)

	return sb.String()
}

// generateValueSection creates the value table with why, benefits, metrics, how
func generateValueSection(asset *domain.Asset) string {
	var sb strings.Builder

	// Value table with green header column
	sb.WriteString(`<table data-layout="default">`)
	sb.WriteString(`<colgroup><col style="width: 340.0px;" /><col style="width: 680.0px;" /></colgroup>`)
	sb.WriteString(`<tbody>`)

	// Why row - typically a paragraph
	whyValue := asset.Why
	if whyValue == "" {
		whyValue = "-"
	}
	sb.WriteString(generateValueTableRow("Why are we doing this?", formatAsContent(whyValue)))

	// Benefits row - typically a bullet list
	benefitsValue := asset.Benefits
	if benefitsValue == "" {
		benefitsValue = "-"
	}
	sb.WriteString(generateValueTableRow("Economic benefits", formatAsContent(benefitsValue)))

	// Metrics row - typically a bullet list
	metricsValue := asset.Metrics
	if metricsValue == "" {
		metricsValue = "-"
	}
	sb.WriteString(generateValueTableRow("How do we judge success?", formatAsContent(metricsValue)))

	// How row - typically a bullet list
	howValue := asset.How
	if howValue == "" {
		howValue = "-"
	}
	sb.WriteString(generateValueTableRow("How it works?", formatAsContent(howValue)))

	sb.WriteString(`</tbody>`)
	sb.WriteString(`</table>`)

	return sb.String()
}

// generateTableRow creates a table row with header and value cells (legacy, used by tests)
func generateTableRow(header, value string) string {
	return fmt.Sprintf(`<tr><th><p><strong>%s</strong></p></th><td>%s</td></tr>`, header, value)
}

// generateOverviewTableRow creates a table row with grey background header cell
func generateOverviewTableRow(header, value string) string {
	return fmt.Sprintf(
		`<tr><th style="background-color: #f4f5f7;"><p><strong>%s</strong></p></th><td><p>%s</p></td></tr>`,
		header, value)
}

// generateValueTableRow creates a table row with green background header cell
func generateValueTableRow(header, value string) string {
	return fmt.Sprintf(
		`<tr><th style="background-color: #e3fcef;"><p><em>%s</em></p></th><td>%s</td></tr>`,
		header, value)
}

// generateStatusMacro creates a Confluence status macro
func generateStatusMacro(badge StatusBadge) string {
	return fmt.Sprintf(
		`<ac:structured-macro ac:name="status" ac:schema-version="1">`+
			`<ac:parameter ac:name="title">%s</ac:parameter>`+
			`<ac:parameter ac:name="colour">%s</ac:parameter>`+
			`</ac:structured-macro>`,
		escapeHTML(badge.Title),
		badge.Colour,
	)
}

// generateDateMacro creates a Confluence date macro
func generateDateMacro(dateStr string) string {
	return fmt.Sprintf(
		`<time datetime="%s" />`,
		dateStr,
	)
}

// formatHumanDate formats a time.Time as "Jan 2, 2006" format
func formatHumanDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// formatAsContent formats content as paragraphs or bullet lists
// If content has multiple lines separated by newlines, it creates a bullet list
// Otherwise, it wraps in paragraph tags
func formatAsContent(content string) string {
	if content == "-" || content == "" {
		return emptyPlaceholder
	}

	// Escape HTML first
	content = escapeHTML(content)

	// Split by newlines
	lines := strings.Split(content, "\n")
	var nonEmptyLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) == 0 {
		return emptyPlaceholder
	}

	// If single line, return as paragraph
	if len(nonEmptyLines) == 1 {
		return fmt.Sprintf("<p>%s</p>", nonEmptyLines[0])
	}

	// Multiple lines - create bullet list
	var sb strings.Builder
	sb.WriteString(`<ul>`)
	for _, line := range nonEmptyLines {
		// Remove leading bullet characters if present
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "• ")
		line = strings.TrimPrefix(line, "* ")
		sb.WriteString(fmt.Sprintf(`<li><p>%s</p></li>`, line))
	}
	sb.WriteString(`</ul>`)
	return sb.String()
}

// generateChecklistSection creates the static Asset Checklist section
func generateChecklistSection() string {
	checklistItems := []string{
		"The asset's use can be proven to be technically feasible",
		"We intend to complete the asset and to use it",
		"Future economic benefits through use are probable",
		"We have the resources (technically, financially etc) to be able to complete the asset and use it",
		"We can reliably measure the expenditure that can be capitalised",
	}

	var sb strings.Builder
	sb.WriteString(`<ac:task-list>`)
	for _, item := range checklistItems {
		sb.WriteString(`<ac:task>`)
		sb.WriteString(`<ac:task-status>complete</ac:task-status>`)
		sb.WriteString(fmt.Sprintf(`<ac:task-body><span class="placeholder-inline-tasks">%s</span></ac:task-body>`, escapeHTML(item)))
		sb.WriteString(`</ac:task>`)
	}
	sb.WriteString(`</ac:task-list>`)
	return sb.String()
}

// formatMultilineContent wraps content in paragraph tags and handles line breaks
func formatMultilineContent(content string) string {
	if content == "-" {
		return emptyPlaceholder
	}

	// Escape HTML first
	content = escapeHTML(content)

	// Split by newlines and wrap each line in a paragraph
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, fmt.Sprintf("<p>%s</p>", line))
		}
	}

	if len(result) == 0 {
		return emptyPlaceholder
	}

	return strings.Join(result, "")
}

// escapeHTML escapes special characters for HTML
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// PagePublishResult contains the result of publishing an asset to Confluence
type PagePublishResult struct {
	PageID   string
	PageURL  string
	SpaceKey string
	Title    string
	Created  bool
}

// CreatePageRequest represents the request body for creating a Confluence page
type CreatePageRequest struct {
	Type  string          `json:"type"`
	Title string          `json:"title"`
	Space CreatePageSpace `json:"space"`
	Body  CreatePageBody  `json:"body"`
}

// CreatePageSpace represents the space key in a create page request
type CreatePageSpace struct {
	Key string `json:"key"`
}

// CreatePageBody represents the body content in a create page request
type CreatePageBody struct {
	Storage CreatePageStorage `json:"storage"`
}

// CreatePageStorage represents the storage format content
type CreatePageStorage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// LabelRequest represents a request to add labels to a page
type LabelRequest struct {
	Prefix string `json:"prefix"`
	Name   string `json:"name"`
}
