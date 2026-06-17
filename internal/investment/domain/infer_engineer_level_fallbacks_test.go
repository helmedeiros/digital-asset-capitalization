package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInferEngineerLevel_FallbackRateRanges exercises the rate-range
// fallback branch of InferEngineerLevel — the one that fires when the
// cost model has no defaults configured. The existing test sets
// defaults for every level, so this branch was untested.
func TestInferEngineerLevel_FallbackRateRanges(t *testing.T) {
	t.Parallel()
	cm, err := NewCostModel(EUR, 8.0, 2.0)
	require.NoError(t, err)
	// No SetDefaultRate calls: forces the switch fallback for every input.

	cases := []struct {
		name string
		rate float64
		want EngineerLevel
	}{
		{"80 -> Principal lower bound", 80.0, Principal},
		{"100 -> Principal high", 100.0, Principal},
		{"70 -> Staff lower bound", 70.0, Staff},
		{"79.99 -> Staff just under principal", 79.99, Staff},
		{"60 -> Senior lower bound", 60.0, Senior},
		{"69.99 -> Senior just under staff", 69.99, Senior},
		{"45 -> Mid lower bound", 45.0, Mid},
		{"59.99 -> Mid just under senior", 59.99, Mid},
		{"44.99 -> Junior just under mid", 44.99, Junior},
		{"0 -> Junior", 0.0, Junior},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, cm.InferEngineerLevel(c.rate))
		})
	}
}

// TestInferEngineerLevel_OutsideToleranceFallsThrough exercises the
// case where some default rates are configured but the input rate is
// far outside their +/-10% tolerance — the loop completes without a
// match and the rate-range switch decides instead.
func TestInferEngineerLevel_OutsideToleranceFallsThrough(t *testing.T) {
	t.Parallel()
	cm, err := NewCostModel(EUR, 8.0, 2.0)
	require.NoError(t, err)
	// Configure ONLY the Senior default. A rate of 30 is well outside
	// 65 +/- 10%, so the configured-rate loop must fail and the
	// fallback switch must return Junior.
	cm.SetDefaultRate(Senior, 65.0)
	assert.Equal(t, Junior, cm.InferEngineerLevel(30.0))
}
