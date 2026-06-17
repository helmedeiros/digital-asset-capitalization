package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	deploymentsapp "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/application"
	deploymentsusecase "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/application/usecase"
	deploymentsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
	deploymentsports "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
	deploymentsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/infrastructure"
)

const (
	deploymentsDir  = ".assetcap"
	deploymentsFile = "deployments.json"
)

// createDeploymentCommands builds the `deployments` CLI command. The
// Action closures delegate to named methods so each handler is
// unit-testable against stubs.
//
// On first call, this lazily wires the deployment service + repo into
// App fields so the Action methods can find them. Tests can pre-set
// those fields on a bare App and skip this initializer entirely.
func (a *App) createDeploymentCommands() *cli.Command {
	a.ensureDeploymentService()
	return &cli.Command{
		Name:  "deployments",
		Usage: "Manage deployment tracking",
		Subcommands: []*cli.Command{
			{
				Name:  "record",
				Usage: "Record a new deployment",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: "tasks", Usage: "Task keys deployed (comma-separated or multiple flags)", Required: true},
					&cli.StringFlag{Name: "env", Usage: "Deployment environment (production, staging, qa, development)", Required: true},
					&cli.StringFlag{Name: "version", Usage: "Version or tag deployed", Required: true},
					&cli.StringFlag{Name: "deployed-by", Usage: "User or system that deployed (optional)"},
					&cli.StringFlag{Name: "commit", Usage: "Git commit SHA (optional)"},
					&cli.StringFlag{Name: "pipeline-id", Usage: "CI/CD pipeline ID (optional)"},
					&cli.StringFlag{Name: "pipeline-url", Usage: "CI/CD pipeline URL (optional)"},
					&cli.StringFlag{Name: "notes", Usage: "Additional notes (optional)"},
				},
				Action: a.deploymentsRecordAction,
			},
			{
				Name:  "list",
				Usage: "List deployments",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "task", Usage: "Filter by task key"},
					&cli.StringFlag{Name: "env", Usage: "Filter by environment"},
					&cli.IntFlag{Name: "limit", Usage: "Limit number of results", Value: 20},
				},
				Action: a.deploymentsListAction,
			},
			{
				Name:  "history",
				Usage: "Show deployment history for an asset",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
					&cli.IntFlag{Name: "limit", Usage: "Limit number of results", Value: 10},
				},
				Action: a.deploymentsHistoryAction,
			},
			{
				Name:  "timeline",
				Usage: "Show deployment timeline for a time range",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)", Required: true},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)", Required: true},
					&cli.StringFlag{Name: "env", Usage: "Filter by environment"},
					&cli.BoolFlag{Name: "resolve-assets", Usage: "Resolve and display affected assets"},
				},
				Action: a.deploymentsTimelineAction,
			},
			{
				Name:  "mock",
				Usage: "Generate mock deployment data for testing",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "count", Usage: "Number of deployments to generate", Value: 10},
					&cli.StringFlag{Name: "from", Usage: "Start date for mock data (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date for mock data (YYYY-MM-DD)"},
					&cli.BoolFlag{Name: "sample-file", Usage: "Generate sample mock_deployments.json file instead"},
				},
				Action: a.deploymentsMockAction,
			},
		},
	}
}

// ensureDeploymentService lazily initializes deploymentService and
// deploymentRepo on first use. Tests can pre-populate either field on
// a bare *App to inject stubs; in that case this is a no-op.
func (a *App) ensureDeploymentService() {
	if a.deploymentRepo == nil {
		a.deploymentRepo = deploymentsinfra.NewJSONRepository(deploymentsinfra.JSONRepositoryConfig{
			Directory: deploymentsDir,
			Filename:  deploymentsFile,
		})
	}
	if a.deploymentService == nil {
		assetResolver := deploymentsinfra.NewAssetResolverAdapter(a.taskRepo, a.taskClassifier)
		a.deploymentService = deploymentsapp.NewDeploymentService(a.deploymentRepo, assetResolver)
	}
}

// deploymentsRecordAction backs `assetcap deployments record`.
func (a *App) deploymentsRecordAction(c *cli.Context) error {
	ctx := context.Background()

	env := deploymentsdomain.Environment(c.String("env"))
	if env != deploymentsdomain.EnvironmentProduction &&
		env != deploymentsdomain.EnvironmentStaging &&
		env != deploymentsdomain.EnvironmentQA &&
		env != deploymentsdomain.EnvironmentDevelopment {
		return fmt.Errorf("invalid environment: %s", c.String("env"))
	}

	taskKeys := c.StringSlice("tasks")
	if len(taskKeys) == 1 && strings.Contains(taskKeys[0], ",") {
		taskKeys = strings.Split(taskKeys[0], ",")
	}

	var metadata *deploymentsdomain.DeploymentMetadata
	if c.String("pipeline-id") != "" || c.String("pipeline-url") != "" || c.String("notes") != "" {
		metadata = &deploymentsdomain.DeploymentMetadata{
			PipelineID:  c.String("pipeline-id"),
			PipelineURL: c.String("pipeline-url"),
			Notes:       c.String("notes"),
		}
	}

	useCase := deploymentsusecase.NewRecordDeploymentUseCase(a.deploymentService)
	deployment, err := useCase.Execute(ctx, deploymentsapp.RecordDeploymentInput{
		TaskKeys:    taskKeys,
		Environment: env,
		Version:     c.String("version"),
		DeployedBy:  c.String("deployed-by"),
		CommitSHA:   c.String("commit"),
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to record deployment: %w", err)
	}

	fmt.Printf("✅ Deployment recorded successfully\n")
	fmt.Printf("ID: %s\n", deployment.ID)
	fmt.Printf("Environment: %s\n", deployment.Environment)
	fmt.Printf("Version: %s\n", deployment.Version)
	fmt.Printf("Tasks: %s\n", strings.Join(deployment.TaskKeys, ", "))
	fmt.Printf("Deployed at: %s\n", deployment.DeployedAt.Format(time.RFC3339))

	return nil
}

// deploymentsListAction backs `assetcap deployments list`.
func (a *App) deploymentsListAction(c *cli.Context) error {
	ctx := context.Background()

	useCase := deploymentsusecase.NewGetDeploymentHistoryUseCase(a.deploymentService)

	filter := deploymentsusecase.HistoryFilter{
		TaskKey: c.String("task"),
		Limit:   c.Int("limit"),
	}

	if c.String("env") != "" {
		env := deploymentsdomain.Environment(c.String("env"))
		filter.Environment = &env
	}

	deployments, err := useCase.Execute(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments) == 0 {
		fmt.Println("No deployments found")
		return nil
	}

	fmt.Printf("\n📦 DEPLOYMENTS\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	for _, dep := range deployments {
		statusIcon := "✅"
		if dep.Status == deploymentsdomain.DeploymentStatusFailed {
			statusIcon = "❌"
		} else if dep.Status == deploymentsdomain.DeploymentStatusRolledBack {
			statusIcon = "↩️"
		} else if dep.Status == deploymentsdomain.DeploymentStatusInProgress {
			statusIcon = "🔄"
		}

		fmt.Printf("%s [%s] %s - %s\n",
			statusIcon,
			dep.DeployedAt.Format("2006-01-02 15:04"),
			dep.Environment,
			dep.Version)
		fmt.Printf("   Tasks: %s\n", strings.Join(dep.TaskKeys, ", "))
		if dep.CommitSHA != "" {
			fmt.Printf("   Commit: %s\n", dep.CommitSHA)
		}
		if dep.DeployedBy != "" {
			fmt.Printf("   Deployed by: %s\n", dep.DeployedBy)
		}
		fmt.Println()
	}

	return nil
}

// deploymentsHistoryAction backs `assetcap deployments history`.
func (a *App) deploymentsHistoryAction(c *cli.Context) error {
	ctx := context.Background()

	useCase := deploymentsusecase.NewGetDeploymentHistoryUseCase(a.deploymentService)

	deployments, err := useCase.Execute(ctx, deploymentsusecase.HistoryFilter{
		AssetName: c.String("asset"),
		Limit:     c.Int("limit"),
	})
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	if len(deployments) == 0 {
		fmt.Printf("No deployments found for asset: %s\n", c.String("asset"))
		return nil
	}

	fmt.Printf("\n📦 DEPLOYMENT HISTORY FOR: %s\n", c.String("asset"))
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	for _, dep := range deployments {
		fmt.Printf("[%s] %s - %s (%s)\n",
			dep.DeployedAt.Format("2006-01-02 15:04"),
			dep.Environment,
			dep.Version,
			dep.Status)

		if len(dep.ResolvedAssets) > 0 {
			fmt.Printf("   Affected Assets:\n")
			for _, asset := range dep.ResolvedAssets {
				fmt.Printf("   - %s (%d tasks)\n", asset.Name, asset.TaskCount)
			}
		}

		fmt.Printf("   Tasks: %s\n", strings.Join(dep.TaskKeys, ", "))
		fmt.Println()
	}

	return nil
}

// deploymentsTimelineAction backs `assetcap deployments timeline`.
func (a *App) deploymentsTimelineAction(c *cli.Context) error {
	ctx := context.Background()

	from, err := time.Parse("2006-01-02", c.String("from"))
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}

	to, err := time.Parse("2006-01-02", c.String("to"))
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}
	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	input := deploymentsusecase.TimelineInput{
		From:          from,
		To:            to,
		ResolveAssets: c.Bool("resolve-assets"),
	}

	if c.String("env") != "" {
		env := deploymentsdomain.Environment(c.String("env"))
		input.Environment = &env
	}

	useCase := deploymentsusecase.NewGetDeploymentsTimelineUseCase(a.deploymentService)
	output, err := useCase.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to get deployment timeline: %w", err)
	}

	fmt.Printf("\n📅 DEPLOYMENTS TIMELINE: %s\n", output.Period)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	if len(output.Timeline) == 0 {
		fmt.Println("No deployments found in this time range")
		return nil
	}

	for _, entry := range output.Timeline {
		fmt.Printf("%s (%s)\n", entry.Date.Format("2006-01-02"), entry.Date.Format("Monday"))
		fmt.Println(strings.Repeat("-", 60))

		for _, dep := range entry.Deployments {
			statusIcon := "✓"
			if dep.Status == deploymentsdomain.DeploymentStatusFailed {
				statusIcon = "✗"
			} else if dep.Status == deploymentsdomain.DeploymentStatusRolledBack {
				statusIcon = "↩"
			}

			fmt.Printf("[%s] %s - %s (%s)\n",
				dep.DeployedAt.Format("15:04"),
				strings.ToUpper(string(dep.Environment)),
				dep.Version,
				dep.CommitSHA[:7])

			if c.Bool("resolve-assets") && len(dep.ResolvedAssets) > 0 {
				assetNames := make([]string, 0, len(dep.ResolvedAssets))
				for _, asset := range dep.ResolvedAssets {
					assetNames = append(assetNames, fmt.Sprintf("%s (%d tasks)", asset.Name, asset.TaskCount))
				}
				fmt.Printf("  Resolved Assets: %s\n", strings.Join(assetNames, ", "))
			}

			fmt.Printf("  Tasks: %s\n", strings.Join(dep.TaskKeys, ", "))
			if dep.DeployedBy != "" {
				fmt.Printf("  Deployed by: %s\n", dep.DeployedBy)
			}
			fmt.Printf("  Status: %s %s\n", statusIcon, dep.Status)
			fmt.Println()
		}
		fmt.Println()
	}

	if output.Statistics != nil {
		fmt.Println("Summary:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Total Deployments: %d\n", output.Statistics.TotalDeployments)

		for env, count := range output.Statistics.ByEnvironment {
			envName := strings.ToUpper(env[:1]) + env[1:]
			fmt.Printf("%s: %d, ", envName, count)
		}
		fmt.Println()

		fmt.Printf("Unique Tasks Deployed: %d\n", len(output.Statistics.UniqueTaskKeys))
	}

	return nil
}

// deploymentsMockAction backs `assetcap deployments mock`.
func (a *App) deploymentsMockAction(c *cli.Context) error {
	ctx := context.Background()

	if c.Bool("sample-file") {
		err := deploymentsinfra.GenerateSampleMockFile(ctx, "mock_deployments.json")
		if err != nil {
			return fmt.Errorf("failed to generate sample file: %w", err)
		}
		fmt.Println("✅ Sample mock_deployments.json file generated successfully")
		return nil
	}

	provider := deploymentsinfra.NewMockDataProvider(a.deploymentRepo)

	config := deploymentsinfra.DefaultMockDataConfig()
	config.Count = c.Int("count")

	if c.String("from") != "" {
		from, err := time.Parse("2006-01-02", c.String("from"))
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
		config.StartDate = from
	}

	if c.String("to") != "" {
		to, err := time.Parse("2006-01-02", c.String("to"))
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
		config.EndDate = to
	}

	err := provider.GenerateMockDeployments(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to generate mock deployments: %w", err)
	}

	fmt.Printf("✅ Generated %d mock deployments successfully\n", config.Count)
	return nil
}

// Ensure deploymentsports stays imported even if usage migrates.
var _ deploymentsports.DeploymentRepository
