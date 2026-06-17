package domain

import "testing"

// TestMinInt_BothBranches covers the b<=a branch that wasn't
// exercised by the existing jira_config tests.
func TestMinInt_BothBranches(t *testing.T) {
	t.Parallel()
	if got := minInt(1, 2); got != 1 {
		t.Errorf("minInt(1,2) = %d, want 1", got)
	}
	if got := minInt(5, 3); got != 3 {
		t.Errorf("minInt(5,3) = %d, want 3", got)
	}
	if got := minInt(7, 7); got != 7 {
		t.Errorf("minInt(7,7) tie should pick b, got %d", got)
	}
}
