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
	deploymentsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/deployments/infrastructure"
)

const (
	deploymentsDir  = ".assetcap"
	deploymentsFile = "deployments.json"
)

// createDeploymentCommands creates the deployment-related CLI commands
func (a *App) createDeploymentCommands() *cli.Command {
	// Initialize deployment repository
	deploymentRepo := deploymentsinfra.NewJSONRepository(deploymentsinfra.JSONRepositoryConfig{
		Directory: deploymentsDir,
		Filename:  deploymentsFile,
	})

	// Initialize asset resolver using the app's task repository and classifier
	assetResolver := deploymentsinfra.NewAssetResolverAdapter(a.taskRepo, a.taskClassifier)

	// Initialize deployment service
	deploymentService := deploymentsapp.NewDeploymentService(deploymentRepo, assetResolver)

	return &cli.Command{
		Name:  "deployments",
		Usage: "Manage deployment tracking",
		Subcommands: []*cli.Command{
			{
				Name:  "record",
				Usage: "Record a new deployment",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "tasks",
						Usage:    "Task keys deployed (comma-separated or multiple flags)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "env",
						Usage:    "Deployment environment (production, staging, qa, development)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "version",
						Usage:    "Version/tag of the deployment",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "deployed-by",
						Usage: "Who/what deployed this release",
						Value: "manual",
					},
					&cli.StringFlag{
						Name:  "commit",
						Usage: "Git commit SHA",
					},
					&cli.StringFlag{
						Name:  "pipeline-id",
						Usage: "CI/CD pipeline ID",
					},
					&cli.StringFlag{
						Name:  "pipeline-url",
						Usage: "CI/CD pipeline URL",
					},
					&cli.StringFlag{
						Name:  "notes",
						Usage: "Additional notes about the deployment",
					},
				},
				Action: func(c *cli.Context) error {
					ctx := context.Background()

					// Parse environment
					env := deploymentsdomain.Environment(c.String("env"))
					if env != deploymentsdomain.EnvironmentProduction &&
						env != deploymentsdomain.EnvironmentStaging &&
						env != deploymentsdomain.EnvironmentQA &&
						env != deploymentsdomain.EnvironmentDevelopment {
						return fmt.Errorf("invalid environment: %s", c.String("env"))
					}

					// Collect all task keys
					taskKeys := c.StringSlice("tasks")
					if len(taskKeys) == 1 && strings.Contains(taskKeys[0], ",") {
						// Handle comma-separated input
						taskKeys = strings.Split(taskKeys[0], ",")
					}

					// Prepare metadata
					var metadata *deploymentsdomain.DeploymentMetadata
					if c.String("pipeline-id") != "" || c.String("pipeline-url") != "" || c.String("notes") != "" {
						metadata = &deploymentsdomain.DeploymentMetadata{
							PipelineID:  c.String("pipeline-id"),
							PipelineURL: c.String("pipeline-url"),
							Notes:       c.String("notes"),
						}
					}

					// Record deployment
					useCase := deploymentsusecase.NewRecordDeploymentUseCase(deploymentService)
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
				},
			},
			{
				Name:  "list",
				Usage: "List deployments",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "task",
						Usage: "Filter by task key",
					},
					&cli.StringFlag{
						Name:  "env",
						Usage: "Filter by environment",
					},
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Limit number of results",
						Value: 20,
					},
				},
				Action: func(c *cli.Context) error {
					ctx := context.Background()

					useCase := deploymentsusecase.NewGetDeploymentHistoryUseCase(deploymentService)

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
				},
			},
			{
				Name:  "history",
				Usage: "Show deployment history for an asset",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "asset",
						Usage:    "Asset name",
						Required: true,
					},
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Limit number of results",
						Value: 10,
					},
				},
				Action: func(c *cli.Context) error {
					ctx := context.Background()

					useCase := deploymentsusecase.NewGetDeploymentHistoryUseCase(deploymentService)

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
				},
			},
			{
				Name:  "timeline",
				Usage: "Show deployment timeline for a time range",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "from",
						Usage:    "Start date (YYYY-MM-DD)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "to",
						Usage:    "End date (YYYY-MM-DD)",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "env",
						Usage: "Filter by environment",
					},
					&cli.BoolFlag{
						Name:  "resolve-assets",
						Usage: "Resolve and display affected assets",
					},
				},
				Action: func(c *cli.Context) error {
					ctx := context.Background()

					// Parse dates
					from, err := time.Parse("2006-01-02", c.String("from"))
					if err != nil {
						return fmt.Errorf("invalid from date: %w", err)
					}

					to, err := time.Parse("2006-01-02", c.String("to"))
					if err != nil {
						return fmt.Errorf("invalid to date: %w", err)
					}
					// Set to end of day
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

					useCase := deploymentsusecase.NewGetDeploymentsTimelineUseCase(deploymentService)
					output, err := useCase.Execute(ctx, input)
					if err != nil {
						return fmt.Errorf("failed to get deployment timeline: %w", err)
					}

					// Display timeline
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

					// Display statistics
					if output.Statistics != nil {
						fmt.Println("Summary:")
						fmt.Println(strings.Repeat("-", 60))
						fmt.Printf("Total Deployments: %d\n", output.Statistics.TotalDeployments)

						for env, count := range output.Statistics.ByEnvironment {
							// Capitalize first letter of environment name
							envName := strings.ToUpper(env[:1]) + env[1:]
							fmt.Printf("%s: %d, ", envName, count)
						}
						fmt.Println()

						fmt.Printf("Unique Tasks Deployed: %d\n", len(output.Statistics.UniqueTaskKeys))
					}

					return nil
				},
			},
			{
				Name:  "mock",
				Usage: "Generate mock deployment data for testing",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "count",
						Usage: "Number of deployments to generate",
						Value: 10,
					},
					&cli.StringFlag{
						Name:  "from",
						Usage: "Start date for mock data (YYYY-MM-DD)",
					},
					&cli.StringFlag{
						Name:  "to",
						Usage: "End date for mock data (YYYY-MM-DD)",
					},
					&cli.BoolFlag{
						Name:  "sample-file",
						Usage: "Generate sample mock_deployments.json file instead",
					},
				},
				Action: func(c *cli.Context) error {
					ctx := context.Background()

					if c.Bool("sample-file") {
						// Generate sample file
						err := deploymentsinfra.GenerateSampleMockFile(ctx, "mock_deployments.json")
						if err != nil {
							return fmt.Errorf("failed to generate sample file: %w", err)
						}
						fmt.Println("✅ Sample mock_deployments.json file generated successfully")
						return nil
					}

					// Generate mock data in the repository
					provider := deploymentsinfra.NewMockDataProvider(deploymentRepo)

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
				},
			},
		},
	}
}
