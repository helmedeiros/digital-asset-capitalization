package domain

import (
	"errors"
	"fmt"
	"time"
)

// CommandType represents the type of action being performed
type CommandType string

const (
	CommandTypeCreate CommandType = "create"
	CommandTypeRead   CommandType = "read"
	CommandTypeUpdate CommandType = "update"
	CommandTypeDelete CommandType = "delete"
	CommandTypeList   CommandType = "list"
	CommandTypeSync   CommandType = "sync"
	CommandTypeHelp   CommandType = "help"
	CommandTypeOther  CommandType = "other"
)

// ResourceType represents the resource being operated on
type ResourceType string

const (
	ResourceTypeAsset      ResourceType = "asset"
	ResourceTypeTask       ResourceType = "task"
	ResourceTypeSprint     ResourceType = "sprint"
	ResourceTypeInvestment ResourceType = "investment"
	ResourceTypeConfig     ResourceType = "config"
	ResourceTypeContext    ResourceType = "context"
	ResourceTypeUnknown    ResourceType = "unknown"
)

// Command represents a parsed command from user input
type Command struct {
	ID          string                 // Unique identifier for the command
	Raw         string                 // Original user input
	Interpreted string                 // Parsed CLI command
	Intent      CommandIntent          // Structured intent
	Parameters  map[string]interface{} // Command parameters
	Confidence  float64                // AI confidence score (0.0-1.0)
	Timestamp   time.Time              // When the command was created
	SessionID   string                 // Associated session
}

// CommandIntent represents the structured intent of a command
type CommandIntent struct {
	Action    CommandType  // The action to perform
	Resource  ResourceType // The resource to act on
	Modifiers []string     // Additional modifiers (e.g., "all", "detailed")
	Target    string       // Specific target (e.g., asset name)
}

// NewCommand creates a new command with validation
func NewCommand(sessionID, raw, interpreted string, confidence float64) (*Command, error) {
	if sessionID == "" {
		return nil, errors.New("session ID is required")
	}
	if raw == "" {
		return nil, errors.New("raw input is required")
	}
	if interpreted == "" {
		return nil, errors.New("interpreted command is required")
	}
	if confidence < 0.0 || confidence > 1.0 {
		return nil, fmt.Errorf("confidence must be between 0.0 and 1.0, got %f", confidence)
	}

	return &Command{
		ID:          generateCommandID(),
		Raw:         raw,
		Interpreted: interpreted,
		Parameters:  make(map[string]interface{}),
		Confidence:  confidence,
		Timestamp:   time.Now(),
		SessionID:   sessionID,
	}, nil
}

// Validate checks if the command is valid
func (c *Command) Validate() error {
	if c.ID == "" {
		return errors.New("command ID is required")
	}
	if c.SessionID == "" {
		return errors.New("session ID is required")
	}
	if c.Raw == "" {
		return errors.New("raw input is required")
	}
	if c.Interpreted == "" {
		return errors.New("interpreted command is required")
	}
	if c.Confidence < 0.0 || c.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got %f", c.Confidence)
	}
	return nil
}

// IsHighConfidence returns true if the command has high confidence
func (c *Command) IsHighConfidence() bool {
	return c.Confidence >= 0.8
}

// RequiresClarification returns true if the command needs clarification
func (c *Command) RequiresClarification() bool {
	return c.Confidence < 0.6
}

// AddParameter adds a parameter to the command
func (c *Command) AddParameter(key string, value interface{}) {
	if c.Parameters == nil {
		c.Parameters = make(map[string]interface{})
	}
	c.Parameters[key] = value
}

// GetParameter retrieves a parameter value
func (c *Command) GetParameter(key string) (interface{}, bool) {
	if c.Parameters == nil {
		return nil, false
	}
	val, exists := c.Parameters[key]
	return val, exists
}

// GetStringParameter retrieves a parameter as a string
func (c *Command) GetStringParameter(key string) (string, bool) {
	val, exists := c.GetParameter(key)
	if !exists {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// SetIntent sets the command intent
func (c *Command) SetIntent(action CommandType, resource ResourceType, target string) {
	c.Intent = CommandIntent{
		Action:   action,
		Resource: resource,
		Target:   target,
	}
}

// generateCommandID generates a unique command ID
func generateCommandID() string {
	return fmt.Sprintf("cmd-%d", time.Now().UnixNano())
}

// CommandResult represents the result of executing a command
type CommandResult struct {
	CommandID string
	Success   bool
	Output    interface{}
	Error     error
	Duration  time.Duration
}

// NewCommandResult creates a new command result
func NewCommandResult(commandID string, success bool, output interface{}, err error, duration time.Duration) *CommandResult {
	return &CommandResult{
		CommandID: commandID,
		Success:   success,
		Output:    output,
		Error:     err,
		Duration:  duration,
	}
}
