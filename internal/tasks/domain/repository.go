package domain

// TaskRepository defines the interface for task storage operations
type TaskRepository interface {
	// Save saves or updates a task
	Save(task *Task) error

	// FindByKey finds a task by its key
	FindByKey(key string) (*Task, error)

	// FindByProjectAndSprint finds all tasks for a given project and sprint
	FindByProjectAndSprint(project, sprint string) ([]*Task, error)

	// FindByProject finds all tasks for a given project
	FindByProject(project string) ([]*Task, error)

	// FindBySprint finds all tasks for a given sprint
	FindBySprint(sprint string) ([]*Task, error)

	// FindByPlatform finds all tasks from a specific platform
	FindByPlatform(platform string) ([]*Task, error)

	// FindAll returns all tasks
	FindAll() ([]*Task, error)

	// Delete deletes a task by its key
	Delete(key string) error

	// DeleteByProjectAndSprint deletes all tasks for a given project and sprint
	DeleteByProjectAndSprint(project, sprint string) error
}
