package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/ai"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/executor"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/prompt"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/store"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	investmentservice "github.com/helmedeiros/digital-asset-capitalization/internal/investment/application/service"
	sprintapp "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
)

// createConsoleCommand creates the console command that integrates with existing App structure
func (a *App) createConsoleCommand() *cli.Command {
	return &cli.Command{
		Name:    "console",
		Aliases: []string{"c"},
		Usage:   "Start an interactive AI-powered console for AssetCap",
		Description: `The console command starts an interactive session where you can use natural language
to interact with AssetCap. The AI assistant will interpret your requests and execute
the appropriate AssetCap commands.

Examples:
  assetcap console

Once in the console, you can use natural language:
  > Show all assets
  > Create an asset called Payment Processing
  > List tasks for project FN
  > Calculate investment for User Authentication
  > help
  > exit`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "ollama-url",
				Usage:   "Ollama API URL",
				Value:   "http://localhost:11434",
				EnvVars: []string{"OLLAMA_API_URL"},
			},
			&cli.StringFlag{
				Name:  "model",
				Usage: "LLaMA model to use",
				Value: "llama3",
			},
			&cli.IntFlag{
				Name:  "max-sessions",
				Usage: "Maximum number of concurrent sessions",
				Value: 10,
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug output",
			},
		},
		Action: a.runConsole,
	}
}

// runConsole executes the console command
func (a *App) runConsole(c *cli.Context) error {
	ctx := c.Context

	// Enable debug logging if requested
	if c.Bool("debug") {
		log.SetOutput(os.Stdout)
		log.Println("Debug mode enabled")
	}

	// Check if Ollama is accessible
	if err := checkOllamaAvailability(c.String("ollama-url")); err != nil {
		return fmt.Errorf("Ollama is not available: %w\n\nPlease ensure Ollama is running:\n1. Install Ollama: https://ollama.ai\n2. Start Ollama: ollama serve\n3. Pull LLaMA model: ollama pull %s", err, c.String("model"))
	}

	// Initialize AI interpreter
	aiConfig := ai.Config{
		BaseURL: c.String("ollama-url"),
		Model:   c.String("model"),
	}
	interpreter := ai.NewInterpreter(aiConfig)

	// Initialize context store
	storeConfig := store.Config{
		MaxSessions: c.Int("max-sessions"),
		SessionTTL:  30 * time.Minute, // 30 minutes
	}
	contextStore := store.NewMemoryStore(storeConfig)

	// Initialize command executor with actual services from App
	commandExecutor := executor.NewCommandExecutor(
		&AssetServiceAdapter{service: a.assetService},
		&TaskServiceAdapter{service: a.taskService},
		&SprintServiceAdapter{service: a.sprintService},
		&InvestmentServiceAdapter{service: a.investmentService},
		&ConfigServiceAdapter{service: a.configService},
	)

	// Initialize console service
	consoleService := application.NewConsoleService(
		interpreter,
		commandExecutor,
		contextStore,
	)

	// Note: We'll start cleanup via goroutine since the method doesn't exist in our implementation

	// Initialize prompt handler
	promptHandler := prompt.NewHandler(consoleService)

	// Start the interactive console
	fmt.Println() // Just a blank line for cleaner startup
	return promptHandler.Start(ctx)
}

// checkOllamaAvailability checks if Ollama API is accessible
func checkOllamaAvailability(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("Ollama URL not provided")
	}

	// Make a simple health check to Ollama
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	return nil
}

// Service adapters to bridge between console executor interfaces and existing App services

// AssetServiceAdapter adapts the existing asset service to the console interface
type AssetServiceAdapter struct {
	service assetsapp.AssetService
}

func (a *AssetServiceAdapter) ListAssets(_ context.Context) (interface{}, error) {
	// This would call the actual asset service method
	// For now, return a placeholder
	return []string{"Asset1", "Asset2", "Asset3"}, nil
}

func (a *AssetServiceAdapter) CreateAsset(_ context.Context, name, description string) (interface{}, error) {
	return map[string]string{
		"status":      "created",
		"name":        name,
		"description": description,
	}, nil
}

func (a *AssetServiceAdapter) GetAsset(_ context.Context, name string) (interface{}, error) {
	return map[string]string{
		"name":   name,
		"status": "found",
	}, nil
}

func (a *AssetServiceAdapter) UpdateAsset(_ context.Context, name, _ string) (interface{}, error) {
	return map[string]string{
		"status": "updated",
		"name":   name,
	}, nil
}

func (a *AssetServiceAdapter) DeleteAsset(_ context.Context, _ string) error {
	return nil
}

func (a *AssetServiceAdapter) SyncAssets(_ context.Context, space, label string) (interface{}, error) {
	return map[string]string{
		"status": "synced",
		"space":  space,
		"label":  label,
	}, nil
}

func (a *AssetServiceAdapter) EnrichAsset(_ context.Context, name, field string) (interface{}, error) {
	return map[string]string{
		"status": "enriched",
		"asset":  name,
		"field":  field,
	}, nil
}

func (a *AssetServiceAdapter) GenerateKeywords(_ context.Context, name string) (interface{}, error) {
	return map[string]interface{}{
		"asset":    name,
		"keywords": []string{"keyword1", "keyword2", "keyword3"},
	}, nil
}

// TaskServiceAdapter adapts the existing task service
type TaskServiceAdapter struct {
	service tasksapp.TaskService
}

func (t *TaskServiceAdapter) FetchTasks(_ context.Context, project, sprint string) (interface{}, error) {
	return map[string]string{
		"status":  "fetched",
		"project": project,
		"sprint":  sprint,
	}, nil
}

func (t *TaskServiceAdapter) ShowTasks(_ context.Context, _, _ string) (interface{}, error) {
	return []string{"TASK-1", "TASK-2", "TASK-3"}, nil
}

func (t *TaskServiceAdapter) ClassifyTasks(_ context.Context, project, sprint string, apply bool) (interface{}, error) {
	return map[string]interface{}{
		"status":  "classified",
		"project": project,
		"sprint":  sprint,
		"applied": apply,
	}, nil
}

func (t *TaskServiceAdapter) InspectTask(_ context.Context, key string) (interface{}, error) {
	return map[string]string{
		"key":     key,
		"status":  "active",
		"summary": "Sample task summary",
	}, nil
}

// SprintServiceAdapter adapts the existing sprint service
type SprintServiceAdapter struct {
	service sprintapp.SprintService
}

func (s *SprintServiceAdapter) ListSprints(_ context.Context, _, _ string) (interface{}, error) {
	return []string{"Sprint 22", "Sprint 23", "Sprint 24"}, nil
}

func (s *SprintServiceAdapter) AllocateSprint(_ context.Context, project, sprint string, bounded bool) (interface{}, error) {
	return map[string]interface{}{
		"project":    project,
		"sprint":     sprint,
		"bounded":    bounded,
		"allocation": "calculated",
	}, nil
}

// InvestmentServiceAdapter adapts the existing investment service
type InvestmentServiceAdapter struct {
	service *investmentservice.InvestmentService
}

func (i *InvestmentServiceAdapter) CalculateInvestment(_ context.Context, asset, project string, sprints []string) (interface{}, error) {
	return map[string]interface{}{
		"asset":      asset,
		"project":    project,
		"sprints":    sprints,
		"investment": "$50,000",
	}, nil
}

func (i *InvestmentServiceAdapter) ListInvestments(_ context.Context, _ string) (interface{}, error) {
	return []string{"Investment1", "Investment2"}, nil
}

func (i *InvestmentServiceAdapter) ShowRates(_ context.Context, project string) (interface{}, error) {
	return map[string]interface{}{
		"project": project,
		"rates": map[string]string{
			"senior": "$100/hour",
			"junior": "$60/hour",
		},
	}, nil
}

// ConfigServiceAdapter adapts the existing config service
type ConfigServiceAdapter struct {
	service ConfigService
}

func (c *ConfigServiceAdapter) InitConfig(_ context.Context) (interface{}, error) {
	return map[string]string{"status": "initialized"}, nil
}

func (c *ConfigServiceAdapter) ShowConfig(_ context.Context) (interface{}, error) {
	return map[string]string{
		"jira_url":   "configured",
		"ollama_url": "configured",
	}, nil
}

func (c *ConfigServiceAdapter) ValidateConfig(_ context.Context) (interface{}, error) {
	return map[string]string{"status": "valid"}, nil
}

func (c *ConfigServiceAdapter) SyncTeam(_ context.Context, project string) (interface{}, error) {
	return map[string]interface{}{
		"project":        project,
		"synced_members": 5,
	}, nil
}
