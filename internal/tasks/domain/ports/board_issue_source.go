package ports

import "context"

// BoardIssueSource lists the issue keys of a sprint across the boards a
// project has configured, including kanban boards that have no sprints of
// their own (matched by the sprint's date window).
type BoardIssueSource interface {
	// FindIssueKeysForSprint returns the keys of issues on the project's
	// configured boards for the given sprint
	FindIssueKeysForSprint(ctx context.Context, project, sprint string) ([]string, error)
}
