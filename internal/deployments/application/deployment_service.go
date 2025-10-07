package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
)

// DeploymentService provides business logic for deployments
type DeploymentService struct {
	repo          ports.DeploymentRepository
	assetResolver ports.AssetResolver
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(repo ports.DeploymentRepository, assetResolver ports.AssetResolver) *DeploymentService {
	return &DeploymentService{
		repo:          repo,
		assetResolver: assetResolver,
	}
}

// RecordDeploymentInput contains input for recording a deployment
type RecordDeploymentInput struct {
	TaskKeys    []string
	Environment domain.Environment
	Version     string
	DeployedBy  string
	CommitSHA   string
	Metadata    *domain.DeploymentMetadata
}

// RecordDeployment records a new deployment
func (s *DeploymentService) RecordDeployment(ctx context.Context, input RecordDeploymentInput) (*domain.Deployment, error) {
	if len(input.TaskKeys) == 0 {
		return nil, errors.New("at least one task key is required")
	}

	deployment, err := domain.NewDeployment(input.TaskKeys, input.Environment, input.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	if input.DeployedBy != "" {
		deployment.SetDeployedBy(input.DeployedBy)
	}

	if input.CommitSHA != "" {
		deployment.SetCommitSHA(input.CommitSHA)
	}

	if input.Metadata != nil {
		deployment.SetMetadata(input.Metadata)
	}

	// Mark as successful by default (can be updated later)
	deployment.SetStatus(domain.DeploymentStatusSuccessful)

	if err := s.repo.Save(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	return deployment, nil
}

// UpdateDeploymentStatus updates the status of a deployment
func (s *DeploymentService) UpdateDeploymentStatus(ctx context.Context, deploymentID string, status domain.DeploymentStatus) error {
	deployment, err := s.repo.FindByID(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to find deployment: %w", err)
	}

	deployment.SetStatus(status)

	if err := s.repo.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return nil
}

// GetDeploymentByID retrieves a deployment by ID
func (s *DeploymentService) GetDeploymentByID(ctx context.Context, id string) (*domain.Deployment, error) {
	return s.repo.FindByID(ctx, id)
}

// GetDeploymentsByTask retrieves all deployments for a specific task
func (s *DeploymentService) GetDeploymentsByTask(ctx context.Context, taskKey string) ([]*domain.Deployment, error) {
	deployments, err := s.repo.FindByTaskKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find deployments for task %s: %w", taskKey, err)
	}

	// Sort by deployed_at descending
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].DeployedAt.After(deployments[j].DeployedAt)
	})

	return deployments, nil
}

// DeploymentWithAssets represents a deployment with resolved assets
type DeploymentWithAssets struct {
	*domain.Deployment
	ResolvedAssets []ports.AssetInfo `json:"resolved_assets,omitempty"`
}

// GetDeploymentsByAsset retrieves all deployments affecting a specific asset
func (s *DeploymentService) GetDeploymentsByAsset(ctx context.Context, assetName string) ([]*DeploymentWithAssets, error) {
	if s.assetResolver == nil {
		return nil, errors.New("asset resolver not configured")
	}

	// Get all deployments
	allDeployments, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var result []*DeploymentWithAssets

	// Check each deployment to see if it affects the asset
	for _, deployment := range allDeployments {
		assets, err := s.assetResolver.ResolveAssetsForTasks(ctx, deployment.TaskKeys)
		if err != nil {
			// Continue even if resolution fails for some deployments
			continue
		}

		// Check if the asset is affected
		for _, asset := range assets {
			if asset.Name == assetName {
				result = append(result, &DeploymentWithAssets{
					Deployment:     deployment,
					ResolvedAssets: assets,
				})
				break
			}
		}
	}

	// Sort by deployed_at descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].DeployedAt.After(result[j].DeployedAt)
	})

	return result, nil
}

// TimelineEntry represents a deployment in the timeline view
type TimelineEntry struct {
	Date        time.Time
	Deployments []*DeploymentWithAssets
}

// GetDeploymentsTimeline retrieves deployments in a timeline format
func (s *DeploymentService) GetDeploymentsTimeline(ctx context.Context, from, to time.Time, environment *domain.Environment, resolveAssets bool) ([]TimelineEntry, error) {
	timeRange := domain.TimeRange{
		From: from,
		To:   to,
	}

	if !timeRange.IsValid() {
		return nil, errors.New("invalid time range")
	}

	var deployments []*domain.Deployment
	var err error

	if environment != nil {
		deployments, err = s.repo.FindByEnvironmentAndTimeRange(ctx, *environment, timeRange)
	} else {
		deployments, err = s.repo.FindByTimeRange(ctx, timeRange)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find deployments: %w", err)
	}

	// Group by date
	dateMap := make(map[string][]*DeploymentWithAssets)

	for _, deployment := range deployments {
		dateKey := deployment.DeployedAt.Format("2006-01-02")

		deploymentWithAssets := &DeploymentWithAssets{
			Deployment: deployment,
		}

		// Resolve assets if requested
		if resolveAssets && s.assetResolver != nil {
			assets, err := s.assetResolver.ResolveAssetsForTasks(ctx, deployment.TaskKeys)
			if err == nil {
				deploymentWithAssets.ResolvedAssets = assets
			}
		}

		dateMap[dateKey] = append(dateMap[dateKey], deploymentWithAssets)
	}

	// Convert to timeline entries
	timeline := make([]TimelineEntry, 0, len(dateMap))
	for dateStr, deps := range dateMap {
		date, _ := time.Parse("2006-01-02", dateStr)

		// Sort deployments within each day by time
		sort.Slice(deps, func(i, j int) bool {
			return deps[i].DeployedAt.Before(deps[j].DeployedAt)
		})

		timeline = append(timeline, TimelineEntry{
			Date:        date,
			Deployments: deps,
		})
	}

	// Sort timeline by date
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Date.Before(timeline[j].Date)
	})

	return timeline, nil
}

// GetDeploymentStatistics returns statistics for deployments in a time range
func (s *DeploymentService) GetDeploymentStatistics(ctx context.Context, from, to time.Time) (*DeploymentStatistics, error) {
	timeRange := domain.TimeRange{
		From: from,
		To:   to,
	}

	deployments, err := s.repo.FindByTimeRange(ctx, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to find deployments: %w", err)
	}

	stats := &DeploymentStatistics{
		TotalDeployments: len(deployments),
		ByEnvironment:    make(map[string]int),
		ByStatus:         make(map[string]int),
		UniqueTaskKeys:   make(map[string]bool),
	}

	for _, deployment := range deployments {
		// Count by environment
		stats.ByEnvironment[string(deployment.Environment)]++

		// Count by status
		stats.ByStatus[string(deployment.Status)]++

		// Collect unique task keys
		for _, taskKey := range deployment.TaskKeys {
			stats.UniqueTaskKeys[taskKey] = true
		}
	}

	return stats, nil
}

// DeploymentStatistics contains deployment statistics
type DeploymentStatistics struct {
	TotalDeployments int             `json:"total_deployments"`
	ByEnvironment    map[string]int  `json:"by_environment"`
	ByStatus         map[string]int  `json:"by_status"`
	UniqueTaskKeys   map[string]bool `json:"unique_task_keys"`
}

// ListAllDeployments lists all deployments
func (s *DeploymentService) ListAllDeployments(ctx context.Context) ([]*domain.Deployment, error) {
	deployments, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	// Sort by deployed_at descending
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].DeployedAt.After(deployments[j].DeployedAt)
	})

	return deployments, nil
}
