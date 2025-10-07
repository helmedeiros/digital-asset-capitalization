package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
)

// GetDeploymentsTimelineUseCase handles retrieving deployments in a timeline format
type GetDeploymentsTimelineUseCase struct {
	service *application.DeploymentService
}

// NewGetDeploymentsTimelineUseCase creates a new get deployments timeline use case
func NewGetDeploymentsTimelineUseCase(service *application.DeploymentService) *GetDeploymentsTimelineUseCase {
	return &GetDeploymentsTimelineUseCase{
		service: service,
	}
}

// TimelineInput contains input for timeline retrieval
type TimelineInput struct {
	From          time.Time
	To            time.Time
	Environment   *domain.Environment
	ResolveAssets bool
}

// TimelineOutput contains the timeline result
type TimelineOutput struct {
	Timeline   []application.TimelineEntry       `json:"timeline"`
	Statistics *application.DeploymentStatistics `json:"statistics"`
	Period     string                            `json:"period"`
}

// Execute retrieves deployments in a timeline format
func (uc *GetDeploymentsTimelineUseCase) Execute(ctx context.Context, input TimelineInput) (*TimelineOutput, error) {
	if uc.service == nil {
		return nil, errors.New("deployment service not configured")
	}

	// Validate input
	if input.From.IsZero() || input.To.IsZero() {
		return nil, errors.New("from and to dates are required")
	}

	if input.To.Before(input.From) {
		return nil, errors.New("to date must be after from date")
	}

	// Get timeline
	timeline, err := uc.service.GetDeploymentsTimeline(ctx, input.From, input.To, input.Environment, input.ResolveAssets)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments timeline: %w", err)
	}

	// Get statistics
	stats, err := uc.service.GetDeploymentStatistics(ctx, input.From, input.To)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment statistics: %w", err)
	}

	// Format period string
	period := fmt.Sprintf("%s to %s", input.From.Format("2006-01-02"), input.To.Format("2006-01-02"))

	return &TimelineOutput{
		Timeline:   timeline,
		Statistics: stats,
		Period:     period,
	}, nil
}
