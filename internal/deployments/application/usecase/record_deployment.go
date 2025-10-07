package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
)

// RecordDeploymentUseCase handles recording a new deployment
type RecordDeploymentUseCase struct {
	service *application.DeploymentService
}

// NewRecordDeploymentUseCase creates a new record deployment use case
func NewRecordDeploymentUseCase(service *application.DeploymentService) *RecordDeploymentUseCase {
	return &RecordDeploymentUseCase{
		service: service,
	}
}

// Execute records a new deployment
func (uc *RecordDeploymentUseCase) Execute(ctx context.Context, input application.RecordDeploymentInput) (*domain.Deployment, error) {
	if uc.service == nil {
		return nil, errors.New("deployment service not configured")
	}

	// Validate input
	if len(input.TaskKeys) == 0 {
		return nil, errors.New("at least one task key is required")
	}

	if input.Environment == "" {
		return nil, errors.New("environment is required")
	}

	if input.Version == "" {
		return nil, errors.New("version is required")
	}

	// Record the deployment
	deployment, err := uc.service.RecordDeployment(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to record deployment: %w", err)
	}

	return deployment, nil
}
