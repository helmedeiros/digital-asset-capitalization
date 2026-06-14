package confluence

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAsContent_EmptyOrDashYieldsPlaceholder(t *testing.T) {
	assert.Equal(t, emptyPlaceholder, formatAsContent(""))
	assert.Equal(t, emptyPlaceholder, formatAsContent("-"))
}

func TestFormatAsContent_WhitespaceOnlyLinesYieldPlaceholder(t *testing.T) {
	// All lines are whitespace, so nonEmptyLines stays empty and the
	// function falls through to the placeholder.
	assert.Equal(t, emptyPlaceholder, formatAsContent("   \n\t\n "))
}

func TestFormatAsContent_SingleLineWrapsInParagraph(t *testing.T) {
	got := formatAsContent("Hello world")
	assert.Equal(t, "<p>Hello world</p>", got)
}

func TestFormatAsContent_SingleLineEscapesHTML(t *testing.T) {
	got := formatAsContent("<script>alert(1)</script>")
	assert.NotContains(t, got, "<script>")
	assert.Contains(t, got, "&lt;script&gt;")
}

func TestFormatAsContent_MultilineBuildsBulletList(t *testing.T) {
	got := formatAsContent("first\nsecond\nthird")
	assert.True(t, strings.HasPrefix(got, "<ul>"))
	assert.True(t, strings.HasSuffix(got, "</ul>"))
	assert.Contains(t, got, "<li><p>first</p></li>")
	assert.Contains(t, got, "<li><p>second</p></li>")
	assert.Contains(t, got, "<li><p>third</p></li>")
}

func TestFormatAsContent_StripsLeadingBulletMarkers(t *testing.T) {
	cases := []string{
		"- alpha\n• beta\n* gamma",
	}
	for _, in := range cases {
		got := formatAsContent(in)
		assert.Contains(t, got, "<li><p>alpha</p></li>")
		assert.Contains(t, got, "<li><p>beta</p></li>")
		assert.Contains(t, got, "<li><p>gamma</p></li>")
		// The raw bullet markers must NOT survive in the rendered HTML
		// (other than as parts of legitimate HTML).
		assert.NotContains(t, got, "<li><p>- alpha")
		assert.NotContains(t, got, "<li><p>• beta")
		assert.NotContains(t, got, "<li><p>* gamma")
	}
}

func TestFormatAsContent_MultilineEscapesHTMLPerLine(t *testing.T) {
	got := formatAsContent("first <b>bold</b>\nsecond & more")
	assert.Contains(t, got, "&lt;b&gt;bold&lt;/b&gt;")
	assert.Contains(t, got, "second &amp; more")
}

func TestFormatAsContent_BlankLinesBetweenItemsAreDropped(t *testing.T) {
	got := formatAsContent("alpha\n\n\nbeta")
	assert.Contains(t, got, "<li><p>alpha</p></li>")
	assert.Contains(t, got, "<li><p>beta</p></li>")
	// Only two list items emitted.
	assert.Equal(t, 2, strings.Count(got, "<li>"))
}
