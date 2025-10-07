package domain

import (
	"errors"
	"fmt"
	"time"
)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusSuccessful DeploymentStatus = "successful"
	DeploymentStatusFailed     DeploymentStatus = "failed"
	DeploymentStatusRolledBack DeploymentStatus = "rolled_back"
	DeploymentStatusInProgress DeploymentStatus = "in_progress"
)

// Environment represents the deployment environment
type Environment string

const (
	EnvironmentProduction  Environment = "production"
	EnvironmentStaging     Environment = "staging"
	EnvironmentDevelopment Environment = "development"
	EnvironmentQA          Environment = "qa"
)

// DeploymentMetadata contains additional deployment information
type DeploymentMetadata struct {
	PipelineID                string  `json:"pipeline_id,omitempty"`
	PipelineURL               string  `json:"pipeline_url,omitempty"`
	DeploymentDurationSeconds int     `json:"deployment_duration_seconds,omitempty"`
	RollbackFrom              *string `json:"rollback_from,omitempty"`
	TriggeredBy               string  `json:"triggered_by,omitempty"`
	Notes                     string  `json:"notes,omitempty"`
}

// Deployment represents a deployment event
type Deployment struct {
	ID          string              `json:"id"`
	TaskKeys    []string            `json:"task_keys"`
	Environment Environment         `json:"environment"`
	DeployedAt  time.Time           `json:"deployed_at"`
	DeployedBy  string              `json:"deployed_by"`
	Version     string              `json:"version"`
	CommitSHA   string              `json:"commit_sha"`
	Status      DeploymentStatus    `json:"status"`
	Metadata    *DeploymentMetadata `json:"metadata,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// NewDeployment creates a new deployment with validation
func NewDeployment(taskKeys []string, environment Environment, version string) (*Deployment, error) {
	if len(taskKeys) == 0 {
		return nil, errors.New("deployment must have at least one task key")
	}

	if environment == "" {
		return nil, errors.New("environment is required")
	}

	if version == "" {
		return nil, errors.New("version is required")
	}

	now := time.Now()
	id := generateDeploymentID(now)

	return &Deployment{
		ID:          id,
		TaskKeys:    taskKeys,
		Environment: environment,
		DeployedAt:  now,
		Version:     version,
		Status:      DeploymentStatusInProgress,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// generateDeploymentID generates a unique deployment ID
func generateDeploymentID(t time.Time) string {
	// Format: dep-YYYYMMDD-HHMMSS
	return fmt.Sprintf("dep-%s-%s",
		t.Format("20060102"),
		t.Format("150405"))
}

// SetStatus updates the deployment status
func (d *Deployment) SetStatus(status DeploymentStatus) {
	d.Status = status
	d.UpdatedAt = time.Now()
}

// SetDeployedBy sets who deployed this release
func (d *Deployment) SetDeployedBy(deployedBy string) {
	d.DeployedBy = deployedBy
	d.UpdatedAt = time.Now()
}

// SetCommitSHA sets the commit SHA for the deployment
func (d *Deployment) SetCommitSHA(sha string) {
	d.CommitSHA = sha
	d.UpdatedAt = time.Now()
}

// SetMetadata sets the deployment metadata
func (d *Deployment) SetMetadata(metadata *DeploymentMetadata) {
	d.Metadata = metadata
	d.UpdatedAt = time.Now()
}

// AddTaskKey adds a task key to the deployment
func (d *Deployment) AddTaskKey(key string) error {
	if key == "" {
		return errors.New("task key cannot be empty")
	}

	// Check for duplicates
	for _, existingKey := range d.TaskKeys {
		if existingKey == key {
			return fmt.Errorf("task key %s already exists in deployment", key)
		}
	}

	d.TaskKeys = append(d.TaskKeys, key)
	d.UpdatedAt = time.Now()
	return nil
}

// IsProduction checks if this is a production deployment
func (d *Deployment) IsProduction() bool {
	return d.Environment == EnvironmentProduction
}

// IsSuccessful checks if the deployment was successful
func (d *Deployment) IsSuccessful() bool {
	return d.Status == DeploymentStatusSuccessful
}

// Validate validates the deployment
func (d *Deployment) Validate() error {
	if d.ID == "" {
		return errors.New("deployment ID is required")
	}

	if len(d.TaskKeys) == 0 {
		return errors.New("deployment must have at least one task key")
	}

	if d.Environment == "" {
		return errors.New("environment is required")
	}

	if d.Version == "" {
		return errors.New("version is required")
	}

	if d.DeployedAt.IsZero() {
		return errors.New("deployed_at timestamp is required")
	}

	return nil
}

// TimeRange represents a time range for filtering deployments
type TimeRange struct {
	From time.Time
	To   time.Time
}

// Contains checks if a given time is within the range
func (tr TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.From) && !t.After(tr.To)
}

// IsValid checks if the time range is valid
func (tr TimeRange) IsValid() bool {
	return !tr.From.IsZero() && !tr.To.IsZero() && !tr.To.Before(tr.From)
}
