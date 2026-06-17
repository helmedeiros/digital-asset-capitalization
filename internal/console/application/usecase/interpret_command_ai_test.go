package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// stubAIInterpreter is the minimum AIInterpreter surface needed for
// these tests. The use case only calls Interpret on the
// Execute non-special-command path, so the other two methods just
// panic to flag accidental use in future tests.
type stubAIInterpreter struct {
	interpretFn func(ctx context.Context, input string, sessionContext *domain.Context) (*domain.Command, error)
}

func (s *stubAIInterpreter) Interpret(ctx context.Context, input string, sc *domain.Context) (*domain.Command, error) {
	return s.interpretFn(ctx, input, sc)
}
func (s *stubAIInterpreter) GetClarification(context.Context, string, []string) (string, error) {
	panic("GetClarification should not be called by these tests")
}
func (s *stubAIInterpreter) AnalyzeIntent(context.Context, string) (*domain.CommandIntent, error) {
	panic("AnalyzeIntent should not be called by these tests")
}

func TestInterpretCommandUseCase_Execute_DelegatesToInterpreter(t *testing.T) {
	t.Parallel()
	wantCmd, _ := domain.NewCommand("s", "Show all assets", "list", 0.9)
	wantCmd.SetIntent(domain.CommandTypeList, domain.ResourceTypeAsset, "")

	captured := struct {
		input string
		ctx   *domain.Context
	}{}
	interpreter := &stubAIInterpreter{
		interpretFn: func(_ context.Context, input string, sc *domain.Context) (*domain.Command, error) {
			captured.input = input
			captured.ctx = sc
			return wantCmd, nil
		},
	}
	uc := NewInterpretCommandUseCase(interpreter)
	sc := domain.NewContext("s")

	cmd, err := uc.Execute(context.Background(), "  Show all assets  ", sc)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Same(t, wantCmd, cmd)

	// The use case must trim whitespace before delegating; the AI gets
	// the cleaned-up input, not the raw "  Show all assets  ".
	assert.Equal(t, "Show all assets", captured.input)
	assert.Same(t, sc, captured.ctx)
}

func TestInterpretCommandUseCase_Execute_PropagatesInterpreterError(t *testing.T) {
	t.Parallel()
	interpreter := &stubAIInterpreter{
		interpretFn: func(context.Context, string, *domain.Context) (*domain.Command, error) {
			return nil, errors.New("model unavailable")
		},
	}
	uc := NewInterpretCommandUseCase(interpreter)
	sc := domain.NewContext("s")

	cmd, err := uc.Execute(context.Background(), "list everything", sc)
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Equal(t, "model unavailable", err.Error())
}

func TestInterpretCommandUseCase_Execute_SpecialCommandsBypassInterpreter(t *testing.T) {
	t.Parallel()
	// If the special-command handler returns non-nil, Interpret must
	// not be called. We use an interpreter whose Interpret panics to
	// guarantee that. "bye" is the special-command that's not yet
	// exercised by the existing exit/help table tests across cases.
	interpreter := &stubAIInterpreter{
		interpretFn: func(context.Context, string, *domain.Context) (*domain.Command, error) {
			panic("Interpret must not be called for special commands")
		},
	}
	uc := NewInterpretCommandUseCase(interpreter)
	sc := domain.NewContext("s")

	cmd, err := uc.Execute(context.Background(), "bye", sc)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "exit", cmd.Interpreted)
}
