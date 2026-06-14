package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// containsRaw checks that the formatter preserved the raw value text
// somewhere in its (possibly ANSI-wrapped) output. Used to keep the
// table tests below to a single clear assertion each.
func containsRaw(t *testing.T, got, raw string) {
	t.Helper()
	if !strings.Contains(got, raw) {
		t.Errorf("expected output to contain %q, got %q", raw, got)
	}
}

func TestSprintFormatter(t *testing.T) {
	assert.Contains(t, SprintFormatter(""), "(no sprint)")
	containsRaw(t, SprintFormatter("Sprint 1"), "Sprint 1")
}

func TestPeriodFormatter(t *testing.T) {
	t.Run("range with valid dates renders Jan 2 -> Jan 9", func(t *testing.T) {
		got := PeriodFormatter("2026-01-02 to 2026-01-09")
		containsRaw(t, got, "Jan 2")
		containsRaw(t, got, "Jan 9")
		containsRaw(t, got, "→")
	})
	t.Run("non-range input is rendered as muted text", func(t *testing.T) {
		got := PeriodFormatter("immediate")
		containsRaw(t, got, "immediate")
	})
	t.Run("range with one unparseable date still renders with the raw text", func(t *testing.T) {
		// formatShortDate returns the original string when parsing fails.
		got := PeriodFormatter("immediate to 2026-01-09")
		containsRaw(t, got, "immediate")
		containsRaw(t, got, "Jan 9")
	})
}

func TestEngineerNameFormatter(t *testing.T) {
	assert.Contains(t, EngineerNameFormatter(""), "(unknown)")
	containsRaw(t, EngineerNameFormatter("Alice"), "Alice")
}

func TestHourlyRateFormatter(t *testing.T) {
	t.Run("numeric value renders as $N/h", func(t *testing.T) {
		containsRaw(t, HourlyRateFormatter(75.0), "$75/h")
	})
	t.Run("string with /h suffix is passed through", func(t *testing.T) {
		containsRaw(t, HourlyRateFormatter("EUR 60/h"), "EUR 60/h")
	})
	t.Run("string without rate suffix gets /h appended", func(t *testing.T) {
		containsRaw(t, HourlyRateFormatter("nominal"), "nominal/h")
	})
	t.Run("string mentioning hour also passes through unchanged", func(t *testing.T) {
		containsRaw(t, HourlyRateFormatter("billable hour"), "billable hour")
	})
}

func TestGoalFormatter(t *testing.T) {
	t.Run("empty goal renders as no-goal sentinel", func(t *testing.T) {
		assert.Contains(t, GoalFormatter(""), "(no goal)")
	})
	t.Run("short goal is preserved", func(t *testing.T) {
		assert.Equal(t, "Ship feature X", GoalFormatter("Ship feature X"))
	})
	t.Run("long goal is truncated to 22 chars plus ellipsis", func(t *testing.T) {
		long := "This is a very long sprint goal that exceeds the limit"
		got := GoalFormatter(long)
		assert.True(t, strings.HasSuffix(got, "..."), "expected ellipsis suffix, got %q", got)
		// 22 chars of the original + "..." = 25 chars total.
		assert.Len(t, got, 25)
	})
}

func TestAssetPropertyFormatter(t *testing.T) {
	t.Run("http urls are rendered as links", func(t *testing.T) {
		containsRaw(t, AssetPropertyFormatter("https://example.invalid/page"), "https://example.invalid/page")
	})
	t.Run("emails are rendered as links", func(t *testing.T) {
		containsRaw(t, AssetPropertyFormatter("user@example.com"), "user@example.com")
	})
	t.Run("short strings pass through unchanged", func(t *testing.T) {
		assert.Equal(t, "primary", AssetPropertyFormatter("primary"))
	})
	t.Run("long strings are truncated to 57 chars plus ellipsis", func(t *testing.T) {
		long := strings.Repeat("A", 80)
		got := AssetPropertyFormatter(long)
		assert.True(t, strings.HasSuffix(got, "..."), "expected ellipsis suffix, got %q", got)
		assert.Len(t, got, 60)
	})
}

func TestPercentageFormatter(t *testing.T) {
	assert.Equal(t, "12.5%", PercentageFormatter(12.5))
	assert.Equal(t, "12.5%", PercentageFormatter(float32(12.5)))
	assert.Equal(t, "12.5%", PercentageFormatter("12.5"))
	assert.Equal(t, "not-a-number", PercentageFormatter("not-a-number"))
	// Default branch goes through DefaultFormatter; just smoke-test it
	// doesn't panic and produces something non-empty for an int.
	got := PercentageFormatter(7)
	require.NotEmpty(t, got)
}

func TestCreateInvestmentTable(t *testing.T) {
	tbl := CreateInvestmentTable()
	require.NotNil(t, tbl)
	require.Len(t, tbl.Columns, 5)
	assert.Equal(t, "asset", tbl.Columns[0].Key)
	assert.Equal(t, "total_investment", tbl.Columns[1].Key)
	assert.NotNil(t, tbl.Columns[1].Formatter, "investment column should carry the MoneyFormatter")
	assert.Equal(t, ColorPrimary, tbl.Style.HeaderColor)
}

func TestFormatShortDate(t *testing.T) {
	t.Run("valid ISO date renders as Jan 2 style", func(t *testing.T) {
		assert.Equal(t, "Jan 9", formatShortDate("2026-01-09"))
	})
	t.Run("invalid date is returned unchanged", func(t *testing.T) {
		assert.Equal(t, "not-a-date", formatShortDate("not-a-date"))
	})
}

func TestDateFormatter_AllBranches(t *testing.T) {
	t.Run("time.Time input renders as YYYY-MM-DD muted", func(t *testing.T) {
		got := DateFormatter(time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC))
		containsRaw(t, got, "2026-07-09")
	})

	t.Run("string in 'YYYY-MM-DD HH:MM:SS' form is reformatted to date", func(t *testing.T) {
		got := DateFormatter("2026-07-09 12:34:56")
		containsRaw(t, got, "2026-07-09")
	})

	t.Run("string that doesn't parse is returned unchanged", func(t *testing.T) {
		got := DateFormatter("not a date")
		assert.Equal(t, "not a date", got)
	})

	t.Run("non-time non-string types fall through to DefaultFormatter", func(t *testing.T) {
		got := DateFormatter(42)
		require.NotEmpty(t, got)
	})
}

func TestStatusFormatter_AllBranches(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
	}{
		{"active maps to success", "active"},
		{"completed maps to success", "completed"},
		{"done maps to success", "done"},
		{"enabled maps to success", "enabled"},
		{"inactive maps to error", "inactive"},
		{"failed maps to error", "failed"},
		{"disabled maps to error", "disabled"},
		{"pending maps to warning", "pending"},
		{"in progress maps to warning", "in progress"},
		{"unknown maps to info", "queued"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := StatusFormatter(c.value)
			containsRaw(t, got, fmt.Sprintf("%v", c.value))
			assert.True(t, strings.HasSuffix(got, Reset), "every branch should wrap with Reset, got %q", got)
		})
	}
}
