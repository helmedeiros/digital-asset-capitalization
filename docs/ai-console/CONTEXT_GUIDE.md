# AI Console Context Management Guide

## Overview

This guide documents how the AI Console maintains context across interactions, enabling natural conversational flows and intelligent command interpretation. Context preservation is crucial for handling follow-up questions, references to previous entities, and maintaining session state.

## Context Architecture

### 1. Context Entity Structure

```go
type Context struct {
    // Session Information
    SessionID       string
    StartTime       time.Time
    LastActivity    time.Time
    
    // Command History
    Commands        []Command      // All executed commands
    CommandResults  map[string]interface{} // Results by command ID
    
    // Current State
    CurrentProject  string
    CurrentSprint   string
    CurrentSpace    string
    
    // Entity References
    LastAsset       *domain.Asset
    LastTask        *domain.Task
    LastTeam        *domain.Team
    RecentAssets    []domain.Asset // Last 5 mentioned
    RecentTasks     []domain.Task  // Last 5 mentioned
    
    // User Preferences
    PreferredFormat string // json, yaml, table
    Verbosity       string // quiet, normal, verbose
    
    // Working Variables
    Variables       map[string]interface{}
}
```

### 2. Context Updates

Context is updated after each successful command execution:

```go
func (c *Context) UpdateAfterCommand(cmd Command, result interface{}) {
    c.Commands = append(c.Commands, cmd)
    c.CommandResults[cmd.ID] = result
    c.LastActivity = time.Now()
    
    // Update entity references based on command type
    switch cmd.Resource {
    case "asset":
        c.updateAssetContext(cmd, result)
    case "task":
        c.updateTaskContext(cmd, result)
    case "sprint":
        c.updateSprintContext(cmd, result)
    }
}
```

## Context Usage Patterns

### 1. Pronoun Resolution

When users use pronouns, the console resolves them using context:

| User Input | Context Check | Resolution |
|------------|---------------|------------|
| "Show its details" | LastAsset != nil | Use LastAsset.Name |
| "Delete that" | Check last entity type | Use appropriate last entity |
| "Do the same for Payment" | Last command available | Repeat with new parameter |

### 2. Implicit Parameters

Context provides defaults for missing parameters:

| Command | Missing Parameter | Context Default |
|---------|------------------|-----------------|
| `tasks fetch` | --project | CurrentProject |
| `sprint allocate` | --sprint | CurrentSprint |
| `assets sync` | --space | CurrentSpace |

### 3. Follow-up Questions

Context enables natural follow-ups:

```
User: "List all assets"
Console: [Shows asset list]
Context: Updates RecentAssets with results

User: "Which ones need enrichment?"
Console: [Analyzes RecentAssets for missing fields]
```

## Context Preservation Strategies

### 1. In-Memory Storage

Primary storage for active sessions:

```go
type InMemoryContextStore struct {
    contexts sync.Map // map[sessionID]*Context
    mu       sync.RWMutex
}

func (s *InMemoryContextStore) Save(ctx context.Context, context *Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.contexts.Store(context.SessionID, context)
    return nil
}
```

### 2. Persistent Storage (Optional)

For long-running sessions:

```json
{
  "session_id": "uuid",
  "start_time": "2024-01-01T10:00:00Z",
  "last_activity": "2024-01-01T10:30:00Z",
  "current_project": "FN",
  "current_sprint": "Sprint 23",
  "command_history": [
    {
      "id": "cmd-1",
      "raw": "show all assets",
      "interpreted": "assets list",
      "timestamp": "2024-01-01T10:00:00Z"
    }
  ],
  "recent_entities": {
    "assets": ["Payment Processing", "User Authentication"],
    "tasks": ["FN-1234", "FN-5678"]
  }
}
```

### 3. Context Expiration

Contexts expire after inactivity:

```go
const (
    SessionTimeout = 30 * time.Minute
    MaxSessionSize = 100 * 1024 // 100KB
)

func (s *InMemoryContextStore) CleanupExpired() {
    s.contexts.Range(func(key, value interface{}) bool {
        ctx := value.(*Context)
        if time.Since(ctx.LastActivity) > SessionTimeout {
            s.contexts.Delete(key)
        }
        return true
    })
}
```

## Context-Aware Command Interpretation

### 1. Reference Resolution Algorithm

```
1. Check for pronouns (it, that, them, these)
2. Identify reference type (asset, task, sprint)
3. Look up in context:
   - LastAsset/Task/Team for singular
   - RecentAssets/Tasks for plural
4. Substitute reference with actual value
5. Validate resolved command
```

### 2. Contextual Hints

The AI interpreter receives context hints:

```json
{
  "has_recent_assets": true,
  "last_command_type": "list",
  "current_scope": {
    "project": "FN",
    "sprint": "Sprint 23"
  },
  "available_references": {
    "asset": "Payment Processing",
    "assets": ["Payment Processing", "User Auth"]
  }
}
```

### 3. Smart Defaults

Context provides intelligent defaults:

| Scenario | Default Behavior |
|----------|-----------------|
| After `assets list` | Next asset command uses listed assets |
| After `tasks fetch` | Classification uses fetched tasks |
| In project context | All commands default to that project |

## Context Management Commands

Built-in context commands for users:

| Command | Description |
|---------|------------|
| `context show` | Display current context |
| `context clear` | Reset context |
| `context set project FN` | Set project context |
| `context history` | Show command history |

## Implementation Checklist

### Phase 1: Basic Context
- [ ] Context entity definition
- [ ] In-memory storage
- [ ] Session management
- [ ] Basic updates

### Phase 2: Smart Context
- [ ] Pronoun resolution
- [ ] Reference tracking
- [ ] Implicit parameters
- [ ] Context hints

### Phase 3: Advanced Features
- [ ] Persistent storage
- [ ] Context commands
- [ ] Multi-session support
- [ ] Context sharing

## Best Practices

1. **Minimal Storage**: Only store necessary context
2. **Clear Expiration**: Remove stale contexts
3. **Thread Safety**: Use concurrent-safe structures
4. **Privacy**: Don't persist sensitive data
5. **Performance**: Limit context size

## Testing Context

### Test Scenarios

1. **Basic Flow**
   ```
   > list assets
   > show details for Payment Processing
   > update its description
   ```

2. **Project Context**
   ```
   > set project FN
   > list sprints
   > show sprint 23 tasks
   ```

3. **Complex References**
   ```
   > show all assets
   > which ones have no keywords?
   > generate keywords for them
   ```

## Context Recovery

When context is lost:

1. Check for session file
2. Restore recent commands
3. Re-establish entity references
4. Prompt user if needed

## Monitoring Context

Track context effectiveness:

- Session duration
- Commands per session
- Context hit rate
- Reference resolution success
- Memory usage

This context system enables natural, conversational interactions while maintaining the precision needed for command execution.