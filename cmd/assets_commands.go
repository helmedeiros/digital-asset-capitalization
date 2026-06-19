package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	assetsusecase "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase"
	assetsinfra "github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	configinfra "github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
)

// createAssetsCommand builds the `assets` CLI command with all its
// subcommands (create/list/sync/publish/update/show/enrich/keywords/
// sync-and-enrich/sync-contributors/documentation/tasks/teams etc.).
// Extracted from cmd/main.go.
func (a *App) createAssetsCommand() *cli.Command {
	return &cli.Command{
		Name:  "assets",
		Usage: "Manage digital assets",
		Subcommands: []*cli.Command{
			{
				Name:   "create",
				Usage:  "Create a new asset",
				Action: a.assetsCreateAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name", Required: true},
					&cli.StringFlag{Name: "description", Usage: "Asset description", Required: true},
				},
			},
			{
				Name:   "delete",
				Usage:  "Delete an existing asset",
				Action: a.assetsDeleteAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name", Required: true},
					&cli.BoolFlag{Name: "delete-page", Usage: "Also delete the associated Confluence page"},
				},
			},
			{
				Name:   "list",
				Usage:  "List all assets",
				Action: a.assetsListAction,
			},
			{
				Name:   "sync",
				Usage:  "Sync assets from Confluence",
				Action: a.assetsSyncAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "space", Usage: "Confluence space key(s). Single: 'MZN', Multiple: 'MZN,CAP,DOC', All: '*' or omit", Required: false},
					&cli.StringFlag{Name: "label", Usage: "Filter pages by label (e.g. cap-asset)", Required: true},
					&cli.BoolFlag{Name: "debug", Usage: "Enable debug logging", Value: false},
				},
			},
			{
				Name:   "publish",
				Usage:  "Publish an asset to Confluence as a new page",
				Action: a.assetsPublishAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name to publish", Required: true},
					&cli.StringFlag{Name: "space", Usage: "Confluence space key (e.g., Conversion)", Required: true},
					&cli.StringFlag{Name: "parent-page", Usage: "Parent page ID to create under (overrides team config)"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Preview without creating the page", Value: false},
					&cli.BoolFlag{Name: "debug", Usage: "Enable debug output", Value: false},
				},
			},
			{
				Name:   "update-confluence",
				Usage:  "Update an existing Confluence page with the asset's current content",
				Action: a.assetsUpdateConfluenceAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name to update", Required: true},
					&cli.BoolFlag{Name: "dry-run", Usage: "Preview without updating the page", Value: false},
					&cli.BoolFlag{Name: "debug", Usage: "Enable debug output", Value: false},
				},
			},
			{
				Name:   "update",
				Usage:  "Update an asset's description",
				Action: a.assetsUpdateAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name", Required: true},
					&cli.StringFlag{Name: "description", Usage: "New asset description", Required: true},
					&cli.StringFlag{Name: "why", Usage: "Why are we doing this?", Required: true},
					&cli.StringFlag{Name: "benefits", Usage: "Economic benefits", Required: true},
					&cli.StringFlag{Name: "how", Usage: "How it works?", Required: true},
					&cli.StringFlag{Name: "metrics", Usage: "How do we judge success?", Required: true},
				},
			},
			{
				Name:   "show",
				Usage:  "Show detailed information about an asset",
				Action: a.assetsShowAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name", Required: true},
				},
			},
			{
				Name:  "documentation",
				Usage: "Manage asset documentation",
				Subcommands: []*cli.Command{
					{
						Name:   "update",
						Usage:  "Mark asset documentation as updated",
						Action: a.assetsDocUpdateAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
						},
					},
				},
			},
			{
				Name:  "tasks",
				Usage: "Manage asset tasks",
				Subcommands: []*cli.Command{
					{
						Name:   "increment",
						Usage:  "Increment task count for an asset",
						Action: a.assetsTasksIncrementAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
						},
					},
					{
						Name:   "decrement",
						Usage:  "Decrement task count for an asset",
						Action: a.assetsTasksDecrementAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
						},
					},
				},
			},
			{
				Name:   "enrich",
				Usage:  "Enrich asset fields using LLaMA 3",
				Action: a.assetsEnrichAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name or ID", Required: true},
					&cli.StringFlag{Name: "field", Usage: "Field to enrich (e.g., description)", Required: true},
				},
			},
			{
				Name:   "keywords",
				Usage:  "Generate keywords for an asset using LLaMA 3",
				Action: a.assetsKeywordsAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Asset name or ID", Required: true},
				},
			},
			{
				Name:  "sync-and-enrich",
				Usage: "Sync assets from Confluence and enrich them with keywords and fields using AI",
				Action: func(ctx *cli.Context) error {
					spaceKey := ctx.String("space")
					label := ctx.String("label")
					debug := ctx.Bool("debug")
					enrichKeywords := ctx.Bool("keywords")
					enrichFields := ctx.StringSlice("fields")
					dryRun := ctx.Bool("dry-run")
					maxConcurrent := ctx.Int("max-concurrent")
					if maxConcurrent < 1 {
						maxConcurrent = 1
					}
					// field-filter is reserved for the future bulk use case path;
					// the inline enrichment here always touches every synced asset.
					_ = ctx.String("field-filter")

					// Validate required parameters
					if label == "" {
						return fmt.Errorf("label is required")
					}

					fmt.Printf("Starting sync-and-enrich workflow...\n")
					fmt.Printf("Space: %s, Label: %s, Keywords: %v, Fields: %v, MaxConcurrent: %d\n", spaceKey, label, enrichKeywords, enrichFields, maxConcurrent)

					if dryRun {
						fmt.Printf("DRY RUN: Would sync assets and enrich with keywords=%v, fields=%v\n", enrichKeywords, enrichFields)
						return nil
					}

					// Step 1: Sync assets
					fmt.Printf("Step 1: Syncing assets from Confluence...\n")
					result, err := a.assetService.SyncFromConfluence(spaceKey, label, debug)
					if err != nil {
						return fmt.Errorf("failed to sync assets: %w", err)
					}

					fmt.Printf("Synced %d assets\n", len(result.SyncedAssets))

					// Step 2: Enrich keywords if requested
					if enrichKeywords && len(result.SyncedAssets) > 0 {
						fmt.Printf("Step 2: Generating keywords for synced assets (max %d in flight)...\n", maxConcurrent)
						eg := new(errgroup.Group)
						eg.SetLimit(maxConcurrent)
						for _, asset := range result.SyncedAssets {
							asset := asset
							eg.Go(func() error {
								if err := a.assetService.GenerateKeywords(asset.Name); err != nil {
									fmt.Printf("Warning: Failed to generate keywords for %s: %v\n", asset.Name, err)
								} else {
									fmt.Printf("Generated keywords for: %s\n", asset.Name)
								}
								return nil
							})
						}
						_ = eg.Wait()
					}

					// Step 3: Enrich fields if requested
					if len(enrichFields) > 0 && len(result.SyncedAssets) > 0 {
						fmt.Printf("Step 3: Enriching fields %v for synced assets (max %d in flight)...\n", enrichFields, maxConcurrent)
						eg := new(errgroup.Group)
						eg.SetLimit(maxConcurrent)
						for _, asset := range result.SyncedAssets {
							for _, field := range enrichFields {
								asset, field := asset, field
								eg.Go(func() error {
									if err := a.assetService.EnrichAsset(asset.Name, field); err != nil {
										fmt.Printf("Warning: Failed to enrich %s field for %s: %v\n", field, asset.Name, err)
									} else {
										fmt.Printf("Enriched %s field for: %s\n", field, asset.Name)
									}
									return nil
								})
							}
						}
						_ = eg.Wait()
					}

					fmt.Printf("Sync-and-enrich workflow completed successfully!\n")
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "space",
						Usage:    "Confluence space key(s). Single: 'MZN', Multiple: 'MZN,CAP,DOC', All: '*' or omit",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "label",
						Usage:    "Confluence label to filter by (e.g., 'cap-asset')",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "debug",
						Usage: "Enable debug output",
					},
					&cli.BoolFlag{
						Name:  "keywords",
						Usage: "Generate keywords for synced assets using AI",
					},
					&cli.StringSliceFlag{
						Name:  "fields",
						Usage: "Fields to enrich using AI (description, why, benefits, how, metrics). Can be specified multiple times",
					},
					&cli.StringFlag{
						Name:  "field-filter",
						Usage: "Filter for field enrichment: 'all', 'missing-fields', 'empty-fields'",
						Value: "missing-fields",
					},
					&cli.IntFlag{
						Name:  "max-concurrent",
						Usage: "Maximum concurrent AI operations",
						Value: 2,
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be done without making changes",
					},
				},
			},
			{
				Name:   "sync-contributors",
				Usage:  "Synchronize asset contributors from JIRA task assignments with optional filtering",
				Action: a.assetsSyncContributorsAction,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Preview changes without making modifications",
						Value: false,
					},
					&cli.IntFlag{
						Name:  "max-results",
						Usage: "Maximum number of JIRA tasks to analyze",
						Value: 1000,
					},
					&cli.StringFlag{
						Name:  "project",
						Usage: "Filter by JIRA project key (e.g., FN)",
					},
					&cli.StringFlag{
						Name:  "sprint",
						Usage: "Filter by sprint name (e.g., Penguins)",
					},
					&cli.StringFlag{
						Name:  "team",
						Usage: "Only sync assets that this team works on",
					},
					&cli.StringFlag{
						Name:  "asset",
						Usage: "Only sync contributors for this specific asset",
					},
				},
			},
			{
				Name:  "teams",
				Usage: "Manage asset team assignments",
				Subcommands: []*cli.Command{
					{
						Name:   "assign",
						Usage:  "Assign teams to an asset",
						Action: a.assetsTeamsAssignAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
							&cli.StringFlag{Name: "owner", Usage: "Owning team"},
							&cli.StringFlag{Name: "contributors", Usage: "Contributing teams (comma-separated)"},
						},
					},
					{
						Name:   "list",
						Usage:  "List asset team assignments",
						Action: a.assetsTeamsListAction,
					},
					{
						Name:   "show",
						Usage:  "Show team assignments for a specific asset",
						Action: a.assetsTeamsShowAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
						},
					},
					{
						Name:   "add-contributor",
						Usage:  "Add a contributing team to an asset",
						Action: a.assetsTeamsAddContributorAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
							&cli.StringFlag{Name: "team", Usage: "Team name to add as contributor", Required: true},
						},
					},
					{
						Name:   "remove-contributor",
						Usage:  "Remove a contributing team from an asset",
						Action: a.assetsTeamsRemoveContributorAction,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
							&cli.StringFlag{Name: "team", Usage: "Team name to remove as contributor", Required: true},
						},
					},
				},
			},
		},
	}
}

// assetsCreateAction backs `assetcap assets create`.
func (a *App) assetsCreateAction(ctx *cli.Context) error {
	name := ctx.String("name")
	description := ctx.String("description")
	if err := a.assetService.CreateAsset(name, description); err != nil {
		return err
	}
	fmt.Printf("Created asset: %s\n", name)
	return nil
}

// assetsDeleteAction backs `assetcap assets delete`.
func (a *App) assetsDeleteAction(ctx *cli.Context) error {
	name := ctx.String("name")
	deletePage := ctx.Bool("delete-page")
	if err := a.assetService.DeleteAsset(name, deletePage); err != nil {
		return err
	}
	fmt.Printf("Deleted asset: %s\n", name)
	if deletePage {
		fmt.Printf("Confluence page also deleted.\n")
	}
	return nil
}

// assetsListAction backs `assetcap assets list`.
func (a *App) assetsListAction(_ *cli.Context) error {
	assets, err := a.assetService.ListAssets()
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		fmt.Println("No assets found")
		return nil
	}
	fmt.Println("Assets:")
	for _, asset := range assets {
		fmt.Printf("- %s:\n", asset.Name)
		fmt.Printf("  Description: %s\n", asset.Description)
		fmt.Printf("  Why: %s\n", asset.Why)
		fmt.Printf("  Benefits: %s\n", asset.Benefits)
		fmt.Printf("  How: %s\n", asset.How)
		fmt.Printf("  Metrics: %s\n", asset.Metrics)
		if asset.DocLink != "" {
			fmt.Printf("  DocLink: %s\n", asset.DocLink)
		}
		if asset.OwningTeam != "" {
			fmt.Printf("  👤 Owner: %s\n", asset.OwningTeam)
		}
		if len(asset.ContributingTeams) > 0 {
			fmt.Printf("  🤝 Contributors: %s\n", strings.Join(asset.ContributingTeams, ", "))
		}
		fmt.Println()
	}
	return nil
}

// assetsUpdateAction backs `assetcap assets update`.
func (a *App) assetsUpdateAction(ctx *cli.Context) error {
	name := ctx.String("name")
	description := ctx.String("description")
	why := ctx.String("why")
	benefits := ctx.String("benefits")
	how := ctx.String("how")
	metrics := ctx.String("metrics")
	if err := a.assetService.UpdateAsset(name, description, why, benefits, how, metrics); err != nil {
		return err
	}
	fmt.Printf("✓ Updated asset: %s\n", name)
	return nil
}

// assetsShowAction backs `assetcap assets show`.
func (a *App) assetsShowAction(ctx *cli.Context) error {
	name := ctx.String("name")
	asset, err := a.assetService.GetAsset(name)
	if err != nil {
		return err
	}
	fmt.Printf("Asset: %s\n", asset.Name)
	fmt.Printf("Description: %s\n", asset.Description)
	fmt.Printf("Why: %s\n", asset.Why)
	fmt.Printf("Benefits: %s\n", asset.Benefits)
	fmt.Printf("How: %s\n", asset.How)
	fmt.Printf("Metrics: %s\n", asset.Metrics)
	fmt.Printf("Created: %s\n", asset.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", asset.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Task Count: %d\n", asset.AssociatedTaskCount)
	if len(asset.Keywords) > 0 {
		fmt.Printf("Keywords: %s\n", strings.Join(asset.Keywords, ", "))
	}
	if asset.DocLink != "" {
		fmt.Printf("DocLink: %s\n", asset.DocLink)
	}
	if asset.OwningTeam != "" {
		fmt.Printf("👤 Owner: %s\n", asset.OwningTeam)
	}
	if len(asset.ContributingTeams) > 0 {
		fmt.Printf("🤝 Contributors: %s\n", strings.Join(asset.ContributingTeams, ", "))
	}
	return nil
}

// assetsEnrichAction backs `assetcap assets enrich`.
func (a *App) assetsEnrichAction(ctx *cli.Context) error {
	name := ctx.String("name")
	field := ctx.String("field")
	if err := a.assetService.EnrichAsset(name, field); err != nil {
		return err
	}
	fmt.Printf("Enriched %s field for asset: %s\n", field, name)
	return nil
}

// assetsKeywordsAction backs `assetcap assets keywords`. Checks the
// asset exists before invoking the generator so a typo produces a
// clear error instead of a downstream stack from the LLM client.
func (a *App) assetsKeywordsAction(ctx *cli.Context) error {
	name := ctx.String("name")
	if _, err := a.assetService.GetAsset(name); err != nil {
		return fmt.Errorf("asset not found: %s", name)
	}
	if err := a.assetService.GenerateKeywords(name); err != nil {
		return err
	}
	fmt.Printf("Generated keywords for asset: %s\n", name)
	return nil
}

// assetsDocUpdateAction backs `assetcap assets documentation update`.
// Pre-checks the asset exists so a typo produces a clear "not found"
// instead of a misleading downstream error.
func (a *App) assetsDocUpdateAction(ctx *cli.Context) error {
	assetName := ctx.String("asset")
	if _, err := a.assetService.GetAsset(assetName); err != nil {
		return err
	}
	if err := a.assetService.UpdateDocumentation(assetName); err != nil {
		return err
	}
	fmt.Printf("Marked documentation as updated for asset %s\n", assetName)
	return nil
}

// assetsTasksIncrementAction backs `assetcap assets tasks increment`.
func (a *App) assetsTasksIncrementAction(ctx *cli.Context) error {
	assetName := ctx.String("asset")
	if _, err := a.assetService.GetAsset(assetName); err != nil {
		return err
	}
	if err := a.assetService.IncrementTaskCount(assetName); err != nil {
		return err
	}
	fmt.Printf("Incremented task count for asset %s\n", assetName)
	return nil
}

// assetsTasksDecrementAction backs `assetcap assets tasks decrement`.
func (a *App) assetsTasksDecrementAction(ctx *cli.Context) error {
	assetName := ctx.String("asset")
	if _, err := a.assetService.GetAsset(assetName); err != nil {
		return err
	}
	if err := a.assetService.DecrementTaskCount(assetName); err != nil {
		return err
	}
	fmt.Printf("Decremented task count for asset %s\n", assetName)
	return nil
}

// assetsTeamsAssignAction backs `assetcap assets teams assign`.
func (a *App) assetsTeamsAssignAction(ctx *cli.Context) error {
	assetName := ctx.String("asset")
	owningTeam := ctx.String("owner")
	contributingTeams := parseCommaSeparated(ctx.String("contributors"))

	if err := a.assetService.AssignTeam(assetName, owningTeam, contributingTeams); err != nil {
		return err
	}

	fmt.Printf("✓ Successfully assigned teams to asset '%s'\n", assetName)
	if owningTeam != "" {
		fmt.Printf("  Owner: %s\n", owningTeam)
	}
	if len(contributingTeams) > 0 {
		fmt.Printf("  Contributors: %s\n", strings.Join(contributingTeams, ", "))
	}
	return nil
}

// assetsTeamsListAction backs `assetcap assets teams list`.
func (a *App) assetsTeamsListAction(_ *cli.Context) error {
	assetTeams, err := a.assetService.GetAssetTeams()
	if err != nil {
		return err
	}

	if len(assetTeams) == 0 {
		fmt.Println("No team assignments found")
		return nil
	}

	fmt.Println("Asset Team Assignments:")
	fmt.Println("═══════════════════════════════════════")
	for _, info := range assetTeams {
		fmt.Printf("📦 %s\n", info.AssetName)
		if info.OwningTeam != "" {
			fmt.Printf("  👤 Owner: %s\n", info.OwningTeam)
		}
		if len(info.ContributingTeams) > 0 {
			fmt.Printf("  🤝 Contributors: %s\n", strings.Join(info.ContributingTeams, ", "))
		}
		fmt.Println()
	}
	return nil
}

// assetsTeamsShowAction backs `assetcap assets teams show`.
func (a *App) assetsTeamsShowAction(ctx *cli.Context) error {
	assetName := ctx.String("asset")
	info, err := a.assetService.GetAssetTeamInfo(assetName)
	if err != nil {
		return err
	}

	fmt.Printf("Team Assignments for '%s':\n", info.AssetName)
	fmt.Println("─────────────────────────────────")
	if info.OwningTeam != "" {
		fmt.Printf("👤 Owner: %s\n", info.OwningTeam)
	} else {
		fmt.Println("👤 Owner: Not assigned")
	}

	if len(info.ContributingTeams) > 0 {
		fmt.Printf("🤝 Contributors: %s\n", strings.Join(info.ContributingTeams, ", "))
	} else {
		fmt.Println("🤝 Contributors: None")
	}
	return nil
}

// assetsTeamsAddContributorAction backs `assetcap assets teams add-contributor`.
func (a *App) assetsTeamsAddContributorAction(ctx *cli.Context) error {
	assetName := ctx.String("asset")
	teamName := ctx.String("team")
	if err := a.assetService.AddContributingTeam(assetName, teamName); err != nil {
		return err
	}
	fmt.Printf("✓ Added '%s' as contributor to asset '%s'\n", teamName, assetName)
	return nil
}

// assetsTeamsRemoveContributorAction backs `assetcap assets teams remove-contributor`.
func (a *App) assetsTeamsRemoveContributorAction(ctx *cli.Context) error {
	assetName := ctx.String("asset")
	teamName := ctx.String("team")
	if err := a.assetService.RemoveContributingTeam(assetName, teamName); err != nil {
		return err
	}
	fmt.Printf("✓ Removed '%s' as contributor from asset '%s'\n", teamName, assetName)
	return nil
}

// assetsSyncAction backs `assetcap assets sync`. Special-cases the
// "no assets found with label" service error as an informational
// success (since that's expected when running against an empty space)
// while surfacing every other error to the caller.
func (a *App) assetsSyncAction(ctx *cli.Context) error {
	space := ctx.String("space")
	label := ctx.String("label")
	debug := ctx.Bool("debug")

	result, err := a.assetService.SyncFromConfluence(space, label, debug)
	if err != nil {
		if strings.Contains(err.Error(), "no assets found with label") {
			fmt.Println(err)
			return nil
		}
		return err
	}

	totalAssets := len(result.SyncedAssets) + len(result.NotSyncedAssets)
	fmt.Printf("Successfully synced %d/%d assets from Confluence\n", len(result.SyncedAssets), totalAssets)

	if len(result.NotSyncedAssets) > 0 {
		fmt.Printf("\nWarning: %d assets could not be synced due to missing information:\n", len(result.NotSyncedAssets))
		for _, asset := range result.NotSyncedAssets {
			fmt.Printf("\n- %s:\n", asset.Name)
			fmt.Printf("  Missing fields: %s\n", strings.Join(asset.MissingFields, ", "))
			fmt.Println("  Available fields:")
			for field, value := range asset.AvailableFields {
				if value != "" {
					fmt.Printf("    %s: %s\n", field, value)
				}
			}
		}
	}
	return nil
}

// assetsPublishAction backs `assetcap assets publish`.
func (a *App) assetsPublishAction(ctx *cli.Context) error {
	name := ctx.String("name")
	space := ctx.String("space")
	parentPage := ctx.String("parent-page")
	dryRun := ctx.Bool("dry-run")
	debug := ctx.Bool("debug")

	result, err := a.assetService.PublishToConfluence(context.Background(), name, space, parentPage, dryRun, debug)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would create page for asset '%s' in space '%s'\n", result.AssetName, result.SpaceKey)
		fmt.Printf("Labels to add: %v\n", result.Labels)
		fmt.Printf("\nPreview of page content:\n")
		fmt.Println("────────────────────────────────────────")
		fmt.Println(result.Preview)
		fmt.Println("────────────────────────────────────────")
		return nil
	}

	fmt.Printf("Successfully published asset '%s' to Confluence\n", result.AssetName)
	fmt.Printf("  Page ID: %s\n", result.PageID)
	fmt.Printf("  Space: %s\n", result.SpaceKey)
	fmt.Printf("  URL: %s\n", result.PageURL)
	fmt.Printf("  Labels: %v\n", result.Labels)
	if result.DocLinkSaved {
		fmt.Printf("  DocLink updated in asset\n")
	}
	return nil
}

// assetsUpdateConfluenceAction backs `assetcap assets update-confluence`.
func (a *App) assetsUpdateConfluenceAction(ctx *cli.Context) error {
	name := ctx.String("name")
	dryRun := ctx.Bool("dry-run")
	debug := ctx.Bool("debug")

	result, err := a.assetService.UpdateConfluencePage(context.Background(), name, dryRun, debug)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would update page for asset '%s'\n", result.AssetName)
		fmt.Printf("  Page ID: %s\n", result.PageID)
		fmt.Printf("  Space: %s\n", result.SpaceKey)
		fmt.Printf("\nPreview of page content:\n")
		fmt.Println("────────────────────────────────────────")
		fmt.Println(result.Preview)
		fmt.Println("────────────────────────────────────────")
		return nil
	}

	fmt.Printf("Successfully updated Confluence page for asset '%s'\n", result.AssetName)
	fmt.Printf("  Page ID: %s\n", result.PageID)
	fmt.Printf("  Space: %s\n", result.SpaceKey)
	fmt.Printf("  URL: %s\n", result.PageURL)
	return nil
}

// ensureSyncContributorsService lazily builds the JIRA + team-config
// + asset-repo + use-case graph via a.syncContributorsServiceFactory
// (which tests can override) and stashes the result on App. Mirrors
// ensureSyncTeamService; exposes the construction error because
// *assetsinfra.JiraQueryAdapter requires env vars and can fail.
func (a *App) ensureSyncContributorsService() error {
	if a.syncContributorsService != nil {
		return nil
	}
	factory := a.syncContributorsServiceFactory
	if factory == nil {
		factory = a.defaultSyncContributorsServiceFactory
	}
	svc, err := factory()
	if err != nil {
		return err
	}
	a.syncContributorsService = svc
	return nil
}

// defaultSyncContributorsServiceFactory wires the JIRA query
// adapter + team-config adapter + JSON asset repository + the use
// case the same way the original inline Action did.
func (a *App) defaultSyncContributorsServiceFactory() (SyncAssetContributorsService, error) {
	jiraQueryAdapter, err := assetsinfra.NewJiraQueryAdapter(a.configService)
	if err != nil {
		return nil, fmt.Errorf("failed to create JIRA query adapter: %v", err)
	}
	configRepo := configinfra.NewFileRepository(configDir)
	teamConfigAdapter := assetsinfra.NewTeamConfigAdapter(configRepo)
	assetRepo := assetsinfra.NewJSONRepository(assetsinfra.RepositoryConfig{
		Directory: assetsDir,
		Filename:  assetsFile,
		FileMode:  0644,
		DirMode:   0755,
	})
	return assetsusecase.NewSyncAssetContributorsFromJiraUseCase(
		assetRepo,
		jiraQueryAdapter,
		teamConfigAdapter,
	), nil
}

// assetsSyncContributorsAction backs `assetcap assets sync-contributors`.
func (a *App) assetsSyncContributorsAction(ctx *cli.Context) error {
	dryRun := ctx.Bool("dry-run")
	projectKey := ctx.String("project")
	sprintName := ctx.String("sprint")
	teamName := ctx.String("team")
	assetName := ctx.String("asset")

	if err := a.ensureSyncContributorsService(); err != nil {
		return err
	}

	input := assetsusecase.SyncContributorsInput{
		DryRun:     dryRun,
		MaxResults: ctx.Int("max-results"),
		ProjectKey: projectKey,
		SprintName: sprintName,
		TeamName:   teamName,
		AssetName:  assetName,
	}

	result, err := a.syncContributorsService.Execute(context.Background(), input)
	if err != nil {
		return fmt.Errorf("failed to sync contributors: %v", err)
	}

	if projectKey != "" || sprintName != "" || teamName != "" || assetName != "" {
		fmt.Printf("🎯 Filtered sync")
		if projectKey != "" {
			fmt.Printf(" project:%s", projectKey)
		}
		if sprintName != "" {
			fmt.Printf(" sprint:%s", sprintName)
		}
		if teamName != "" {
			fmt.Printf(" team:%s", teamName)
		}
		if assetName != "" {
			fmt.Printf(" asset:%s", assetName)
		}
		fmt.Println()
	}

	fmt.Printf("🔍 Analyzed %d JIRA tasks with asset labels\n", result.TotalTasks)
	fmt.Printf("📦 Processed %d assets\n", len(result.AssetsProcessed))

	if dryRun {
		fmt.Println("\n🔍 DRY RUN - No changes were made")
	} else {
		fmt.Printf("✅ Updated %d assets\n", result.AssetsUpdated)
	}

	for _, assetResult := range result.AssetsProcessed {
		fmt.Printf("\n📦 %s (analyzed %d tasks)\n", assetResult.AssetName, assetResult.TasksAnalyzed)

		if assetResult.Error != "" {
			fmt.Printf("  ❌ Error: %s\n", assetResult.Error)
			continue
		}

		if len(assetResult.TeamsFound) > 0 {
			fmt.Printf("  🔍 Teams found: %s\n", strings.Join(assetResult.TeamsFound, ", "))
		}

		if len(assetResult.CurrentContributors) > 0 {
			fmt.Printf("  👥 Current contributors: %s\n", strings.Join(assetResult.CurrentContributors, ", "))
		}

		if len(assetResult.NewContributors) > 0 {
			fmt.Printf("  ➕ New contributors: %s\n", strings.Join(assetResult.NewContributors, ", "))
		}

		if len(assetResult.RemovedContributors) > 0 {
			fmt.Printf("  ➖ Removed contributors: %s\n", strings.Join(assetResult.RemovedContributors, ", "))
		}

		if assetResult.Updated {
			fmt.Printf("  ✅ Updated\n")
		} else if !dryRun {
			fmt.Printf("  ⏸️  No changes needed\n")
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n⚠️  Errors encountered:\n")
		for _, syncErr := range result.Errors {
			fmt.Printf("  - %s\n", syncErr)
		}
	}

	return nil
}
