package prompt

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// ConsoleService is the slice of *application.ConsoleService the prompt
// handlers actually depend on. Pulling these four methods out as a
// local interface lets test code stub the service without rebuilding
// the AIInterpreter / CommandExecutor / ContextStore graph behind it,
// and keeps the handlers honest about exactly what they touch.
//
// *application.ConsoleService satisfies this interface implicitly, so
// the public NewHandler / NewEnhancedHandler signatures keep accepting
// the concrete type and no caller has to change.
type ConsoleService interface {
	StartSession(ctx context.Context) (*domain.Context, error)
	ProcessInput(ctx context.Context, sessionID, input string) (*application.ProcessResult, error)
	GetSessionContext(ctx context.Context, sessionID string) (*domain.Context, error)
	EndSession(ctx context.Context, sessionID string) error
}
