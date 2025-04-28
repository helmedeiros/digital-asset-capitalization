package domain

// ClassifyTasksInput represents the input parameters for classifying tasks
type ClassifyTasksInput struct {
	Project string
	Sprint  string
	DryRun  bool
	Apply   bool
}
