package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderStructuredOutput_BlankLinesArePreserved(t *testing.T) {
	ps := NewPromptSession(80)
	got := ps.renderStructuredOutput("\n   \n")
	// Two empty/whitespace-only lines plus a trailing empty split.
	assert.Equal(t, "\n\n\n", got)
}

func TestRenderStructuredOutput_KeyValueLineIsColorized(t *testing.T) {
	ps := NewPromptSession(80)
	got := ps.renderStructuredOutput("name: AssetCap")
	// Key gets the primary-color prefix; value goes through formatValue.
	assert.Contains(t, got, "name:")
	assert.Contains(t, got, "AssetCap")
}

func TestRenderStructuredOutput_PlainURLLineSkipsKeyValueSplit(t *testing.T) {
	ps := NewPromptSession(80)
	got := ps.renderStructuredOutput("http://example.invalid/page")
	// Lines starting with http even when they contain a colon should
	// NOT be split as key:value — they fall through to the plain
	// regular-line branch.
	assert.Contains(t, got, "http://example.invalid/page")
}

func TestRenderStructuredOutput_LineWithSingleColonOnlySplitsOnce(t *testing.T) {
	ps := NewPromptSession(80)
	got := ps.renderStructuredOutput("title: a:b:c")
	// SplitN with n=2 must leave the rest intact under the value.
	assert.Contains(t, got, "a:b:c")
}

func TestRenderStructuredOutput_LineWithoutColonGoesThroughRegularBranch(t *testing.T) {
	ps := NewPromptSession(80)
	got := ps.renderStructuredOutput("just a heading")
	// The line is preserved with its trailing newline.
	assert.True(t, strings.Contains(got, "just a heading"))
}
