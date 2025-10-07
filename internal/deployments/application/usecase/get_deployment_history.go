package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
)

// GetDeploymentHistoryUseCase handles retrieving deployment history
type GetDeploymentHistoryUseCase struct {
	service *application.DeploymentService
}

// NewGetDeploymentHistoryUseCase creates a new get deployment history use case
func NewGetDeploymentHistoryUseCase(service *application.DeploymentService) *GetDeploymentHistoryUseCase {
	return &GetDeploymentHistoryUseCase{
		service: service,
	}
}

// HistoryFilter contains filter options for deployment history
type HistoryFilter struct {
	TaskKey     string
	AssetName   string
	Environment *domain.Environment
	Limit       int
}

// Execute retrieves deployment history based on filters
func (uc *GetDeploymentHistoryUseCase) Execute(ctx context.Context, filter HistoryFilter) ([]*application.DeploymentWithAssets, error) {
	if uc.service == nil {
		return nil, errors.New("deployment service not configured")
	}

	// Handle different filter types
	if filter.TaskKey != "" {
		deployments, err := uc.service.GetDeploymentsByTask(ctx, filter.TaskKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployments for task: %w", err)
		}

		// Convert to DeploymentWithAssets
		result := make([]*application.DeploymentWithAssets, 0, len(deployments))
		for _, dep := range deployments {
			result = append(result, &application.DeploymentWithAssets{
				Deployment: dep,
			})
		}

		// Apply limit if specified
		if filter.Limit > 0 && len(result) > filter.Limit {
			result = result[:filter.Limit]
		}

		return result, nil
	}

	if filter.AssetName != "" {
		deployments, err := uc.service.GetDeploymentsByAsset(ctx, filter.AssetName)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployments for asset: %w", err)
		}

		// Apply limit if specified
		if filter.Limit > 0 && len(deployments) > filter.Limit {
			deployments = deployments[:filter.Limit]
		}

		return deployments, nil
	}

	// If no specific filter, return all deployments
	allDeployments, err := uc.service.ListAllDeployments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all deployments: %w", err)
	}

	// Convert to DeploymentWithAssets and filter by environment if specified
	result := make([]*application.DeploymentWithAssets, 0)
	for _, dep := range allDeployments {
		if filter.Environment != nil && dep.Environment != *filter.Environment {
			continue
		}
		result = append(result, &application.DeploymentWithAssets{
			Deployment: dep,
		})
	}

	// Apply limit if specified
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, nil
}
