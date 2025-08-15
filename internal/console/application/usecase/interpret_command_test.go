package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

func TestNewInterpretCommandUseCase(t *testing.T) {
	useCase := NewInterpretCommandUseCase(nil)
	require.NotNil(t, useCase)
}

func TestInterpretCommandUseCase_Execute_EmptyInput(t *testing.T) {
	useCase := NewInterpretCommandUseCase(nil)
	ctx := context.Background()
	sessionContext := domain.NewContext("test-session")

	// Test empty input
	cmd, err := useCase.Execute(ctx, "", sessionContext)
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "input cannot be empty")

	// Test whitespace only
	cmd, err = useCase.Execute(ctx, "   ", sessionContext)
	require.Error(t, err)
	assert.Nil(t, cmd)
}

func TestInterpretCommandUseCase_Execute_ExitCommands(t *testing.T) {
	useCase := NewInterpretCommandUseCase(nil)
	ctx := context.Background()
	sessionContext := domain.NewContext("test-session")

	testCases := []string{"exit", "quit", "bye", "EXIT", "QUIT"}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			cmd, err := useCase.Execute(ctx, input, sessionContext)
			require.NoError(t, err)
			require.NotNil(t, cmd)
			assert.Equal(t, input, cmd.Raw)
			assert.Equal(t, "exit", cmd.Interpreted)
			assert.Equal(t, 1.0, cmd.Confidence)
			assert.Equal(t, domain.CommandTypeOther, cmd.Intent.Action)
			assert.Equal(t, domain.ResourceTypeContext, cmd.Intent.Resource)
		})
	}
}

func TestInterpretCommandUseCase_Execute_HelpCommands(t *testing.T) {
	useCase := NewInterpretCommandUseCase(nil)
	ctx := context.Background()
	sessionContext := domain.NewContext("test-session")

	testCases := []string{"help", "?", "HELP"}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			cmd, err := useCase.Execute(ctx, input, sessionContext)
			require.NoError(t, err)
			require.NotNil(t, cmd)
			assert.Equal(t, input, cmd.Raw)
			assert.Equal(t, "help", cmd.Interpreted)
			assert.Equal(t, 1.0, cmd.Confidence)
			assert.Equal(t, domain.CommandTypeHelp, cmd.Intent.Action)
		})
	}
}
