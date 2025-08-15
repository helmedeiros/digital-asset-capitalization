package domain

import (
	"fmt"
	"sync"
	"time"
)

// Context represents the conversation context for a console session
type Context struct {
	mu sync.RWMutex

	// Session Information
	SessionID    string
	StartTime    time.Time
	LastActivity time.Time

	// Command History
	Commands       []Command                // All executed commands in order
	CommandResults map[string]CommandResult // Results by command ID

	// Current State
	CurrentProject string
	CurrentSprint  string
	CurrentSpace   string

	// Entity References - storing as interface{} to avoid circular dependencies
	LastAsset    interface{} // Will be *asset.Asset in practice
	LastTask     interface{} // Will be *task.Task in practice
	LastTeam     interface{} // Will be *team.Team in practice
	RecentAssets []string    // Asset names for the last 5 mentioned
	RecentTasks  []string    // Task keys for the last 5 mentioned

	// User Preferences
	PreferredFormat string // json, yaml, table
	Verbosity       string // quiet, normal, verbose

	// Working Variables
	Variables map[string]interface{}
}

// NewContext creates a new context for a session
func NewContext(sessionID string) *Context {
	return &Context{
		SessionID:       sessionID,
		StartTime:       time.Now(),
		LastActivity:    time.Now(),
		Commands:        make([]Command, 0),
		CommandResults:  make(map[string]CommandResult),
		RecentAssets:    make([]string, 0, 5),
		RecentTasks:     make([]string, 0, 5),
		Variables:       make(map[string]interface{}),
		PreferredFormat: "table",
		Verbosity:       "normal",
	}
}

// AddCommand adds a command to the history
func (c *Context) AddCommand(cmd Command) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Commands = append(c.Commands, cmd)
	c.LastActivity = time.Now()
}

// AddCommandResult stores the result of a command execution
func (c *Context) AddCommandResult(result CommandResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.CommandResults[result.CommandID] = result
	c.LastActivity = time.Now()
}

// GetLastCommand returns the most recent command
func (c *Context) GetLastCommand() (*Command, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Commands) == 0 {
		return nil, false
	}
	return &c.Commands[len(c.Commands)-1], true
}

// GetCommandHistory returns the last n commands
func (c *Context) GetCommandHistory(n int) []Command {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if n <= 0 || len(c.Commands) == 0 {
		return []Command{}
	}

	start := len(c.Commands) - n
	if start < 0 {
		start = 0
	}

	result := make([]Command, len(c.Commands[start:]))
	copy(result, c.Commands[start:])
	return result
}

// UpdateAssetContext updates the asset-related context
func (c *Context) UpdateAssetContext(assetName string, assetObj interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastAsset = assetObj
	c.addRecentAsset(assetName)
	c.LastActivity = time.Now()
}

// UpdateTaskContext updates the task-related context
func (c *Context) UpdateTaskContext(taskKey string, taskObj interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastTask = taskObj
	c.addRecentTask(taskKey)
	c.LastActivity = time.Now()
}

// UpdateTeamContext updates the team-related context
func (c *Context) UpdateTeamContext(teamObj interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastTeam = teamObj
	c.LastActivity = time.Now()
}

// SetCurrentProject sets the current project context
func (c *Context) SetCurrentProject(project string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.CurrentProject = project
	c.LastActivity = time.Now()
}

// SetCurrentSprint sets the current sprint context
func (c *Context) SetCurrentSprint(sprint string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.CurrentSprint = sprint
	c.LastActivity = time.Now()
}

// SetCurrentSpace sets the current Confluence space context
func (c *Context) SetCurrentSpace(space string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.CurrentSpace = space
	c.LastActivity = time.Now()
}

// SetVariable sets a working variable
func (c *Context) SetVariable(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Variables == nil {
		c.Variables = make(map[string]interface{})
	}
	c.Variables[key] = value
	c.LastActivity = time.Now()
}

// GetVariable retrieves a working variable
func (c *Context) GetVariable(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Variables == nil {
		return nil, false
	}
	val, exists := c.Variables[key]
	return val, exists
}

// IsExpired checks if the context has expired based on inactivity
func (c *Context) IsExpired(timeout time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.LastActivity) > timeout
}

// GetSessionDuration returns how long the session has been active
func (c *Context) GetSessionDuration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.StartTime)
}

// Clear resets the context while keeping the session ID
func (c *Context) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Commands = make([]Command, 0)
	c.CommandResults = make(map[string]CommandResult)
	c.CurrentProject = ""
	c.CurrentSprint = ""
	c.CurrentSpace = ""
	c.LastAsset = nil
	c.LastTask = nil
	c.LastTeam = nil
	c.RecentAssets = make([]string, 0, 5)
	c.RecentTasks = make([]string, 0, 5)
	c.Variables = make(map[string]interface{})
	c.LastActivity = time.Now()
}

// GetSummary returns a summary of the current context
func (c *Context) GetSummary() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := fmt.Sprintf("Session: %s\n", c.SessionID)
	summary += fmt.Sprintf("Duration: %s\n", c.GetSessionDuration())
	summary += fmt.Sprintf("Commands executed: %d\n", len(c.Commands))

	if c.CurrentProject != "" {
		summary += fmt.Sprintf("Current project: %s\n", c.CurrentProject)
	}
	if c.CurrentSprint != "" {
		summary += fmt.Sprintf("Current sprint: %s\n", c.CurrentSprint)
	}
	if c.CurrentSpace != "" {
		summary += fmt.Sprintf("Current space: %s\n", c.CurrentSpace)
	}

	if len(c.RecentAssets) > 0 {
		summary += fmt.Sprintf("Recent assets: %v\n", c.RecentAssets)
	}
	if len(c.RecentTasks) > 0 {
		summary += fmt.Sprintf("Recent tasks: %v\n", c.RecentTasks)
	}

	return summary
}

// Helper methods

func (c *Context) addRecentAsset(assetName string) {
	// Remove if already exists
	for i, name := range c.RecentAssets {
		if name == assetName {
			c.RecentAssets = append(c.RecentAssets[:i], c.RecentAssets[i+1:]...)
			break
		}
	}

	// Add to front
	c.RecentAssets = append([]string{assetName}, c.RecentAssets...)

	// Keep only last 5
	if len(c.RecentAssets) > 5 {
		c.RecentAssets = c.RecentAssets[:5]
	}
}

func (c *Context) addRecentTask(taskKey string) {
	// Remove if already exists
	for i, key := range c.RecentTasks {
		if key == taskKey {
			c.RecentTasks = append(c.RecentTasks[:i], c.RecentTasks[i+1:]...)
			break
		}
	}

	// Add to front
	c.RecentTasks = append([]string{taskKey}, c.RecentTasks...)

	// Keep only last 5
	if len(c.RecentTasks) > 5 {
		c.RecentTasks = c.RecentTasks[:5]
	}
}
