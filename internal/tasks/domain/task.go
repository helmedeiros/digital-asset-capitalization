package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyKey        = errors.New("task key cannot be empty")
	ErrEmptySummary    = errors.New("task summary cannot be empty")
	ErrEmptyProject    = errors.New("task project cannot be empty")
	ErrEmptySprint     = errors.New("task sprint cannot be empty")
	ErrEmptyPlatform   = errors.New("task platform cannot be empty")
	ErrInvalidStatus   = errors.New("invalid task status")
	ErrInvalidType     = errors.New("invalid task type")
	ErrInvalidPriority = errors.New("invalid task priority")
	ErrInvalidWorkType = errors.New("invalid work type")
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "TODO"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusDone       TaskStatus = "DONE"
	TaskStatusBlocked    TaskStatus = "BLOCKED"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeStory   TaskType = "STORY"
	TaskTypeTask    TaskType = "TASK"
	TaskTypeBug     TaskType = "BUG"
	TaskTypeEpic    TaskType = "EPIC"
	TaskTypeSubtask TaskType = "SUBTASK"
)

// TaskPriority represents the priority level of a task
type TaskPriority string

const (
	TaskPriorityHighest TaskPriority = "HIGHEST"
	TaskPriorityHigh    TaskPriority = "HIGH"
	TaskPriorityMedium  TaskPriority = "MEDIUM"
	TaskPriorityLow     TaskPriority = "LOW"
	TaskPriorityLowest  TaskPriority = "LOWEST"
)

// WorkType represents the type of work being done in a task
type WorkType string

const (
	WorkTypeMaintenance WorkType = "cap-maintenance"
	WorkTypeDiscovery   WorkType = "cap-discovery"
	WorkTypeDevelopment WorkType = "cap-development"
)

// Task represents a task from a project management platform
type Task struct {
	Key         string       `json:"key"`
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	Project     string       `json:"project"`
	Sprint      string       `json:"sprint"`            // Legacy field for backward compatibility
	Sprints     []string     `json:"sprints,omitempty"` // New array field for multi-sprint support
	Platform    string       `json:"platform"`
	Status      TaskStatus   `json:"status"`
	Type        TaskType     `json:"type"`
	Priority    TaskPriority `json:"priority"`
	WorkType    WorkType     `json:"work_type"`
	Labels      []string     `json:"labels"`
	Epic        string       `json:"epic"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Version     int          `json:"version"`
}

// NewTask creates a new task with the given parameters
func NewTask(key, summary, project, sprint, platform string) (*Task, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}
	if summary == "" {
		return nil, ErrEmptySummary
	}
	if project == "" {
		return nil, ErrEmptyProject
	}
	if sprint == "" {
		return nil, ErrEmptySprint
	}
	if platform == "" {
		return nil, ErrEmptyPlatform
	}

	now := time.Now()
	task := &Task{
		Key:       key,
		Summary:   summary,
		Project:   project,
		Sprint:    sprint,
		Platform:  platform,
		Status:    TaskStatusTodo,
		Type:      TaskTypeTask,
		Priority:  TaskPriorityMedium,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	// Set sprints array based on sprint string
	task.setSprintsFromString(sprint)

	return task, nil
}

// NewTaskWithSprints creates a new task with multiple sprints
func NewTaskWithSprints(key, summary, project string, sprints []string, platform string) (*Task, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}
	if summary == "" {
		return nil, ErrEmptySummary
	}
	if project == "" {
		return nil, ErrEmptyProject
	}
	if len(sprints) == 0 {
		return nil, ErrEmptySprint
	}
	if platform == "" {
		return nil, ErrEmptyPlatform
	}

	now := time.Now()
	task := &Task{
		Key:       key,
		Summary:   summary,
		Project:   project,
		Sprints:   sprints,
		Platform:  platform,
		Status:    TaskStatusTodo,
		Type:      TaskTypeTask,
		Priority:  TaskPriorityMedium,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	// Set legacy sprint field for backward compatibility
	task.Sprint = strings.Join(sprints, ", ")

	return task, nil
}

// NewTaskWithoutSprint creates a new task without sprint assignment
func NewTaskWithoutSprint(key, summary, project, platform string) (*Task, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}
	if summary == "" {
		return nil, ErrEmptySummary
	}
	if project == "" {
		return nil, ErrEmptyProject
	}
	if platform == "" {
		return nil, ErrEmptyPlatform
	}

	now := time.Now()
	task := &Task{
		Key:       key,
		Summary:   summary,
		Project:   project,
		Sprint:    "",         // Empty sprint
		Sprints:   []string{}, // Empty sprints array
		Platform:  platform,
		Status:    TaskStatusTodo,
		Type:      TaskTypeTask,
		Priority:  TaskPriorityMedium,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	return task, nil
}

// GetSprints returns the sprint names as an array
func (t *Task) GetSprints() []string {
	// If we have the new sprints array, use it
	if len(t.Sprints) > 0 {
		return t.Sprints
	}

	// Fall back to parsing the legacy sprint string
	if t.Sprint != "" {
		return t.parseSprintString(t.Sprint)
	}

	return []string{}
}

// GetPrimarySprint returns the first/primary sprint
func (t *Task) GetPrimarySprint() string {
	sprints := t.GetSprints()
	if len(sprints) > 0 {
		return sprints[0]
	}
	return ""
}

// HasSprint checks if the task belongs to a specific sprint
func (t *Task) HasSprint(sprintName string) bool {
	sprints := t.GetSprints()
	for _, sprint := range sprints {
		if sprint == sprintName {
			return true
		}
	}
	return false
}

// SetSprints updates the task sprints
func (t *Task) SetSprints(sprints []string) {
	t.Sprints = sprints
	t.Sprint = strings.Join(sprints, ", ") // Maintain backward compatibility
	t.UpdatedAt = time.Now()
	t.Version++
}

// AddSprint adds a sprint to the task if it doesn't already exist
func (t *Task) AddSprint(sprintName string) {
	if !t.HasSprint(sprintName) {
		sprints := t.GetSprints()
		sprints = append(sprints, sprintName)
		t.SetSprints(sprints)
	}
}

// RemoveSprint removes a sprint from the task
func (t *Task) RemoveSprint(sprintName string) {
	sprints := t.GetSprints()
	var newSprints []string
	for _, sprint := range sprints {
		if sprint != sprintName {
			newSprints = append(newSprints, sprint)
		}
	}
	if len(newSprints) != len(sprints) {
		t.SetSprints(newSprints)
	}
}

// setSprintsFromString parses sprint string and sets the sprints array
func (t *Task) setSprintsFromString(sprintStr string) {
	if sprintStr != "" {
		t.Sprints = t.parseSprintString(sprintStr)
	}
}

// parseSprintString parses a comma-separated sprint string into an array
func (t *Task) parseSprintString(sprintStr string) []string {
	if sprintStr == "" {
		return []string{}
	}

	// Split by comma and trim whitespace
	parts := strings.Split(sprintStr, ",")
	var sprints []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			sprints = append(sprints, trimmed)
		}
	}
	return sprints
}

// MarshalJSON handles custom JSON marshaling to ensure data consistency
func (t *Task) MarshalJSON() ([]byte, error) {
	// Ensure consistency between Sprint and Sprints fields
	if len(t.Sprints) > 0 && t.Sprint == "" {
		t.Sprint = strings.Join(t.Sprints, ", ")
	} else if t.Sprint != "" && len(t.Sprints) == 0 {
		t.Sprints = t.parseSprintString(t.Sprint)
	}

	// Create a copy of the struct for marshaling
	type TaskAlias Task
	return json.Marshal((*TaskAlias)(t))
}

// UnmarshalJSON handles custom JSON unmarshaling to ensure data consistency
func (t *Task) UnmarshalJSON(data []byte) error {
	// Use alias to avoid infinite recursion
	type TaskAlias Task
	aux := &TaskAlias{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	*t = Task(*aux)

	// Ensure consistency between Sprint and Sprints fields
	if len(t.Sprints) > 0 {
		// If we have sprints array, ensure Sprint field is consistent
		t.Sprint = strings.Join(t.Sprints, ", ")
	} else if t.Sprint != "" {
		// If we only have Sprint field, populate Sprints array
		t.Sprints = t.parseSprintString(t.Sprint)
	}

	return nil
}

// UpdateStatus updates the task status
func (t *Task) UpdateStatus(status TaskStatus) error {
	switch status {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone, TaskStatusBlocked:
		t.Status = status
		t.UpdatedAt = time.Now()
		t.Version++
		return nil
	default:
		return ErrInvalidStatus
	}
}

// UpdateType updates the task type
func (t *Task) UpdateType(taskType TaskType) error {
	switch taskType {
	case TaskTypeStory, TaskTypeTask, TaskTypeBug, TaskTypeEpic, TaskTypeSubtask:
		t.Type = taskType
		t.UpdatedAt = time.Now()
		t.Version++
		return nil
	default:
		return ErrInvalidType
	}
}

// UpdatePriority updates the task priority
func (t *Task) UpdatePriority(priority TaskPriority) error {
	switch priority {
	case TaskPriorityHighest, TaskPriorityHigh, TaskPriorityMedium, TaskPriorityLow, TaskPriorityLowest:
		t.Priority = priority
		t.UpdatedAt = time.Now()
		t.Version++
		return nil
	default:
		return ErrInvalidPriority
	}
}

// UpdateDescription updates the task description
func (t *Task) UpdateDescription(description string) {
	t.Description = description
	t.UpdatedAt = time.Now()
	t.Version++
}

// UpdateWorkType updates the task work type
func (t *Task) UpdateWorkType(workType WorkType) error {
	switch workType {
	case WorkTypeMaintenance, WorkTypeDiscovery, WorkTypeDevelopment:
		t.WorkType = workType
		t.UpdatedAt = time.Now()
		t.Version++
		return nil
	default:
		return ErrInvalidWorkType
	}
}

// IsDone returns true if the task is in DONE status
func (t *Task) IsDone() bool {
	return t.Status == TaskStatusDone
}

// IsInProgress returns true if the task is in IN_PROGRESS status
func (t *Task) IsInProgress() bool {
	return t.Status == TaskStatusInProgress
}

// IsBlocked returns true if the task is in BLOCKED status
func (t *Task) IsBlocked() bool {
	return t.Status == TaskStatusBlocked
}
