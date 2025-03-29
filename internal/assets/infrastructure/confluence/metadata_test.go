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
				<tr><td><strong>Why are we doing this?</strong></td><td><p>Test why</p></td></tr>
				<tr><td><strong>Economic benefits</strong></td><td><p>Test benefits</p></td></tr>
				<tr><td><strong>How it works?</strong></td><td><p>Test how</p></td></tr>
				<tr><td><strong>How do we judge success?</strong></td><td><p>Test metrics</p></td></tr>
				<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
				<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
				<tr><td><strong>Launch date</strong></td><td><p>March 4, 2022</p></td></tr>
			</table>
			{"metadata":{"labels":{"results":[{"name":"test-label"},{"name":"cap-asset-test-asset"}]}}}`,
			expected: &PageMetadata{
				Description: "Test why",
				Why:         "Test why",
				Benefits:    "Test benefits",
				How:         "Test how",
				Metrics:     "Test metrics",
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
				<tr><td><strong>Why are we doing this?</strong></td><td><p>Test why</p></td></tr>
				<tr><td><strong>Economic benefits</strong></td><td><p>Test benefits</p></td></tr>
				<tr><td><strong>How it works?</strong></td><td><p>Test how</p></td></tr>
				<tr><td><strong>How do we judge success?</strong></td><td><p>Test metrics</p></td></tr>
				<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
				<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
				<tr><td><strong>Launch date</strong></td><td><p>March 4, 2022</p></td></tr>
			</table>
			<div class="labels">{"label":"test-label"},{"label":"cap-asset-test-asset"}</div>`,
			expected: &PageMetadata{
				Description: "Test why",
				Why:         "Test why",
				Benefits:    "Test benefits",
				How:         "Test how",
				Metrics:     "Test metrics",
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
				<tr><td><strong>Why are we doing this?</strong></td><td><p>Test why</p></td></tr>
				<tr><td><strong>Economic benefits</strong></td><td><p>Test benefits</p></td></tr>
				<tr><td><strong>How it works?</strong></td><td><p>Test how</p></td></tr>
				<tr><td><strong>How do we judge success?</strong></td><td><p>Test metrics</p></td></tr>
				<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
				<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
				<tr><td><strong>Launch date</strong></td><td><p>since 2022</p></td></tr>
			</table>
			<div class="labels">{"label":"test-label"},{"label":"cap-asset-test-asset"}</div>`,
			expected: &PageMetadata{
				Description: "Test why",
				Why:         "Test why",
				Benefits:    "Test benefits",
				How:         "Test how",
				Metrics:     "Test metrics",
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
				Why:         "",
				Benefits:    "Fallback description",
				How:         "",
				Metrics:     "",
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
			if got.Why != tt.expected.Why {
				t.Errorf("Why = %v, want %v", got.Why, tt.expected.Why)
			}
			if got.Benefits != tt.expected.Benefits {
				t.Errorf("Benefits = %v, want %v", got.Benefits, tt.expected.Benefits)
			}
			if got.How != tt.expected.How {
				t.Errorf("How = %v, want %v", got.How, tt.expected.How)
			}
			if got.Metrics != tt.expected.Metrics {
				t.Errorf("Metrics = %v, want %v", got.Metrics, tt.expected.Metrics)
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
			name:     "extract value from standard table with td tags",
			content:  `<table><tr><td><strong>Test</strong></td><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "<p>Value</p>",
		},
		{
			name:     "extract value from table with th tags",
			content:  `<table><tr><th data-highlight-colour="#e3fcef"><p><strong>Why are we doing this?</strong></p></th><th data-highlight-colour="#ffffff"><p>Test value</p></th></tr></table>`,
			header:   "Why are we doing this?",
			expected: "<p>Test value</p>",
		},
		{
			name:     "extract value from mixed td and th tags",
			content:  `<table><tr><th><strong>Test</strong></th><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "<p>Value</p>",
		},
		{
			name:     "extract value with background color",
			content:  `<table><tr><td data-highlight-colour="#e3fcef"><p><strong>Test</strong></p></td><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "<p>Value</p>",
		},
		{
			name:     "extract value with complex content",
			content:  `<table><tr><td><strong>Test</strong></td><td><p>First line<br/>Second line</p></td></tr></table>`,
			header:   "Test",
			expected: "<p>First line<br/>Second line</p>",
		},
		{
			name:     "extract value when header is not in strong tags",
			content:  `<table><tr><td>Test</td><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "<p>Value</p>",
		},
		{
			name:     "header not found",
			content:  `<table><tr><td><strong>Other</strong></td><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "",
		},
		{
			name:     "malformed table - no closing td",
			content:  `<table><tr><td><strong>Test</strong><td><p>Value</p></td></tr></table>`,
			header:   "Test",
			expected: "",
		},
		{
			name:     "malformed table - no closing tr",
			content:  `<table><tr><td><strong>Test</strong></td><td><p>Value</p></td></table>`,
			header:   "Test",
			expected: "",
		},
		{
			name:     "extract value with status macro",
			content:  `<table><tr><td><strong>Status</strong></td><td><p><ac:structured-macro ac:name="status"><ac:parameter ac:name="title">in development</ac:parameter></ac:structured-macro></p></td></tr></table>`,
			header:   "Status",
			expected: "<p><ac:structured-macro ac:name=\"status\"><ac:parameter ac:name=\"title\">in development</ac:parameter></ac:structured-macro></p>",
		},
		{
			name: "extract value from table cell",
			content: `
				<table>
					<tr>
						<td><strong>Test Header</strong></td>
						<td>Test Value</td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "Test Value",
		},
		{
			name: "extract value from table cell with paragraph",
			content: `
				<table>
					<tr>
						<td><p><strong>Test Header</strong></p></td>
						<td><p>Test Value</p></td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "<p>Test Value</p>",
		},
		{
			name: "extract value from table cell with highlight",
			content: `
				<table>
					<tr>
						<td data-highlight-colour="#f4f5f7"><p><strong>Test Header</strong></p></td>
						<td><p>Test Value</p></td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "<p>Test Value</p>",
		},
		{
			name: "handle empty paragraph tag",
			content: `
				<table>
					<tr>
						<td><p><strong>Test Header</strong></p></td>
						<td><p /></td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "",
		},
		{
			name: "handle empty paragraph tag with Unicode",
			content: `
				<table>
					<tr>
						<td><p><strong>Test Header</strong></p></td>
						<td>\u003cp /\u003e</td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "",
		},
		{
			name: "handle empty paragraph tag with whitespace",
			content: `
				<table>
					<tr>
						<td><p><strong>Test Header</strong></p></td>
						<td><p /> </td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "",
		},
		{
			name: "handle empty paragraph tag with newline",
			content: `
				<table>
					<tr>
						<td><p><strong>Test Header</strong></p></td>
						<td><p />
</td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "",
		},
		{
			name: "handle empty paragraph tag with multiple spaces",
			content: `
				<table>
					<tr>
						<td><p><strong>Test Header</strong></p></td>
						<td>  <p />  </td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "",
		},
		{
			name: "handle empty paragraph tag with Unicode and whitespace",
			content: `
				<table>
					<tr>
						<td><p><strong>Test Header</strong></p></td>
						<td>  \u003cp /\u003e  </td>
					</tr>
				</table>`,
			header:   "Test Header",
			expected: "",
		},
		{
			name: "extract launch date with time tag",
			content: `
				<table>
					<tr>
						<td data-highlight-colour="#f4f5f7"><p><strong>Launch date</strong></p></td>
						<td><p><time datetime="2019-11-20" /></p></td>
					</tr>
				</table>`,
			header:   "Launch date",
			expected: "<p><time datetime=\"2019-11-20\"></time></p>",
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
		{
			name:     "handle empty p tags",
			input:    "<p /> <p></p>",
			expected: "",
		},
		{
			name:     "handle code tags",
			input:    "This is a <code>bcp</code> in the codebase",
			expected: "This is a bcp in the codebase",
		},
		{
			name:     "handle links",
			input:    "Check the <a href=\"http://example.com\">dashboard</a>",
			expected: "Check the dashboard",
		},
		{
			name:     "handle unicode entities",
			input:    "Life doesn\u0026rsquo;t always go according to plan\u0026hellip;",
			expected: "Life doesn't always go according to plan...",
		},
		{
			name:     "handle multiple mixed elements",
			input:    "<p>Check the <a href=\"http://example.com\"><code>dashboard</code></a> for &amp; metrics\u0026hellip;</p>",
			expected: "Check the dashboard for & metrics...",
		},
		{
			name:     "handle euro symbol",
			input:    "Price is \u0026euro;5 or €5",
			expected: "Price is €5 or €5",
		},
		{
			name:     "handle quotes",
			input:    "\u0026quot;Lock the price\u0026quot; button",
			expected: "\"Lock the price\" button",
		},
		{
			name:     "handle lists",
			input:    "<ul><li>First item</li><li>Second item</li></ul>",
			expected: "First item Second item",
		},
		{
			name:     "handle encoded angle brackets",
			input:    "\u003cli\u003eTest item\u003c/li\u003e",
			expected: "Test item",
		},
		{
			name:     "handle complex mixed content",
			input:    "\u003cul\u003e\u003cli\u003ePrice is \u0026euro;750k - \u0026euro;1.85M\u003c/li\u003e\u003c/ul\u003e",
			expected: "Price is €750k - €1.85M",
		},
		{
			name:     "handle em dash",
			input:    "Data sources\u0026mdash;the Provider Transaction File\u0026mdash;reconciliation",
			expected: "Data sources—the Provider Transaction File—reconciliation",
		},
		{
			name:     "handle multiple unicode ampersands",
			input:    "\u0026mdash;\u0026euro;\u0026hellip;",
			expected: "—€...",
		},
		{
			name:     "handle mixed HTML and unicode entities",
			input:    "&mdash;\u0026euro;&hellip;",
			expected: "—€...",
		},
		{
			name:     "handle unicode paragraph tags",
			input:    "\u003cp\u003eTest paragraph\u003c/p\u003e",
			expected: "Test paragraph",
		},
		{
			name:     "handle unicode list tags",
			input:    "\u003cul\u003e\u003cli\u003eItem 1\u003c/li\u003e\u003cli\u003eItem 2\u003c/li\u003e\u003c/ul\u003e",
			expected: "Item 1 Item 2",
		},
		{
			name:     "handle unicode code tags",
			input:    "\u003ccode\u003eTest code\u003c/code\u003e",
			expected: "Test code",
		},
		{
			name:     "handle unicode strong tags",
			input:    "\u003cstrong\u003eBold text\u003c/strong\u003e",
			expected: "Bold text",
		},
		{
			name:     "handle unicode em tags",
			input:    "\u003cem\u003eItalic text\u003c/em\u003e",
			expected: "Italic text",
		},
		{
			name:     "handle unicode div tags",
			input:    "\u003cdiv\u003eDiv content\u003c/div\u003e",
			expected: "Div content",
		},
		{
			name:     "handle unicode span tags",
			input:    "\u003cspan\u003eSpan content\u003c/span\u003e",
			expected: "Span content",
		},
		{
			name:     "handle unicode br tags",
			input:    "Line 1\u003cbr\u003eLine 2",
			expected: "Line 1 Line 2",
		},
		{
			name:     "handle complex unicode HTML",
			input:    "\u003cp\u003eThis is a \u003cstrong\u003ebold\u003c/strong\u003e and \u003cem\u003eitalic\u003c/em\u003e text with a \u003ca href=\"http://example.com\"\u003elink\u003c/a\u003e\u003c/p\u003e",
			expected: "This is a bold and italic text with a link",
		},
		{
			name:     "handle empty unicode paragraph tags",
			input:    "\u003cp /\u003e",
			expected: "",
		},
		{
			name:     "handle unicode quotes",
			input:    "\u0026ldquo;Vio\u0026rdquo;",
			expected: "\"Vio\"",
		},
		{
			name:     "handle unicode single quotes",
			input:    "\u0026lsquo;test\u0026rsquo;",
			expected: "'test'",
		},
		{
			name:     "handle mixed unicode and HTML quotes",
			input:    "\u0026ldquo;test\u0026rdquo; and &quot;test&quot;",
			expected: "\"test\" and \"test\"",
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

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "parse date from time tag",
			input:    "<p><time datetime=\"2019-11-20\"></time></p>",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with space",
			input:    "<p><time datetime=\"2019-11-20\"> </time></p>",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with content",
			input:    "<p><time datetime=\"2019-11-20\">November 20, 2019</time></p>",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with Unicode",
			input:    "\u003cp\u003e\u003ctime datetime=\"2019-11-20\"\u003e\u003c/time\u003e\u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with self-closing",
			input:    "<p><time datetime=\"2019-11-20\" /></p>",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with self-closing and Unicode",
			input:    "\u003cp\u003e\u003ctime datetime=\"2019-11-20\" /\u003e\u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with whitespace",
			input:    "<p>  <time datetime=\"2019-11-20\">  </time>  </p>",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with newline",
			input:    "<p>\n<time datetime=\"2019-11-20\">\n</time>\n</p>",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with multiple spaces",
			input:    "<p>  <time datetime=\"2019-11-20\">  </time>  </p>",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with Unicode and whitespace",
			input:    "\u003cp\u003e  \u003ctime datetime=\"2019-11-20\"\u003e  \u003c/time\u003e  \u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with Unicode and newline",
			input:    "\u003cp\u003e\n\u003ctime datetime=\"2019-11-20\"\u003e\n\u003c/time\u003e\n\u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with Unicode and multiple spaces",
			input:    "\u003cp\u003e  \u003ctime datetime=\"2019-11-20\"\u003e  \u003c/time\u003e  \u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with Unicode and self-closing",
			input:    "\u003cp\u003e  \u003ctime datetime=\"2019-11-20\" /\u003e  \u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with Unicode and self-closing and newline",
			input:    "\u003cp\u003e\n  \u003ctime datetime=\"2019-11-20\" /\u003e\n  \u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "parse date from time tag with Unicode and self-closing and multiple spaces",
			input:    "\u003cp\u003e  \u003ctime datetime=\"2019-11-20\" /\u003e  \u003c/p\u003e",
			expected: time.Date(2019, 11, 20, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.expected) {
				t.Errorf("parseDate() = %v, want %v", got, tt.expected)
			}
		})
	}
}
