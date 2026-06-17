package ports

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInterpretationError(t *testing.T) {
	t.Parallel()
	err := NewInterpretationError("show me everything", "ambiguous", []string{"assets list", "tasks list"})
	require.NotNil(t, err)
	assert.Equal(t, "show me everything", err.Input)
	assert.Equal(t, "ambiguous", err.Reason)
	assert.Equal(t, []string{"assets list", "tasks list"}, err.Options)
}

func TestInterpretationError_Error(t *testing.T) {
	t.Parallel()
	err := NewInterpretationError("anything", "the reason text", nil)
	assert.Equal(t, "the reason text", err.Error())

	// Also satisfies the standard error interface so errors.As works.
	var ie *InterpretationError
	require.True(t, errors.As(err, &ie))
	assert.Same(t, err, ie)
}
