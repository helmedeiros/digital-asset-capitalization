package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
)

// JSONRepository implements DeploymentRepository using JSON files
type JSONRepository struct {
	dir      string
	filename string
	mu       sync.RWMutex
}

// JSONRepositoryConfig holds configuration for the JSON repository
type JSONRepositoryConfig struct {
	Directory string
	Filename  string
}

// DefaultJSONRepositoryConfig returns a default configuration
func DefaultJSONRepositoryConfig() JSONRepositoryConfig {
	return JSONRepositoryConfig{
		Directory: ".assetcap",
		Filename:  "deployments.json",
	}
}

// NewJSONRepository creates a new JSON repository
func NewJSONRepository(config JSONRepositoryConfig) ports.DeploymentRepository {
	if config.Directory == "" {
		config.Directory = ".assetcap"
	}
	if config.Filename == "" {
		config.Filename = "deployments.json"
	}

	return &JSONRepository{
		dir:      config.Directory,
		filename: config.Filename,
	}
}

// getFilePath returns the full path to the JSON file
func (r *JSONRepository) getFilePath() string {
	return filepath.Join(r.dir, r.filename)
}

// ensureDirectory creates the directory if it doesn't exist
func (r *JSONRepository) ensureDirectory() error {
	return os.MkdirAll(r.dir, 0755)
}

// loadDeployments loads all deployments from the JSON file
func (r *JSONRepository) loadDeployments() (map[string]*domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deployments := make(map[string]*domain.Deployment)

	filePath := r.getFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty map
			return deployments, nil
		}
		return nil, fmt.Errorf("failed to read deployments file: %w", err)
	}

	if len(data) == 0 {
		return deployments, nil
	}

	if err := json.Unmarshal(data, &deployments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployments: %w", err)
	}

	return deployments, nil
}

// saveDeployments saves all deployments to the JSON file
func (r *JSONRepository) saveDeployments(deployments map[string]*domain.Deployment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureDirectory(); err != nil {
		return fmt.Errorf("failed to ensure directory: %w", err)
	}

	data, err := json.MarshalIndent(deployments, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal deployments: %w", err)
	}

	filePath := r.getFilePath()
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write deployments file: %w", err)
	}

	return nil
}

// Save persists a deployment
func (r *JSONRepository) Save(_ context.Context, deployment *domain.Deployment) error {
	if deployment == nil {
		return errors.New("deployment cannot be nil")
	}

	if err := deployment.Validate(); err != nil {
		return fmt.Errorf("invalid deployment: %w", err)
	}

	deployments, err := r.loadDeployments()
	if err != nil {
		return err
	}

	deployments[deployment.ID] = deployment

	return r.saveDeployments(deployments)
}

// FindByID retrieves a deployment by its ID
func (r *JSONRepository) FindByID(_ context.Context, id string) (*domain.Deployment, error) {
	if id == "" {
		return nil, errors.New("deployment ID cannot be empty")
	}

	deployments, err := r.loadDeployments()
	if err != nil {
		return nil, err
	}

	deployment, exists := deployments[id]
	if !exists {
		return nil, fmt.Errorf("deployment with ID %s not found", id)
	}

	return deployment, nil
}

// FindByTaskKey retrieves all deployments containing the specified task key
func (r *JSONRepository) FindByTaskKey(_ context.Context, taskKey string) ([]*domain.Deployment, error) {
	if taskKey == "" {
		return nil, errors.New("task key cannot be empty")
	}

	deployments, err := r.loadDeployments()
	if err != nil {
		return nil, err
	}

	var result []*domain.Deployment
	for _, deployment := range deployments {
		for _, key := range deployment.TaskKeys {
			if key == taskKey {
				result = append(result, deployment)
				break
			}
		}
	}

	return result, nil
}

// FindByTimeRange retrieves all deployments within the specified time range
func (r *JSONRepository) FindByTimeRange(_ context.Context, timeRange domain.TimeRange) ([]*domain.Deployment, error) {
	if !timeRange.IsValid() {
		return nil, errors.New("invalid time range")
	}

	deployments, err := r.loadDeployments()
	if err != nil {
		return nil, err
	}

	var result []*domain.Deployment
	for _, deployment := range deployments {
		if timeRange.Contains(deployment.DeployedAt) {
			result = append(result, deployment)
		}
	}

	return result, nil
}

// FindByEnvironmentAndTimeRange retrieves deployments for a specific environment within a time range
func (r *JSONRepository) FindByEnvironmentAndTimeRange(_ context.Context, environment domain.Environment, timeRange domain.TimeRange) ([]*domain.Deployment, error) {
	if environment == "" {
		return nil, errors.New("environment cannot be empty")
	}

	if !timeRange.IsValid() {
		return nil, errors.New("invalid time range")
	}

	deployments, err := r.loadDeployments()
	if err != nil {
		return nil, err
	}

	var result []*domain.Deployment
	for _, deployment := range deployments {
		if deployment.Environment == environment && timeRange.Contains(deployment.DeployedAt) {
			result = append(result, deployment)
		}
	}

	return result, nil
}

// ListAll retrieves all deployments
func (r *JSONRepository) ListAll(_ context.Context) ([]*domain.Deployment, error) {
	deployments, err := r.loadDeployments()
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Deployment, 0, len(deployments))
	for _, deployment := range deployments {
		result = append(result, deployment)
	}

	return result, nil
}

// Update updates an existing deployment
func (r *JSONRepository) Update(_ context.Context, deployment *domain.Deployment) error {
	if deployment == nil {
		return errors.New("deployment cannot be nil")
	}

	if err := deployment.Validate(); err != nil {
		return fmt.Errorf("invalid deployment: %w", err)
	}

	deployments, err := r.loadDeployments()
	if err != nil {
		return err
	}

	if _, exists := deployments[deployment.ID]; !exists {
		return fmt.Errorf("deployment with ID %s not found", deployment.ID)
	}

	deployments[deployment.ID] = deployment

	return r.saveDeployments(deployments)
}

// Delete removes a deployment by ID
func (r *JSONRepository) Delete(_ context.Context, id string) error {
	if id == "" {
		return errors.New("deployment ID cannot be empty")
	}

	deployments, err := r.loadDeployments()
	if err != nil {
		return err
	}

	if _, exists := deployments[id]; !exists {
		return fmt.Errorf("deployment with ID %s not found", id)
	}

	delete(deployments, id)

	return r.saveDeployments(deployments)
}

// Count returns the total number of deployments
func (r *JSONRepository) Count(_ context.Context) (int, error) {
	deployments, err := r.loadDeployments()
	if err != nil {
		return 0, err
	}

	return len(deployments), nil
}
