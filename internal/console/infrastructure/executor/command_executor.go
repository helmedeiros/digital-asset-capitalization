package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// Constants for repeated strings
const (
	commandList = "list"
	commandShow = "show"
)

// CommandExecutor implements the command execution interface
type CommandExecutor struct {
	// Service dependencies - these would be injected from existing AssetCap services
	assetService      AssetServiceInterface
	taskService       TaskServiceInterface
	sprintService     SprintServiceInterface
	investmentService InvestmentServiceInterface
	configService     ConfigServiceInterface
}

// Service interfaces - these represent the existing AssetCap services
type AssetServiceInterface interface {
	ListAssets(ctx context.Context) (interface{}, error)
	CreateAsset(ctx context.Context, name, description string) (interface{}, error)
	GetAsset(ctx context.Context, name string) (interface{}, error)
	UpdateAsset(ctx context.Context, name, description string) (interface{}, error)
	DeleteAsset(ctx context.Context, name string) error
	SyncAssets(ctx context.Context, space, label string) (interface{}, error)
	EnrichAsset(ctx context.Context, name, field string) (interface{}, error)
	GenerateKeywords(ctx context.Context, name string) (interface{}, error)
	// Team management methods
	AssignTeamOwner(ctx context.Context, asset, team string) (interface{}, error)
	AddContributingTeam(ctx context.Context, asset, team string) (interface{}, error)
	RemoveContributingTeam(ctx context.Context, asset, team string) (interface{}, error)
	ShowTeamAssignments(ctx context.Context, asset string) (interface{}, error)
	ListTeamAssignments(ctx context.Context) (interface{}, error)
	// Advanced asset operations
	SyncAndEnrich(ctx context.Context, space, label string, keywords bool, fields []string) (interface{}, error)
}

type TaskServiceInterface interface {
	FetchTasks(ctx context.Context, project, sprint string) (interface{}, error)
	ShowTasks(ctx context.Context, project, sprint string) (interface{}, error)
	ClassifyTasks(ctx context.Context, project, sprint string, apply bool) (interface{}, error)
	InspectTask(ctx context.Context, key string) (interface{}, error)
}

type SprintServiceInterface interface {
	ListSprints(ctx context.Context, project, period string) (interface{}, error)
	AllocateSprint(ctx context.Context, project, sprint string, bounded bool) (interface{}, error)
}

type InvestmentServiceInterface interface {
	CalculateInvestment(ctx context.Context, asset, project string, sprints []string) (interface{}, error)
	ListInvestments(ctx context.Context, project string) (interface{}, error)
	ShowRates(ctx context.Context, project string) (interface{}, error)
}

type ConfigServiceInterface interface {
	InitConfig(ctx context.Context) (interface{}, error)
	ShowConfig(ctx context.Context) (interface{}, error)
	ValidateConfig(ctx context.Context) (interface{}, error)
	SyncTeam(ctx context.Context, project string) (interface{}, error)
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(
	assetService AssetServiceInterface,
	taskService TaskServiceInterface,
	sprintService SprintServiceInterface,
	investmentService InvestmentServiceInterface,
	configService ConfigServiceInterface,
) *CommandExecutor {
	return &CommandExecutor{
		assetService:      assetService,
		taskService:       taskService,
		sprintService:     sprintService,
		investmentService: investmentService,
		configService:     configService,
	}
}

// Execute runs the command and returns the result
func (e *CommandExecutor) Execute(ctx context.Context, command *domain.Command) (*domain.CommandResult, error) {
	startTime := time.Now()

	// Route command based on resource type and action
	output, err := e.routeCommand(ctx, command)
	duration := time.Since(startTime)

	if err != nil {
		return domain.NewCommandResult(command.ID, false, nil, err, duration), err
	}

	return domain.NewCommandResult(command.ID, true, output, nil, duration), nil
}

// ValidateCommand checks if a command can be executed
func (e *CommandExecutor) ValidateCommand(command *domain.Command) error {
	if command == nil {
		return ports.NewValidationError("command", "command cannot be nil")
	}

	if err := command.Validate(); err != nil {
		return ports.NewValidationError("command", err.Error())
	}

	// Check if command is interpretable
	if command.Interpreted == "" {
		return ports.NewValidationError("interpreted", "command has not been interpreted")
	}

	// Parse and validate command structure
	parts := strings.Fields(command.Interpreted)
	if len(parts) < 2 {
		return ports.NewValidationError("format", "command must have at least resource and action")
	}

	resource := parts[0]
	action := parts[1]

	// Validate resource type
	if !e.isValidResource(resource) {
		return ports.NewValidationError("resource", fmt.Sprintf("unsupported resource: %s", resource))
	}

	// Validate action for resource
	if !e.isValidAction(resource, action) {
		return ports.NewValidationError("action", fmt.Sprintf("unsupported action '%s' for resource '%s'", action, resource))
	}

	// Resource-specific validation
	return e.validateResourceSpecific(command, resource, action)
}

// GetAvailableCommands returns all available commands for help/suggestions
func (e *CommandExecutor) GetAvailableCommands() []ports.CommandInfo {
	return []ports.CommandInfo{
		{
			Command:     "assets list",
			Description: "List all assets",
			Examples:    []string{"show all assets", "list assets"},
			Parameters: []ports.ParameterInfo{
				{Name: "format", Description: "Output format", Required: false, Type: "string", Default: "table"},
			},
		},
		{
			Command:     "assets create",
			Description: "Create a new asset",
			Examples:    []string{"create an asset called Payment Processing"},
			Parameters: []ports.ParameterInfo{
				{Name: "name", Description: "Asset name", Required: true, Type: "string"},
				{Name: "description", Description: "Asset description", Required: false, Type: "string"},
			},
		},
		{
			Command:     "assets show",
			Description: "Show details for a specific asset",
			Examples:    []string{"show asset Payment Processing", "show details for User Authentication"},
			Parameters: []ports.ParameterInfo{
				{Name: "name", Description: "Asset name", Required: true, Type: "string"},
			},
		},
		{
			Command:     "assets teams assign",
			Description: "Assign team ownership to an asset",
			Examples:    []string{"assign team Platform to asset Payment Processing", "make team Backend owner of User Authentication"},
			Parameters: []ports.ParameterInfo{
				{Name: "asset", Description: "Asset name", Required: true, Type: "string"},
				{Name: "team", Description: "Team name", Required: true, Type: "string"},
			},
		},
		{
			Command:     "assets teams add-contributor",
			Description: "Add contributing team to an asset",
			Examples:    []string{"add team Frontend as contributor to Payment Processing"},
			Parameters: []ports.ParameterInfo{
				{Name: "asset", Description: "Asset name", Required: true, Type: "string"},
				{Name: "team", Description: "Team name", Required: true, Type: "string"},
			},
		},
		{
			Command:     "assets teams show",
			Description: "Show team assignments for an asset",
			Examples:    []string{"show teams for Payment Processing", "who owns User Authentication"},
			Parameters: []ports.ParameterInfo{
				{Name: "asset", Description: "Asset name", Required: true, Type: "string"},
			},
		},
		{
			Command:     "assets teams list",
			Description: "List all team assignments",
			Examples:    []string{"show all team assignments", "list asset ownership"},
			Parameters:  []ports.ParameterInfo{},
		},
		{
			Command:     "tasks fetch",
			Description: "Fetch tasks from JIRA",
			Examples:    []string{"fetch tasks for project FN sprint 23"},
			Parameters: []ports.ParameterInfo{
				{Name: "project", Description: "JIRA project key", Required: true, Type: "string"},
				{Name: "sprint", Description: "Sprint name", Required: true, Type: "string"},
			},
		},
		{
			Command:     "tasks classify",
			Description: "Classify tasks to assets",
			Examples:    []string{"classify tasks for project FN"},
			Parameters: []ports.ParameterInfo{
				{Name: "project", Description: "JIRA project key", Required: true, Type: "string"},
				{Name: "sprint", Description: "Sprint name", Required: true, Type: "string"},
				{Name: "apply", Description: "Apply classifications", Required: false, Type: "bool", Default: false},
			},
		},
		{
			Command:     "sprint list",
			Description: "List sprints for a project",
			Examples:    []string{"list sprints for project FN"},
			Parameters: []ports.ParameterInfo{
				{Name: "project", Description: "Project key", Required: true, Type: "string"},
				{Name: "period", Description: "Time period", Required: false, Type: "string"},
			},
		},
		{
			Command:     "investment calculate",
			Description: "Calculate investment for an asset",
			Examples:    []string{"calculate investment for Payment Processing"},
			Parameters: []ports.ParameterInfo{
				{Name: "asset", Description: "Asset name", Required: true, Type: "string"},
				{Name: "project", Description: "Project key", Required: true, Type: "string"},
				{Name: "sprints", Description: "Sprint names", Required: false, Type: "string"},
			},
		},
		{
			Command:     "config sync-team",
			Description: "Sync team members from JIRA for a project",
			Examples:    []string{"show team members for project FN", "sync team for AD project", "get team members for FN"},
			Parameters: []ports.ParameterInfo{
				{Name: "project", Description: "Project key", Required: true, Type: "string"},
			},
		},
		{
			Command:     "config show",
			Description: "Show current configuration",
			Examples:    []string{"show configuration", "display config"},
			Parameters:  []ports.ParameterInfo{},
		},
	}
}

// routeCommand routes the command to the appropriate service
func (e *CommandExecutor) routeCommand(ctx context.Context, command *domain.Command) (interface{}, error) {
	parts := strings.Fields(command.Interpreted)
	if len(parts) < 1 {
		return nil, fmt.Errorf("empty command")
	}

	resource := parts[0]
	var action string
	if len(parts) > 1 {
		action = parts[1]
	}

	switch resource {
	case "assets":
		if action == "" {
			return nil, fmt.Errorf("assets command requires an action")
		}
		return e.executeAssetCommand(ctx, action, command)
	case "tasks":
		if action == "" {
			return nil, fmt.Errorf("tasks command requires an action")
		}
		return e.executeTaskCommand(ctx, action, command)
	case "sprint":
		if action == "" {
			return nil, fmt.Errorf("sprint command requires an action")
		}
		return e.executeSprintCommand(ctx, action, command)
	case "investment":
		if action == "" {
			return nil, fmt.Errorf("investment command requires an action")
		}
		return e.executeInvestmentCommand(ctx, action, command)
	case "config":
		if action == "" {
			return nil, fmt.Errorf("config command requires an action")
		}
		return e.executeConfigCommand(ctx, action, command)
	case "help":
		return e.executeHelpCommand(ctx, command)
	case "exit":
		return map[string]string{"action": "exit"}, nil
	case "context":
		if action == "" {
			return nil, fmt.Errorf("context command requires an action")
		}
		return e.executeContextCommand(ctx, action, command)
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown resource: %s", resource),
			"Use 'help' to see available commands")
	}
}

// executeAssetCommand executes asset-related commands
func (e *CommandExecutor) executeAssetCommand(ctx context.Context, action string, command *domain.Command) (interface{}, error) {
	if e.assetService == nil {
		return nil, fmt.Errorf("asset service not available")
	}

	switch action {
	case commandList:
		return e.assetService.ListAssets(ctx)
	case "create":
		name, _ := command.GetStringParameter("name")
		description, _ := command.GetStringParameter("description")
		if name == "" {
			return nil, ports.NewValidationError("name", "asset name is required")
		}
		return e.assetService.CreateAsset(ctx, name, description)
	case commandShow:
		name, _ := command.GetStringParameter("name")
		if name == "" {
			return nil, ports.NewValidationError("name", "asset name is required")
		}
		return e.assetService.GetAsset(ctx, name)
	case "update":
		name, _ := command.GetStringParameter("name")
		description, _ := command.GetStringParameter("description")
		if name == "" {
			return nil, ports.NewValidationError("name", "asset name is required")
		}
		return e.assetService.UpdateAsset(ctx, name, description)
	case "delete":
		name, _ := command.GetStringParameter("name")
		if name == "" {
			return nil, ports.NewValidationError("name", "asset name is required")
		}
		return nil, e.assetService.DeleteAsset(ctx, name)
	case "sync":
		space, _ := command.GetStringParameter("space")
		label, _ := command.GetStringParameter("label")
		return e.assetService.SyncAssets(ctx, space, label)
	case "enrich":
		name, _ := command.GetStringParameter("name")
		field, _ := command.GetStringParameter("field")
		if name == "" || field == "" {
			return nil, ports.NewValidationError("parameters", "name and field are required")
		}
		return e.assetService.EnrichAsset(ctx, name, field)
	case "keywords":
		name, _ := command.GetStringParameter("name")
		if name == "" {
			return nil, ports.NewValidationError("name", "asset name is required")
		}
		return e.assetService.GenerateKeywords(ctx, name)
	case "teams":
		return e.executeAssetTeamsCommand(ctx, command)
	case "sync-and-enrich":
		space, _ := command.GetStringParameter("space")
		label, _ := command.GetStringParameter("label")
		keywords, _ := command.GetParameter("keywords")
		keywordsBool, _ := keywords.(bool)
		fieldsParam, _ := command.GetParameter("fields")
		var fields []string
		if fieldsStr, ok := fieldsParam.(string); ok && fieldsStr != "" {
			fields = strings.Split(fieldsStr, ",")
		}
		return e.assetService.SyncAndEnrich(ctx, space, label, keywordsBool, fields)
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown asset action: %s", action),
			"Available actions: list, create, show, update, delete, sync, enrich, keywords, teams, sync-and-enrich")
	}
}

// executeAssetTeamsCommand executes asset team management commands
func (e *CommandExecutor) executeAssetTeamsCommand(ctx context.Context, command *domain.Command) (interface{}, error) {
	if e.assetService == nil {
		return nil, fmt.Errorf("asset service not available")
	}

	// Parse the teams subcommand from the interpreted command
	parts := strings.Fields(command.Interpreted)
	if len(parts) < 3 {
		return nil, ports.NewValidationError("subcommand", "teams command requires a subcommand (assign, add-contributor, remove-contributor, show, list)")
	}

	subcommand := parts[2] // assets teams [subcommand]

	switch subcommand {
	case "assign":
		asset, _ := command.GetStringParameter("asset")
		team, _ := command.GetStringParameter("team")
		if asset == "" {
			return nil, ports.NewValidationError("asset", "asset name is required")
		}
		if team == "" {
			return nil, ports.NewValidationError("team", "team name is required")
		}
		return e.assetService.AssignTeamOwner(ctx, asset, team)
	case "add-contributor":
		asset, _ := command.GetStringParameter("asset")
		team, _ := command.GetStringParameter("team")
		if asset == "" {
			return nil, ports.NewValidationError("asset", "asset name is required")
		}
		if team == "" {
			return nil, ports.NewValidationError("team", "team name is required")
		}
		return e.assetService.AddContributingTeam(ctx, asset, team)
	case "remove-contributor":
		asset, _ := command.GetStringParameter("asset")
		team, _ := command.GetStringParameter("team")
		if asset == "" {
			return nil, ports.NewValidationError("asset", "asset name is required")
		}
		if team == "" {
			return nil, ports.NewValidationError("team", "team name is required")
		}
		return e.assetService.RemoveContributingTeam(ctx, asset, team)
	case commandShow:
		asset, _ := command.GetStringParameter("asset")
		if asset == "" {
			return nil, ports.NewValidationError("asset", "asset name is required")
		}
		return e.assetService.ShowTeamAssignments(ctx, asset)
	case commandList:
		return e.assetService.ListTeamAssignments(ctx)
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown teams subcommand: %s", subcommand),
			"Available subcommands: assign, add-contributor, remove-contributor, show, list")
	}
}

// executeTaskCommand executes task-related commands
func (e *CommandExecutor) executeTaskCommand(ctx context.Context, action string, command *domain.Command) (interface{}, error) {
	if e.taskService == nil {
		return nil, fmt.Errorf("task service not available")
	}

	switch action {
	case "fetch":
		project, _ := command.GetStringParameter("project")
		sprint, _ := command.GetStringParameter("sprint")
		if project == "" {
			return nil, ports.NewValidationError("project", "project is required")
		}
		return e.taskService.FetchTasks(ctx, project, sprint)
	case commandShow:
		project, _ := command.GetStringParameter("project")
		sprint, _ := command.GetStringParameter("sprint")
		return e.taskService.ShowTasks(ctx, project, sprint)
	case "classify":
		project, _ := command.GetStringParameter("project")
		sprint, _ := command.GetStringParameter("sprint")
		apply, _ := command.GetParameter("apply")
		applyBool, _ := apply.(bool)
		return e.taskService.ClassifyTasks(ctx, project, sprint, applyBool)
	case "inspect":
		key, _ := command.GetStringParameter("key")
		if key == "" {
			return nil, ports.NewValidationError("key", "task key is required")
		}
		return e.taskService.InspectTask(ctx, key)
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown task action: %s", action),
			"Available actions: fetch, show, classify, inspect")
	}
}

// executeSprintCommand executes sprint-related commands
func (e *CommandExecutor) executeSprintCommand(ctx context.Context, action string, command *domain.Command) (interface{}, error) {
	if e.sprintService == nil {
		return nil, fmt.Errorf("sprint service not available")
	}

	switch action {
	case commandList:
		project, _ := command.GetStringParameter("project")
		period, _ := command.GetStringParameter("period")
		return e.sprintService.ListSprints(ctx, project, period)
	case "allocate":
		project, _ := command.GetStringParameter("project")
		sprint, _ := command.GetStringParameter("sprint")
		bounded, _ := command.GetParameter("sprint-bounded")
		boundedBool, _ := bounded.(bool)
		return e.sprintService.AllocateSprint(ctx, project, sprint, boundedBool)
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown sprint action: %s", action),
			"Available actions: list, allocate")
	}
}

// executeInvestmentCommand executes investment-related commands
func (e *CommandExecutor) executeInvestmentCommand(ctx context.Context, action string, command *domain.Command) (interface{}, error) {
	if e.investmentService == nil {
		return nil, fmt.Errorf("investment service not available")
	}

	switch action {
	case "calculate":
		asset, _ := command.GetStringParameter("asset")
		project, _ := command.GetStringParameter("project")
		sprints, _ := command.GetStringParameter("sprints")
		var sprintList []string
		if sprints != "" {
			sprintList = strings.Split(sprints, ",")
		}
		return e.investmentService.CalculateInvestment(ctx, asset, project, sprintList)
	case commandList:
		project, _ := command.GetStringParameter("project")
		return e.investmentService.ListInvestments(ctx, project)
	case "show-rates":
		project, _ := command.GetStringParameter("project")
		return e.investmentService.ShowRates(ctx, project)
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown investment action: %s", action),
			"Available actions: calculate, list, show-rates")
	}
}

// executeConfigCommand executes config-related commands
func (e *CommandExecutor) executeConfigCommand(ctx context.Context, action string, command *domain.Command) (interface{}, error) {
	if e.configService == nil {
		return nil, fmt.Errorf("config service not available")
	}

	switch action {
	case "init":
		return e.configService.InitConfig(ctx)
	case commandShow:
		return e.configService.ShowConfig(ctx)
	case "validate":
		return e.configService.ValidateConfig(ctx)
	case "sync-team":
		project, _ := command.GetStringParameter("project")
		if project == "" {
			return nil, ports.NewValidationError("project", "project is required")
		}
		return e.configService.SyncTeam(ctx, project)
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown config action: %s", action),
			"Available actions: init, show, validate, sync-team")
	}
}

// executeHelpCommand executes help commands
func (e *CommandExecutor) executeHelpCommand(_ context.Context, _ *domain.Command) (interface{}, error) {
	commands := e.GetAvailableCommands()
	return map[string]interface{}{
		"message":  "Available commands:",
		"commands": commands,
	}, nil
}

// executeContextCommand executes context-related commands
func (e *CommandExecutor) executeContextCommand(_ context.Context, action string, command *domain.Command) (interface{}, error) {
	switch action {
	case commandShow:
		return map[string]string{
			"message": "Context information will be displayed by the console service",
		}, nil
	case "clear":
		return map[string]string{
			"message": "Context will be cleared by the console service",
		}, nil
	default:
		return nil, ports.NewExecutionError(command.Interpreted,
			fmt.Sprintf("Unknown context action: %s", action),
			"Available actions: show, clear")
	}
}

// isValidResource checks if a resource type is valid
func (e *CommandExecutor) isValidResource(resource string) bool {
	validResources := []string{"assets", "tasks", "sprint", "investment", "config", "help", "exit", "context"}
	for _, valid := range validResources {
		if resource == valid {
			return true
		}
	}
	return false
}

// isValidAction checks if an action is valid for a resource
func (e *CommandExecutor) isValidAction(resource, action string) bool {
	validActions := map[string][]string{
		"assets":     {"list", "create", "show", "update", "delete", "sync", "enrich", "keywords", "sync-and-enrich", "teams"},
		"tasks":      {"fetch", "show", "classify", "inspect"},
		"sprint":     {"list", "allocate"},
		"investment": {"calculate", "list", "show-rates"},
		"config":     {"init", "show", "validate", "sync-team"},
		"context":    {"show", "clear"},
	}

	actions, exists := validActions[resource]
	if !exists {
		return false
	}

	for _, valid := range actions {
		if action == valid {
			return true
		}
	}
	return false
}

// validateResourceSpecific performs resource-specific validation
func (e *CommandExecutor) validateResourceSpecific(command *domain.Command, resource, action string) error {
	// Add specific validation rules based on resource and action
	switch resource {
	case "assets":
		if action == "create" || action == "show" || action == "update" || action == "delete" {
			if name, _ := command.GetStringParameter("name"); name == "" {
				return ports.NewValidationError("name", "asset name is required")
			}
		}
	case "tasks":
		if action == "fetch" || action == "classify" {
			if project, _ := command.GetStringParameter("project"); project == "" {
				return ports.NewValidationError("project", "project is required")
			}
		}
	}

	return nil
}
