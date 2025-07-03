package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsConfigurationError(t *testing.T) {
	assert.True(t, IsConfigurationError(ErrMissingBaseURL))
	assert.True(t, IsConfigurationError(ErrMissingEmail))
	assert.True(t, IsConfigurationError(ErrMissingToken))
	assert.True(t, IsConfigurationError(ErrInvalidBaseURL))
	assert.False(t, IsConfigurationError(errors.New("some other error")))
}
