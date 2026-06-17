package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/migration"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/storage"
)

// createTasksCommand builds the `tasks` CLI command with its
// fetch/show/classify/inspect/migrate subcommands. Each Action
// closure delegates to a named method on *App so the handlers are
// unit-testable against stub services.
func (a *App) createTasksCommand() *cli.Command {
	return &cli.Command{
		Name:  "tasks",
		Usage: "Manage tasks from various platforms",
		Subcommands: []*cli.Command{
			{
				Name:   "fetch",
				Usage:  "Fetch tasks from a platform (e.g., Jira)",
				Action: a.tasksFetchAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN) - required when not using --key"},
					&cli.StringFlag{Name: "sprint", Usage: "Sprint name (e.g., Penguins) - required when not using --key"},
					&cli.StringFlag{Name: "key", Usage: "Task key (e.g., FN-1015) - fetch individual task"},
					&cli.StringFlag{Name: "platform", Usage: "Platform to fetch tasks from (e.g., jira)", Required: true},
				},
			},
			{
				Name:   "show",
				Usage:  "Show tasks for a project and sprint",
				Action: a.tasksShowAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project name"},
					&cli.StringFlag{Name: "sprint", Usage: "Sprint name"},
					&cli.StringFlag{Name: "asset", Usage: "Asset name or ID to filter tasks"},
				},
			},
			{
				Name:   "classify",
				Usage:  "Classify tasks for a specific project and sprint",
				Action: a.tasksClassifyAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN)", Required: true},
					&cli.StringFlag{Name: "sprint", Usage: "Sprint name (e.g., Penguins)", Required: true},
					&cli.StringFlag{Name: "platform", Usage: "Platform to classify tasks from (e.g., jira)", Required: true},
					&cli.BoolFlag{Name: "dry-run", Usage: "Preview classification without making changes", Value: false},
					&cli.BoolFlag{Name: "apply", Usage: "Write classifications back to Jira", Value: false},
					&cli.BoolFlag{Name: "force", Usage: "Override sprint classification lock (requires --apply)", Value: false},
					&cli.BoolFlag{Name: "with-llm", Usage: "Enable LLM comparison mode (dry-run only, requires Ollama)", Value: false},
				},
			},
			{
				Name:   "inspect",
				Usage:  "Inspect a specific task by its key",
				Action: a.tasksInspectAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "key", Usage: "Task key (e.g., FN-1015)", Required: true},
				},
			},
			{
				Name:   "migrate",
				Usage:  "Migrate sprint data from comma-separated strings to arrays",
				Action: a.tasksMigrateAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "file", Usage: "Path to tasks.json file (default: .assetcap/tasks.json)"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Preview migration without making changes", Value: false},
					&cli.BoolFlag{Name: "stats", Usage: "Show migration statistics without running migration", Value: false},
					&cli.BoolFlag{Name: "rollback", Usage: "Rollback previous migration using backup file", Value: false},
				},
			},
		},
	}
}

// tasksFetchAction backs `assetcap tasks fetch`.
func (a *App) tasksFetchAction(ctx *cli.Context) error {
	project := ctx.String("project")
	sprint := ctx.String("sprint")
	key := ctx.String("key")
	platform := ctx.String("platform")

	if key != "" {
		if project != "" || sprint != "" {
			return fmt.Errorf("when using --key, do not specify --project or --sprint")
		}
		if err := a.taskService.FetchTaskByKey(context.Background(), key, platform); err != nil {
			return err
		}
		fmt.Printf("✓ Successfully fetched task %s from %s\n", key, platform)
		return nil
	}

	if project == "" || sprint == "" {
		return fmt.Errorf("either --key or both --project and --sprint must be provided")
	}

	if a.teamResolver != nil && project != "" {
		resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
		if err != nil {
			return fmt.Errorf("unknown project or team nickname: %s", project)
		}
		project = resolvedProject
	}

	if err := a.taskService.FetchTasks(context.Background(), project, sprint, platform); err != nil {
		return err
	}
	fmt.Printf("✓ Successfully fetched tasks for project %s, sprint %s from %s\n", project, sprint, platform)
	return nil
}

// tasksShowAction backs `assetcap tasks show`. The --asset branch
// renders tasks filtered to a single asset; without --asset it
// requires --project and --sprint.
func (a *App) tasksShowAction(ctx *cli.Context) error {
	if asset := ctx.String("asset"); asset != "" {
		_, err := a.assetService.GetAsset(asset)
		if err != nil {
			return fmt.Errorf("asset not found: %s", asset)
		}

		tasks, err := a.taskService.GetTasksByAsset(ctx.Context, asset)
		if err != nil {
			return fmt.Errorf("failed to get tasks for asset %s: %w", asset, err)
		}

		fmt.Printf("Tasks for asset %s:\n", asset)
		fmt.Println("----------------------------------------")
		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		for _, task := range tasks {
			fmt.Printf("Key: %s\nType: %s\nSummary: %s\nStatus: %s\nEpic: %s\nWork Type: %s\nLabels: %v\n\n",
				task.Key, task.Type, task.Summary, task.Status, task.Epic, task.WorkType, task.Labels)
		}
		return nil
	}

	project := ctx.String("project")
	sprint := ctx.String("sprint")

	if project == "" || sprint == "" {
		return fmt.Errorf("both project and sprint flags are required")
	}

	tasks, err := a.taskService.GetTasks(ctx.Context, project, sprint)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	fmt.Printf("\nTasks for project %s and sprint %s:\n", project, sprint)
	fmt.Println("----------------------------------------")
	for _, task := range tasks {
		fmt.Printf("Key: %s\nType: %s\nSummary: %s\nStatus: %s\nEpic: %s\nWork Type: %s\nLabels: %v\n",
			task.Key, task.Type, task.Summary, task.Status, task.Epic, task.WorkType, task.Labels)
		if len(task.TPDBusinessUnits) > 0 {
			fmt.Printf("TPD Business Unit: %s\n", strings.Join(task.TPDBusinessUnits, ", "))
		}
		if task.WorkStream != "" {
			fmt.Printf("Work Stream: %s\n", task.WorkStream)
		}
		if task.EngineeringHours != nil {
			fmt.Printf("Engineering Hours: %.2f\n", *task.EngineeringHours)
		}
		fmt.Println()
	}
	return nil
}

// tasksClassifyAction backs `assetcap tasks classify`.
func (a *App) tasksClassifyAction(ctx *cli.Context) error {
	project := ctx.String("project")
	sprint := ctx.String("sprint")
	platform := ctx.String("platform")
	dryRun := ctx.Bool("dry-run")
	apply := ctx.Bool("apply")
	force := ctx.Bool("force")
	withLLM := ctx.Bool("with-llm")

	if a.teamResolver != nil && project != "" {
		resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
		if err != nil {
			return fmt.Errorf("unknown project or team nickname: %s", project)
		}
		project = resolvedProject
	}

	input := tasksdomain.ClassifyTasksInput{
		Project: project,
		Sprint:  sprint,
		DryRun:  dryRun,
		Apply:   apply,
		Force:   force,
		WithLLM: withLLM,
	}
	if err := a.taskService.ClassifyTasks(context.Background(), input); err != nil {
		return err
	}
	if dryRun {
		fmt.Printf("✓ Classification preview completed for project %s, sprint %s\n", project, sprint)
		fmt.Printf("  Use --apply to write classifications to %s\n", platform)
	} else if apply {
		fmt.Printf("✓ Successfully classified and applied labels to tasks for project %s, sprint %s\n", project, sprint)
		fmt.Printf("  Labels written to %s with work types and asset associations\n", platform)
	} else {
		fmt.Printf("✓ Successfully classified tasks for project %s, sprint %s\n", project, sprint)
		fmt.Printf("  Classifications saved locally. Use --apply to write to %s\n", platform)
	}
	return nil
}

// tasksInspectAction backs `assetcap tasks inspect`.
func (a *App) tasksInspectAction(ctx *cli.Context) error {
	key := ctx.String("key")
	if key == "" {
		return fmt.Errorf("task key is required")
	}

	task, err := a.taskService.GetTaskByKey(ctx.Context, key)
	if err != nil {
		return fmt.Errorf("failed to get task %s: %w", key, err)
	}

	if task == nil {
		fmt.Printf("Task %s not found\n", key)
		return nil
	}

	fmt.Printf("Task Details for %s:\n", key)
	fmt.Println("========================================")
	fmt.Printf("Key:           %s\n", task.Key)
	fmt.Printf("Type:          %s\n", task.Type)
	fmt.Printf("Summary:       %s\n", task.Summary)
	fmt.Printf("Status:        %s\n", task.Status)
	fmt.Printf("Project:       %s\n", task.Project)
	fmt.Printf("Sprint:        %s\n", task.Sprint)
	fmt.Printf("Epic:          %s\n", task.Epic)
	fmt.Printf("Work Type:     %s\n", task.WorkType)
	fmt.Printf("Priority:      %s\n", task.Priority)
	fmt.Printf("Platform:      %s\n", task.Platform)
	fmt.Printf("Labels:        %v\n", task.Labels)
	if len(task.TPDBusinessUnits) > 0 {
		fmt.Printf("TPD BU:        %s\n", strings.Join(task.TPDBusinessUnits, ", "))
	}
	if task.WorkStream != "" {
		fmt.Printf("Work Stream:   %s\n", task.WorkStream)
	}
	if task.EngineeringHours != nil {
		fmt.Printf("Eng. Hours:    %.2f\n", *task.EngineeringHours)
	}
	fmt.Printf("Created:       %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:       %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Version:       %d\n", task.Version)

	if task.Description != "" {
		fmt.Printf("Description:\n%s\n", task.Description)
	}

	return nil
}

// tasksMigrateAction backs `assetcap tasks migrate`.
func (a *App) tasksMigrateAction(ctx *cli.Context) error {
	filePath := ctx.String("file")
	if filePath == "" {
		filePath = migration.DefaultTasksFilePath()
	}

	dryRun := ctx.Bool("dry-run")
	stats := ctx.Bool("stats")
	rollback := ctx.Bool("rollback")

	storageDir := filepath.Dir(filePath)
	storageFile := filepath.Base(filePath)
	localStorage := storage.NewJSONStorage(storageDir, storageFile)
	backupDir := filepath.Join(storageDir, "backups")

	migrator := migration.NewSprintMigration(localStorage, backupDir)

	if rollback {
		if err := migrator.RollbackMigration(context.Background()); err != nil {
			return fmt.Errorf("failed to rollback migration: %w", err)
		}
		fmt.Printf("✓ Successfully rolled back migration for %s\n", filePath)
		return nil
	}

	if stats {
		result, err := migrator.GetMigrationStats(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get migration stats: %w", err)
		}

		fmt.Printf("Migration Statistics for %s:\n", filePath)
		fmt.Printf("========================================\n")
		fmt.Printf("Total tasks:      %d\n", result.TasksProcessed)
		fmt.Printf("Tasks to migrate: %d\n", result.Statistics["needs_migration"])
		fmt.Printf("Already migrated: %d\n", result.Statistics["already_migrated"])
		fmt.Printf("Migration %%:      %.1f%%\n", result.Statistics["migration_percentage"])
		return nil
	}

	if err := migrator.ValidateCompatibility(context.Background()); err != nil {
		return fmt.Errorf("migration compatibility check failed: %w", err)
	}

	result, err := migrator.MigrateToArrayFormat(context.Background(), dryRun)
	if err != nil {
		return fmt.Errorf("failed to migrate sprint data: %w", err)
	}

	fmt.Printf("Migration Results for %s:\n", filePath)
	fmt.Printf("========================================\n")
	fmt.Printf("Total tasks:       %d\n", result.TasksProcessed)
	fmt.Printf("Migrated tasks:    %d\n", result.TasksMigrated)
	fmt.Printf("Skipped tasks:     %d\n", result.TasksSkipped)
	if len(result.Errors) > 0 {
		fmt.Printf("Errors:            %d\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if dryRun {
		fmt.Printf("\n🔍 DRY RUN: No changes were made\n")
		fmt.Printf("   Run without --dry-run to apply changes\n")
	} else {
		if result.BackupCreated {
			fmt.Printf("Backup created:    %s\n", result.BackupPath)
		}
		fmt.Printf("\n✓ Migration completed successfully!\n")
		fmt.Printf("   Use --rollback to revert changes if needed\n")
	}

	return nil
}
