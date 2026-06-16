package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTable_formatValue covers both branches of the Table.formatValue
// helper. The existing test suite only exercised the table tests that
// went through the column-formatter path; the nil-formatter fallback
// to DefaultFormatter wasn't reached directly.
func TestTable_formatValue(t *testing.T) {
	tbl := &Table{}

	t.Run("with formatter delegates and ignores DefaultFormatter", func(t *testing.T) {
		called := false
		got := tbl.formatValue("raw", func(v interface{}) string {
			called = true
			return "shaped:" + v.(string)
		})
		assert.True(t, called, "formatter must be called when non-nil")
		assert.Equal(t, "shaped:raw", got)
	})

	t.Run("without formatter falls back to DefaultFormatter", func(t *testing.T) {
		// DefaultFormatter renders strings unchanged.
		assert.Equal(t, "raw", tbl.formatValue("raw", nil))
	})
}
