package domain

import (
	"errors"
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
	Sprint      string       `json:"sprint"`
	Platform    string       `json:"platform"`
	Status      TaskStatus   `json:"status"`
	Type        TaskType     `json:"type"`
	Priority    TaskPriority `json:"priority"`
	WorkType    WorkType     `json:"work_type"`
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
	return &Task{
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
	}, nil
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
