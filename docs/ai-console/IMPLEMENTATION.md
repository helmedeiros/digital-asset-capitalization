# AI Console Implementation Guide

## Overview

The AI Console is an interactive natural language interface for AssetCap that allows users to interact with the system using conversational commands. It leverages the existing LLaMA 3 integration to interpret user requests and execute the appropriate AssetCap commands.

## Architecture Overview

The AI Console follows AssetCap's hexagonal architecture pattern:

```
┌─────────────────────────────────────────────────────────────┐
│                      Interface Layer                         │
│                    (cmd/console.go)                         │
├─────────────────────────────────────────────────────────────┤
│                   Application Layer                          │
│              (internal/console/application/)                 │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │ConsoleService│  │InterpretCmd  │  │ExecuteCommand   │   │
│  └─────────────┘  └──────────────┘  └─────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                     Domain Layer                             │
│               (internal/console/domain/)                     │
│  ┌─────────┐  ┌─────────┐  ┌──────────────────────────┐   │
│  │ Command │  │ Context │  │ Ports (Interfaces)       │   │
│  └─────────┘  └─────────┘  └──────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                 Infrastructure Layer                         │
│           (internal/console/infrastructure/)                 │
│  ┌──────────────┐  ┌────────────────┐  ┌──────────────┐   │
│  │AIInterpreter │  │CommandExecutor │  │ContextStore  │   │
│  └──────────────┘  └────────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Domain Layer

#### Command Entity (`internal/console/domain/command.go`)
```go
type Command struct {
    ID          string
    Raw         string      // Original user input
    Interpreted string      // Parsed CLI command
    Parameters  map[string]interface{}
    Confidence  float64
    Timestamp   time.Time
}

type CommandIntent struct {
    Action    string   // e.g., "create", "list", "show"
    Resource  string   // e.g., "asset", "task", "sprint"
    Modifiers []string // e.g., ["all", "detailed"]
}
```

#### Context Entity (`internal/console/domain/context.go`)
```go
type Context struct {
    SessionID       string
    Commands        []Command
    CurrentProject  string
    CurrentSprint   string
    LastAsset       *Asset
    LastTask        *Task
    Variables       map[string]interface{}
}
```

#### Ports (`internal/console/domain/ports/`)
```go
// ai_interpreter.go
type AIInterpreter interface {
    Interpret(ctx context.Context, input string, context *Context) (*Command, error)
    GetClarification(ctx context.Context, ambiguity string) (string, error)
}

// command_executor.go
type CommandExecutor interface {
    Execute(ctx context.Context, command *Command) (interface{}, error)
    ValidateCommand(command *Command) error
}

// context_store.go
type ContextStore interface {
    Save(ctx context.Context, context *Context) error
    Load(ctx context.Context, sessionID string) (*Context, error)
    Update(ctx context.Context, sessionID string, update func(*Context)) error
}
```

### 2. Application Layer

#### Console Service (`internal/console/application/console_service.go`)
Main orchestrator that coordinates the console session lifecycle.

#### Use Cases

**Interpret Command (`internal/console/application/usecase/interpret_command.go`)**
- Takes user input and context
- Calls AI interpreter
- Handles ambiguity resolution
- Returns structured command

**Execute Command (`internal/console/application/usecase/execute_command.go`)**
- Validates interpreted command
- Maps to appropriate AssetCap service
- Executes command
- Formats response

**Maintain Context (`internal/console/application/usecase/maintain_context.go`)**
- Updates context after each interaction
- Tracks entities and references
- Manages session state

### 3. Infrastructure Layer

#### AI Interpreter (`internal/console/infrastructure/ai_interpreter.go`)
- Uses existing LLaMA client
- Implements prompt engineering
- Handles command parsing
- Manages confidence scoring

#### Command Executor (`internal/console/infrastructure/command_executor.go`)
- Maps commands to existing services
- Handles parameter conversion
- Manages service dependencies
- Returns formatted results

#### Context Store (`internal/console/infrastructure/context_store.go`)
- In-memory storage with optional persistence
- Thread-safe operations
- Session management
- Context expiration

### 4. Interface Layer

#### Console Command (`cmd/console.go`)
- Initializes console mode
- Sets up dependencies
- Manages input/output loop
- Handles exit conditions

## Implementation Phases

### Phase 1: Foundation (Current)
1. Create documentation structure
2. Design domain entities
3. Define interfaces
4. Plan infrastructure

### Phase 2: Core Implementation
1. Implement domain layer
2. Create application services
3. Build infrastructure adapters
4. Wire up console command

### Phase 3: Enhancement
1. Add context awareness
2. Implement clarification flows
3. Add help system
4. Create command suggestions

### Phase 4: Testing & Polish
1. Unit tests (>80% coverage)
2. Integration tests
3. User acceptance testing
4. Documentation

## Key Design Decisions

1. **Stateful Sessions**: Each console session maintains context for natural follow-up questions
2. **Confidence Scoring**: AI interpretations include confidence to handle ambiguity
3. **Service Reuse**: Leverages existing AssetCap services rather than reimplementing
4. **Prompt Engineering**: Structured prompts ensure consistent interpretation
5. **Error Recovery**: Graceful handling of misinterpretation with clarification

## Integration Points

1. **Existing Services**: All domain services (assets, tasks, sprint, etc.)
2. **LLaMA Client**: Reuses existing AI infrastructure
3. **CLI Framework**: Integrates with urfave/cli command structure
4. **Configuration**: Uses existing config management

## Example Flow

```
User: "Show me all assets"
├── AI Interpreter
│   └── Interprets as: "assets list"
├── Command Validation
│   └── Valid command structure
├── Command Execution
│   └── Calls AssetService.List()
├── Context Update
│   └── Stores command and results
└── Response Display
    └── Formats and displays assets
```

## Success Criteria

1. Natural language understanding with >90% accuracy for common commands
2. Context-aware conversations
3. Graceful error handling
4. Performance: <2s response time
5. Memory efficiency: <100MB per session