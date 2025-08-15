package usecase

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// MaintainContextUseCase handles context management operations
type MaintainContextUseCase struct {
	contextStore ports.ContextStore
}

// NewMaintainContextUseCase creates a new maintain context use case
func NewMaintainContextUseCase(contextStore ports.ContextStore) *MaintainContextUseCase {
	return &MaintainContextUseCase{
		contextStore: contextStore,
	}
}

// UpdateContext updates the context after command execution
func (uc *MaintainContextUseCase) UpdateContext(
	ctx context.Context,
	sessionID string,
	command *domain.Command,
	result *domain.CommandResult,
) error {
	return uc.contextStore.Update(ctx, sessionID, func(sessionContext *domain.Context) error {
		// Add command to history
		sessionContext.AddCommand(*command)

		// Add result to history
		sessionContext.AddCommandResult(*result)

		return nil
	})
}
