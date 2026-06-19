package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	assetid "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/id"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
	configapp "github.com/helmedeiros/digital-asset-capitalization/internal/config/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	deploymentsapp "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/application"
	deploymentports "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
	investmentservice "github.com/helmedeiros/digital-asset-capitalization/internal/investment/application/service"
	investmentinfra "github.com/helmedeiros/digital-asset-capitalization/internal/investment/infrastructure"
	sprintapp "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application"
	sprintusecase "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	sprintinfra "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/infrastructure"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
	tasksusecase "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	taskports "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/classifier"
	cliui "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/cli"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/storage"
)

const (
	configDir  = ".assetcap"
	assetsDir  = ".assetcap"
	assetsFile = "assets.json"
	tasksDir   = ".assetcap"
	tasksFile  = "tasks.json"
	teamsFile  = "teams.json"
)

// Version information - these will be overridden by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// App holds all the application dependencies
type App struct {
	assetService           assetsapp.AssetService
	taskService            tasksapp.TaskService
	sprintService          sprintapp.SprintService
	configService          ConfigService
	teamConfigService      TeamConfigService
	syncTeamService        SyncTeamFromJiraService
	syncTeamServiceFactory func() (SyncTeamFromJiraService, error)
	investmentService      *investmentservice.InvestmentService
	deploymentService      *deploymentsapp.DeploymentService
	deploymentRepo         deploymentports.DeploymentRepository
	teamResolver           *configapp.TeamResolverService
	taskRepo               taskports.TaskRepository
	taskClassifier         taskports.TaskClassifier
	allocationLockRepo     taskports.SprintLockRepository
}

// ConfigService interface for configuration operations
type ConfigService interface {
	InitializeConfig(interactive bool) (*usecase.InitializeConfigResult, error)
	GetJiraConfig() (*domain.JiraConfig, error)
}

// TeamConfigService is the subset of *service.ConfigService that the
// team-tribe / team-company / excluded-issue-types subcommand Actions
// call. Pulling it out as an interface lets tests inject a stub
// without standing up the JSON file repository, and decouples the
// cmd/ Actions from a concrete service type. Grows incrementally as
// more Actions get extracted.
type TeamConfigService interface {
	GetTeamConfig() (*domain.TeamConfig, error)
	SaveTeamConfig(*domain.TeamConfig) error
	SetTribeForProject(project, tribe string) error
	GetTribeForProject(project string) (string, error)
	SetCompanyForProject(project, company string) error
	GetCompanyForProject(project string) (string, error)
	SetExcludedIssueTypesForProject(project string, types []string) error
	GetExcludedIssueTypesForProject(project string) ([]string, error)
	SetBoardWorkStream(project string, boardID int, workStream string) error
}

// SyncTeamFromJiraService is the subset of *usecase.SyncTeamFromJira
// that the `config sync-team` Action needs. Pulling it out as an
// interface lets tests inject a stub without standing up the JIRA
// adapter (which depends on env vars and reaches the network).
type SyncTeamFromJiraService interface {
	Execute(projectKey string) (*domain.TeamSyncResult, error)
}

// configServiceImpl implements ConfigService
type configServiceImpl struct {
	initializeConfig *usecase.InitializeConfig
	configService    *service.ConfigService
}

func (c *configServiceImpl) InitializeConfig(interactive bool) (*usecase.InitializeConfigResult, error) {
	return c.initializeConfig.Execute(interactive)
}

func (c *configServiceImpl) GetJiraConfig() (*domain.JiraConfig, error) {
	return c.configService.GetJiraConfig()
}

// NewApp creates a new App instance with the given dependencies
func NewApp(assetService assetsapp.AssetService, taskService tasksapp.TaskService, sprintService sprintapp.SprintService) *App {
	return &App{
		assetService:  assetService,
		taskService:   taskService,
		sprintService: sprintService,
		configService: nil, // Will be set in initializeApp
	}
}

// NewAppWithConfigService creates a new App instance with config service for testing
func NewAppWithConfigService(assetService assetsapp.AssetService, taskService tasksapp.TaskService, sprintService sprintapp.SprintService, configService ConfigService) *App {
	return &App{
		assetService:  assetService,
		taskService:   taskService,
		sprintService: sprintService,
		configService: configService,
	}
}

// Run executes the CLI application
func (a *App) Run() error {
	app := &cli.App{
		Name:                 "AssetCap",
		Usage:                "Digital Asset Capitalization Management Tool",
		Version:              version,
		EnableBashCompletion: true,
		UsageText: `assetcap [global options] command [command options] [arguments...]

COMMANDS:
   version            Show version information
   completion         Generate shell completion scripts
     bash            Generate bash completion script
     zsh             Generate zsh completion script
     fish            Generate fish completion script
   config             Manage configuration settings
     init            Initialize configuration interactively
     show            Show current configuration
     validate        Validate current configuration
   assets             Manage digital assets
     create          Create a new asset
     list            List all assets
     sync            Sync assets from Confluence
     update          Update an asset's description
     show            Show detailed information about an asset
     enrich          Enrich asset fields using LLaMA 3
     keywords        Generate keywords for an asset using LLaMA 3
     documentation   Manage asset documentation
       update        Mark asset documentation as updated
     tasks           Manage asset tasks
       increment     Increment task count for an asset
       decrement     Decrement task count for an asset
     teams           Manage asset team assignments
       assign        Assign teams to an asset
       list          List asset team assignments
       show          Show team assignments for a specific asset
       add-contributor    Add a contributing team to an asset
       remove-contributor Remove a contributing team from an asset
   tasks              Manage tasks from various platforms
     fetch           Fetch tasks from a platform (e.g., Jira)
     show            Show tasks for a project and sprint
     classify        Classify tasks for a specific project and sprint
     inspect         Inspect a specific task by its key
     migrate         Migrate sprint data from comma-separated strings to arrays
   sprint             Manage sprint-related operations
     list            List sprints for a project and time period
     allocate        Calculate time allocation for JIRA issues in a sprint
   investment         Calculate investment costs for digital assets
     init-cost-model Initialize cost model for a project
     set-engineer-rate Set hourly rate for a specific engineer
     show-rates      Show current engineer rates for a project
     calculate       Calculate investment for an asset across sprints
     sprint          Calculate investment for a specific sprint
     list            List all saved investment calculations
   deployments        Manage deployment tracking
     record          Record a new deployment
     list            List deployments
     history         Show deployment history for an asset
     timeline        Show deployment timeline for a time range
     mock            Generate mock deployment data for testing

For more information about a command:
   assetcap [command] --help`,
		Commands: []*cli.Command{
			a.createVersionCommand(),
			a.createCompletionCommand(),
			a.createSprintCommand(),
			a.createAssetsCommand(),
			a.createTasksCommand(),
			a.createConfigCommand(),
			a.createInvestmentCommand(),
			a.createDeploymentCommands(),
			a.createConsoleCommand(),
		},
	}

	return app.Run(os.Args)
}

// maskToken masks sensitive token information for display
func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// handleAllocateApply handles the --apply flow for sprint allocate
func (a *App) handleAllocateApply(project, sprint, override string, sprintBounded, dryRun, force bool, opts []sprintusecase.SprintAllocationOption) error {
	bgCtx := context.Background()

	// Lock check (skip for dry-run)
	if !dryRun && a.allocationLockRepo != nil {
		lockKey := fmt.Sprintf("alloc::%s", sprint)
		existing, err := a.allocationLockRepo.FindLock(bgCtx, project, lockKey)
		if err != nil {
			return fmt.Errorf("failed to check allocation lock: %w", err)
		}
		if existing != nil {
			if !force {
				return fmt.Errorf("sprint %q in project %q was already pushed on %s (%d issues). Use --force to override",
					sprint, project, existing.LockedAt.Format("2006-01-02"), existing.TaskCount)
			}
			fmt.Printf("Warning: sprint %q in project %q was already pushed on %s (%d issues).\n",
				sprint, project, existing.LockedAt.Format("2006-01-02"), existing.TaskCount)
			fmt.Print("Are you sure you want to re-push? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	csvData, pushResult, err := a.sprintService.PushAllocationToJira(project, sprint, override, sprintBounded, dryRun, opts...)
	if err != nil {
		return err
	}

	// Print CSV output
	fmt.Print(csvData)

	// Print push summary table
	if pushResult != nil && len(pushResult.Details) > 0 {
		fmt.Println("\nAllocation Push Summary:")
		fmt.Printf("%-12s | %-20s | %-15s | %-15s | %-12s | %s\n",
			"Issue", "Field", "Current", "New", "Status", "Reason")
		fmt.Println(strings.Repeat("-", 95))
		for _, d := range pushResult.Details {
			oldVal := d.OldValue
			if oldVal == "" {
				oldVal = "(empty)"
			}
			newVal := d.NewValue
			if newVal == "" {
				newVal = ""
			}
			fmt.Printf("%-12s | %-20s | %-15s | %-15s | %-12s | %s\n",
				d.IssueKey, d.Field, oldVal, newVal, d.Status, d.Reason)
		}
		fmt.Printf("\nUpdated: %d  Skipped: %d  Errors: %d\n",
			pushResult.UpdatedCount, pushResult.SkippedCount, pushResult.ErrorCount)
	}

	if dryRun {
		fmt.Println("\nDry run complete. Remove --dry-run to push changes.")
		return nil
	}

	// Save allocation lock after successful push
	if a.allocationLockRepo != nil && pushResult != nil && pushResult.UpdatedCount > 0 {
		lockKey := fmt.Sprintf("alloc::%s", sprint)
		lock := tasksdomain.NewSprintLock(project, lockKey, pushResult.UpdatedCount)
		if err := a.allocationLockRepo.SaveLock(bgCtx, lock); err != nil {
			fmt.Printf("Warning: failed to save allocation lock: %v\n", err)
		}
	}

	return nil
}

// initializeApp creates a new App instance with all dependencies
func initializeApp() (*App, error) {
	// Initialize shared configuration service first
	configRepo := configinfra.NewFileRepository(configDir)
	envProvider := configinfra.NewEnvironmentProvider()
	userInteraction := configinfra.NewCLIUserInteraction()
	initializeConfigUseCase := usecase.NewInitializeConfig(configRepo, envProvider, userInteraction)
	sharedConfigService := service.NewConfigService(configRepo)

	// Initialize repositories
	repoConfig := assetsinfra.RepositoryConfig{
		Directory: assetsDir,
		Filename:  assetsFile,
		FileMode:  0644,
		DirMode:   0755,
	}
	assetRepo := assetsinfra.NewJSONRepository(repoConfig)

	// Create ID generator
	idGenerator := assetid.NewHashIDGenerator()

	// Try to use shared configuration, fallback to legacy if config doesn't exist
	var assetService assetsapp.AssetService
	if configExists, _ := sharedConfigService.ConfigExists(); configExists {
		assetService = assetsapp.NewAssetService(assetRepo, sharedConfigService, idGenerator)
	} else {
		assetService = assetsapp.NewAssetServiceLegacy(assetRepo, idGenerator)
	}

	// Initialize task repositories with graceful fallback
	var jiraRepo taskports.TaskRepository
	var err error

	if configExists, _ := sharedConfigService.ConfigExists(); configExists {
		jiraRepo, err = jira.NewRepository(sharedConfigService)
		if err != nil {
			// Fallback to legacy if shared config fails
			jiraRepo, err = jira.NewRepositoryLegacy()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize Jira repository: %v", err)
			}
		}
	} else {
		jiraRepo, err = jira.NewRepositoryLegacy()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Jira repository: %v", err)
		}
	}

	localRepo := storage.NewJSONStorage(tasksDir, tasksFile)

	// Initialize comprehensive classification system
	// Asset classifier for determining which asset a task belongs to
	assetClassifier := classifier.NewContentBasedAssetClassifier(assetRepo)

	// Work type classifier for determining capitalization category
	workTypeClassifier := classifier.NewBusinessRulesClassifier(assetRepo)

	// Embedding-based asset classifier for comparison mode (optional, uses Ollama)
	llamaConfig := llama.DefaultConfig()
	var llmClassifier taskports.AssetClassifier
	if llamaConfig.BaseURL != "" {
		llamaClient, llamaErr := llama.NewClient(llamaConfig)
		if llamaErr == nil {
			embeddingService := classifier.NewOllamaEmbeddingAdapter(llamaClient, "nomic-embed-text")
			embeddingStorePath := filepath.Join(configDir, "embeddings.json")
			embeddingStore, storeErr := classifier.NewEmbeddingStore(embeddingStorePath)
			if storeErr == nil {
				llmClassifier = classifier.NewEmbeddingAssetClassifier(embeddingService, assetRepo, embeddingStore)
			}
		}
	}

	// Comprehensive classification chain with subtask inheritance and optional LLM support
	classificationChain := classifier.NewComprehensiveClassificationChainWithLLM(assetClassifier, workTypeClassifier, llmClassifier)

	// Create adapter to bridge comprehensive results with existing use case interface
	taskClassifier := classifier.NewComprehensiveClassifierAdapter(classificationChain)

	userInput := cliui.NewUserInput()

	// Initialize sprint service first (needed for SprintResolver)
	jiraAdapter, err := sprintinfra.NewJiraAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Jira adapter: %v", err)
	}
	sprintService := sprintapp.NewSprintService(jiraAdapter)

	// Create SprintResolver with interactive selection capability
	sprintResolver := tasksusecase.NewSprintResolver(jiraAdapter, userInput)

	// Initialize sprint lock storage
	sprintLockStorage := storage.NewSprintLockStorage(tasksDir, "sprint_locks.json")

	// Initialize task service with SprintResolver
	taskService := tasksapp.NewTasksService(jiraRepo, localRepo, taskClassifier, userInput, assetService, sprintResolver, sprintLockStorage)

	// Initialize config service for CLI
	configService := &configServiceImpl{
		initializeConfig: initializeConfigUseCase,
		configService:    sharedConfigService,
	}

	// Initialize investment service
	costModelRepo := investmentinfra.NewCostModelJSONRepository(configDir)
	allocationProvider := investmentinfra.NewTimeAllocationAdapter()
	investmentRepo := investmentinfra.NewInvestmentJSONRepository(configDir)
	investmentService := investmentservice.NewInvestmentService(costModelRepo, allocationProvider, investmentRepo)

	// Initialize team resolver
	teamConfig, err := sharedConfigService.GetTeamConfig()
	if err != nil {
		// Create empty team config if not exists
		teamConfig, _ = domain.NewTeamConfig(make(map[string][]string))
	}
	teamResolver := configapp.NewTeamResolverService(teamConfig)

	// Initialize allocation lock storage (separate from classification locks)
	allocationLockStorage := storage.NewSprintLockStorage(tasksDir, "sprint_allocation_locks.json")

	app := NewApp(assetService, taskService, sprintService)
	app.configService = configService
	app.investmentService = investmentService
	app.teamResolver = teamResolver
	app.taskRepo = jiraRepo
	app.taskClassifier = taskClassifier
	app.allocationLockRepo = allocationLockStorage
	return app, nil
}

func main() {
	// Handle version command without full initialization
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("assetcap version %s\n", version)
		if commit != "" {
			fmt.Printf("commit: %s\n", commit)
		}
		if date != "" {
			fmt.Printf("built: %s\n", date)
		}
		return
	}

	// Handle help command without full initialization
	if len(os.Args) <= 1 || os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help" {
		showHelp()
		return
	}

	// Initialize app only for business commands
	app, err := initializeApp()
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func showHelp() {
	fmt.Println("AssetCap - Digital Asset Capitalization Tool")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("   assetcap [global options] command [command options] [arguments...]")
	fmt.Println()
	fmt.Println("VERSION:")
	fmt.Printf("   %s\n", version)
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("   version     Show version information")
	fmt.Println("   completion  Generate shell completion scripts")
	fmt.Println("   config      Configure application settings")
	fmt.Println("   assets      Manage digital assets")
	fmt.Println("   tasks       Manage tasks and classification")
	fmt.Println("   sprint      Manage sprint-related operations")
	fmt.Println("   investment  Calculate investment costs")
	fmt.Println("   deployments Manage deployment tracking")
	fmt.Println("   console     Start AI-powered interactive console")
	fmt.Println("   help        Show this help message")
	fmt.Println()
	fmt.Println("GLOBAL OPTIONS:")
	fmt.Println("   --help, -h  Show help")
	fmt.Println()
	fmt.Println("For detailed command help, use: assetcap [command] --help")
}
