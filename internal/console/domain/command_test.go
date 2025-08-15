package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommand(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		raw         string
		interpreted string
		confidence  float64
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid command",
			sessionID:   "session-123",
			raw:         "show all assets",
			interpreted: "assets list",
			confidence:  0.95,
			wantErr:     false,
		},
		{
			name:        "empty session ID",
			sessionID:   "",
			raw:         "show all assets",
			interpreted: "assets list",
			confidence:  0.95,
			wantErr:     true,
			errContains: "session ID is required",
		},
		{
			name:        "empty raw input",
			sessionID:   "session-123",
			raw:         "",
			interpreted: "assets list",
			confidence:  0.95,
			wantErr:     true,
			errContains: "raw input is required",
		},
		{
			name:        "empty interpreted command",
			sessionID:   "session-123",
			raw:         "show all assets",
			interpreted: "",
			confidence:  0.95,
			wantErr:     true,
			errContains: "interpreted command is required",
		},
		{
			name:        "confidence too low",
			sessionID:   "session-123",
			raw:         "show all assets",
			interpreted: "assets list",
			confidence:  -0.1,
			wantErr:     true,
			errContains: "confidence must be between 0.0 and 1.0",
		},
		{
			name:        "confidence too high",
			sessionID:   "session-123",
			raw:         "show all assets",
			interpreted: "assets list",
			confidence:  1.1,
			wantErr:     true,
			errContains: "confidence must be between 0.0 and 1.0",
		},
		{
			name:        "minimum confidence",
			sessionID:   "session-123",
			raw:         "show all assets",
			interpreted: "assets list",
			confidence:  0.0,
			wantErr:     false,
		},
		{
			name:        "maximum confidence",
			sessionID:   "session-123",
			raw:         "show all assets",
			interpreted: "assets list",
			confidence:  1.0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := NewCommand(tt.sessionID, tt.raw, tt.interpreted, tt.confidence)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, cmd)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cmd)

				assert.NotEmpty(t, cmd.ID)
				assert.Equal(t, tt.sessionID, cmd.SessionID)
				assert.Equal(t, tt.raw, cmd.Raw)
				assert.Equal(t, tt.interpreted, cmd.Interpreted)
				assert.Equal(t, tt.confidence, cmd.Confidence)
				assert.NotNil(t, cmd.Parameters)
				assert.WithinDuration(t, time.Now(), cmd.Timestamp, time.Second)
			}
		})
	}
}

func TestCommand_Validate(t *testing.T) {
	validCmd := &Command{
		ID:          "cmd-123",
		SessionID:   "session-123",
		Raw:         "show all assets",
		Interpreted: "assets list",
		Confidence:  0.95,
		Timestamp:   time.Now(),
	}

	tests := []struct {
		name        string
		modifyCmd   func(*Command)
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid command",
			modifyCmd: func(_ *Command) {},
			wantErr:   false,
		},
		{
			name:        "empty ID",
			modifyCmd:   func(c *Command) { c.ID = "" },
			wantErr:     true,
			errContains: "command ID is required",
		},
		{
			name:        "empty session ID",
			modifyCmd:   func(c *Command) { c.SessionID = "" },
			wantErr:     true,
			errContains: "session ID is required",
		},
		{
			name:        "empty raw",
			modifyCmd:   func(c *Command) { c.Raw = "" },
			wantErr:     true,
			errContains: "raw input is required",
		},
		{
			name:        "empty interpreted",
			modifyCmd:   func(c *Command) { c.Interpreted = "" },
			wantErr:     true,
			errContains: "interpreted command is required",
		},
		{
			name:        "negative confidence",
			modifyCmd:   func(c *Command) { c.Confidence = -0.1 },
			wantErr:     true,
			errContains: "confidence must be between 0.0 and 1.0",
		},
		{
			name:        "confidence too high",
			modifyCmd:   func(c *Command) { c.Confidence = 1.5 },
			wantErr:     true,
			errContains: "confidence must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := *validCmd // Copy
			tt.modifyCmd(&cmd)

			err := cmd.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCommand_ConfidenceMethods(t *testing.T) {
	tests := []struct {
		name                 string
		confidence           float64
		expectHighConfidence bool
		expectClarification  bool
	}{
		{
			name:                 "very high confidence",
			confidence:           0.95,
			expectHighConfidence: true,
			expectClarification:  false,
		},
		{
			name:                 "high confidence threshold",
			confidence:           0.8,
			expectHighConfidence: true,
			expectClarification:  false,
		},
		{
			name:                 "medium confidence",
			confidence:           0.7,
			expectHighConfidence: false,
			expectClarification:  false,
		},
		{
			name:                 "low confidence threshold",
			confidence:           0.6,
			expectHighConfidence: false,
			expectClarification:  false,
		},
		{
			name:                 "needs clarification",
			confidence:           0.5,
			expectHighConfidence: false,
			expectClarification:  true,
		},
		{
			name:                 "very low confidence",
			confidence:           0.2,
			expectHighConfidence: false,
			expectClarification:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{Confidence: tt.confidence}

			assert.Equal(t, tt.expectHighConfidence, cmd.IsHighConfidence())
			assert.Equal(t, tt.expectClarification, cmd.RequiresClarification())
		})
	}
}

func TestCommand_Parameters(t *testing.T) {
	cmd := &Command{}

	// Test adding parameters
	cmd.AddParameter("name", "Test Asset")
	cmd.AddParameter("type", "software")
	cmd.AddParameter("count", 5)

	// Test getting parameters
	val, exists := cmd.GetParameter("name")
	assert.True(t, exists)
	assert.Equal(t, "Test Asset", val)

	val, exists = cmd.GetParameter("type")
	assert.True(t, exists)
	assert.Equal(t, "software", val)

	val, exists = cmd.GetParameter("count")
	assert.True(t, exists)
	assert.Equal(t, 5, val)

	// Test non-existent parameter
	val, exists = cmd.GetParameter("missing")
	assert.False(t, exists)
	assert.Nil(t, val)

	// Test string parameter helper
	strVal, ok := cmd.GetStringParameter("name")
	assert.True(t, ok)
	assert.Equal(t, "Test Asset", strVal)

	// Test string parameter with non-string value
	strVal, ok = cmd.GetStringParameter("count")
	assert.False(t, ok)
	assert.Empty(t, strVal)

	// Test string parameter with missing key
	strVal, ok = cmd.GetStringParameter("missing")
	assert.False(t, ok)
	assert.Empty(t, strVal)
}

func TestCommand_SetIntent(t *testing.T) {
	cmd := &Command{}

	cmd.SetIntent(CommandTypeCreate, ResourceTypeAsset, "Payment Processing")

	assert.Equal(t, CommandTypeCreate, cmd.Intent.Action)
	assert.Equal(t, ResourceTypeAsset, cmd.Intent.Resource)
	assert.Equal(t, "Payment Processing", cmd.Intent.Target)
}

func TestNewCommandResult(t *testing.T) {
	output := map[string]string{"status": "success"}
	duration := 100 * time.Millisecond

	result := NewCommandResult("cmd-123", true, output, nil, duration)

	assert.Equal(t, "cmd-123", result.CommandID)
	assert.True(t, result.Success)
	assert.Equal(t, output, result.Output)
	assert.Nil(t, result.Error)
	assert.Equal(t, duration, result.Duration)

	// Test with error
	err := assert.AnError
	result = NewCommandResult("cmd-456", false, nil, err, duration)

	assert.Equal(t, "cmd-456", result.CommandID)
	assert.False(t, result.Success)
	assert.Nil(t, result.Output)
	assert.Equal(t, err, result.Error)
	assert.Equal(t, duration, result.Duration)
}
