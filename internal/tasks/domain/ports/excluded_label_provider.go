package ports

// ExcludedLabelProvider tells which JIRA labels mark a project's cards as
// excluded from capitalization (e.g. umbrella tickets whose work is tracked
// by child tickets).
type ExcludedLabelProvider interface {
	// GetExcludedLabels returns the project's excluded labels, or nil
	GetExcludedLabels(project string) []string
}
