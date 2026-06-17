package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	investmentdomain "github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// createInvestmentCommand builds the `investment` CLI command. The
// Action closures delegate to named methods so each handler is
// unit-testable against a stub investment service.
func (a *App) createInvestmentCommand() *cli.Command {
	return &cli.Command{
		Name:  "investment",
		Usage: "Calculate investment costs for digital assets",
		Subcommands: []*cli.Command{
			{
				Name:   "calculate",
				Usage:  "Calculate investment for an asset across sprints",
				Action: a.investmentCalculateAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "asset", Usage: "Asset name", Required: true},
					&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN)", Required: true},
					&cli.StringFlag{Name: "sprints", Usage: "Comma-separated list of sprint names (e.g., 'Spiderman,Onça')"},
				},
			},
			{
				Name:   "sprint",
				Usage:  "Calculate investment for a specific sprint",
				Action: a.investmentSprintAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN)", Required: true},
					&cli.StringFlag{Name: "sprint", Usage: "Sprint name (e.g., Spiderman)", Required: true},
					&cli.StringFlag{Name: "start-date", Usage: "Sprint start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "end-date", Usage: "Sprint end date (YYYY-MM-DD)"},
				},
			},
			{
				Name:   "list",
				Usage:  "List all saved investment calculations",
				Action: a.investmentListAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Filter by project key (e.g., FN)"},
				},
			},
			{
				Name:   "init-cost-model",
				Usage:  "Initialize cost model for a project",
				Action: a.investmentInitCostModelAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN)", Required: true},
				},
			},
			{
				Name:   "set-engineer-rate",
				Usage:  "Set hourly rate for a specific engineer",
				Action: a.investmentSetEngineerRateAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN)", Required: true},
					&cli.StringFlag{Name: "engineer", Usage: "Engineer name (e.g., 'Santhosh Balakrishnan')", Required: true},
					&cli.StringFlag{Name: "rate", Usage: "Hourly rate (e.g., 75.50)", Required: true},
					&cli.StringFlag{Name: "level", Usage: "Engineer level (junior, mid, senior, staff, principal)", Value: "mid"},
				},
			},
			{
				Name:   "show-rates",
				Usage:  "Show current engineer rates for a project",
				Action: a.investmentShowRatesAction,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project key (e.g., FN)", Required: true},
				},
			},
		},
	}
}

// investmentCalculateAction backs `assetcap investment calculate`.
func (a *App) investmentCalculateAction(ctx *cli.Context) error {
	if a.investmentService == nil {
		return fmt.Errorf("investment service not available")
	}

	assetName := ctx.String("asset")
	project := ctx.String("project")
	sprintsStr := ctx.String("sprints")

	if a.teamResolver != nil && project != "" {
		resolvedProject, err := a.teamResolver.ResolveProjectIdentifier(project)
		if err != nil {
			return fmt.Errorf("unknown project or team nickname: %s", project)
		}
		project = resolvedProject
	}

	var sprints []string
	if sprintsStr != "" {
		sprints = strings.Split(sprintsStr, ",")
		for i, sprint := range sprints {
			sprints[i] = strings.TrimSpace(sprint)
		}
	}

	investment, err := a.investmentService.CalculateAssetInvestment(ctx.Context, assetName, project, sprints)
	if err != nil {
		return fmt.Errorf("failed to calculate investment: %w", err)
	}

	fmt.Printf("💰 Investment Calculation for '%s'\n", investment.AssetName)
	fmt.Printf("════════════════════════════════════════════════\n")
	fmt.Printf("Project: %s\n", investment.Project)
	if len(investment.Sprints) > 0 {
		fmt.Printf("Sprints: %s\n", strings.Join(investment.Sprints, ", "))
	}
	fmt.Printf("Period: %s → %s (%d days)\n",
		investment.StartDate.Format("2006-01-02"),
		investment.EndDate.Format("2006-01-02"),
		investment.GetDurationInDays())
	fmt.Printf("\n💵 Cost Breakdown:\n")
	fmt.Printf("  Engineer Costs:      %.2f %s\n", investment.EngineerCosts.Amount, investment.EngineerCosts.Currency)
	fmt.Printf("  Overhead Costs:      %.2f %s\n", investment.OverheadCosts.Amount, investment.OverheadCosts.Currency)
	fmt.Printf("  Infrastructure:      %.2f %s\n", investment.InfrastructureCosts.Amount, investment.InfrastructureCosts.Currency)
	fmt.Printf("  ─────────────────────────────────\n")
	fmt.Printf("  TOTAL INVESTMENT:    %.2f %s\n", investment.TotalCost.Amount, investment.TotalCost.Currency)

	fmt.Printf("\n👥 Engineers (%d):\n", len(investment.EngineersInvolved))
	for _, eng := range investment.EngineersInvolved {
		fmt.Printf("  %s (%s): %.1fh @ %.2f/h = %.2f %s\n",
			eng.Name, eng.Level, eng.TotalHours, eng.HourlyRate, eng.TotalCost.Amount, eng.TotalCost.Currency)
	}

	if len(investment.WorkTypeBreakdown) > 0 {
		fmt.Printf("\n🏗️  Work Type Breakdown:\n")
		for workType, cost := range investment.WorkTypeBreakdown {
			fmt.Printf("  %s: %.2f %s\n", workType, cost.Amount, cost.Currency)
		}
	}

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  Tasks: %d\n", investment.GetTaskCount())
	fmt.Printf("  Engineers: %d\n", investment.GetEngineerCount())
	fmt.Printf("  Duration: %d days\n", investment.GetDurationInDays())
	fmt.Printf("  Calculated: %s\n", investment.CalculatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

// investmentSprintAction backs `assetcap investment sprint`.
func (a *App) investmentSprintAction(ctx *cli.Context) error {
	if a.investmentService == nil {
		return fmt.Errorf("investment service not available")
	}

	project := ctx.String("project")
	sprint := ctx.String("sprint")
	startDateStr := ctx.String("start-date")
	endDateStr := ctx.String("end-date")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
		}
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err)
		}
	}

	investment, err := a.investmentService.CalculateSprintInvestment(ctx.Context, project, sprint, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to calculate sprint investment: %w", err)
	}

	fmt.Printf("💰 Sprint Investment Calculation\n")
	fmt.Printf("════════════════════════════════════════════════\n")
	fmt.Printf("Project: %s\n", investment.Project)
	fmt.Printf("Sprint: %s\n", sprint)
	fmt.Printf("Total Investment: %.2f %s\n", investment.TotalCost.Amount, investment.TotalCost.Currency)
	fmt.Printf("Engineers Involved: %d\n", investment.GetEngineerCount())
	fmt.Printf("Tasks: %d\n", investment.GetTaskCount())

	return nil
}

// investmentListAction backs `assetcap investment list`.
func (a *App) investmentListAction(ctx *cli.Context) error {
	if a.investmentService == nil {
		return fmt.Errorf("investment service not available")
	}

	project := ctx.String("project")

	investments, err := a.investmentService.ListInvestments(ctx.Context, project)
	if err != nil {
		return fmt.Errorf("failed to list investments: %w", err)
	}

	if len(investments) == 0 {
		fmt.Printf("No investment calculations found")
		if project != "" {
			fmt.Printf(" for project %s", project)
		}
		fmt.Println()
		return nil
	}

	fmt.Printf("💰 Investment Calculations")
	if project != "" {
		fmt.Printf(" for %s", project)
	}
	fmt.Printf(" (%d found):\n", len(investments))
	fmt.Printf("════════════════════════════════════════════════\n")

	for _, inv := range investments {
		fmt.Printf("📦 %s (%s)\n", inv.AssetName, inv.Project)
		fmt.Printf("   Total: %.2f %s | Engineers: %d | Tasks: %d\n",
			inv.TotalCost.Amount, inv.TotalCost.Currency,
			inv.GetEngineerCount(), inv.GetTaskCount())
		if len(inv.Sprints) > 0 {
			fmt.Printf("   Sprints: %s\n", strings.Join(inv.Sprints, ", "))
		}
		fmt.Printf("   Calculated: %s\n\n", inv.CalculatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// investmentInitCostModelAction backs `assetcap investment init-cost-model`.
func (a *App) investmentInitCostModelAction(ctx *cli.Context) error {
	if a.investmentService == nil {
		return fmt.Errorf("investment service not available")
	}

	project := ctx.String("project")

	costModel, err := a.investmentService.InitializeCostModel(ctx.Context, project)
	if err != nil {
		return fmt.Errorf("failed to initialize cost model: %w", err)
	}

	fmt.Printf("✅ Cost model initialized for project %s\n", project)
	fmt.Printf("Currency: %s\n", costModel.Currency)
	fmt.Printf("Working hours per day: %.1f\n", costModel.WorkingHoursPerDay)
	fmt.Printf("Overhead multiplier: %.1fx\n", costModel.OverheadMultiplier)
	fmt.Printf("Default engineer rates:\n")
	for level, rate := range costModel.DefaultRatesByLevel {
		fmt.Printf("  %s: %.2f %s/hour\n", level, rate, costModel.Currency)
	}
	fmt.Printf("Monthly infrastructure costs: %.2f %s\n", costModel.GetTotalMonthlyCost(), costModel.Currency)

	return nil
}

// investmentSetEngineerRateAction backs `assetcap investment set-engineer-rate`.
func (a *App) investmentSetEngineerRateAction(ctx *cli.Context) error {
	if a.investmentService == nil {
		return fmt.Errorf("investment service not available")
	}

	project := ctx.String("project")
	engineerName := ctx.String("engineer")
	rateStr := ctx.String("rate")
	levelStr := ctx.String("level")

	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return fmt.Errorf("invalid rate: %w", err)
	}

	costModel, err := a.investmentService.GetCostModel(ctx.Context, project)
	if err != nil {
		return fmt.Errorf("failed to get cost model: %w", err)
	}

	level := parseEngineerLevel(levelStr)

	engineerRate := investmentdomain.EngineerRate{
		Name:       engineerName,
		HourlyRate: rate,
		Level:      level,
		Team:       project,
	}

	if err := costModel.AddEngineerRate(engineerRate); err != nil {
		return fmt.Errorf("failed to add engineer rate: %w", err)
	}

	if err := a.investmentService.UpdateCostModel(ctx.Context, project, costModel); err != nil {
		return fmt.Errorf("failed to save cost model: %w", err)
	}

	fmt.Printf("✅ Set rate for %s: %.2f %s/hour (%s level)\n",
		engineerName, rate, costModel.Currency, level)

	return nil
}

// investmentShowRatesAction backs `assetcap investment show-rates`.
func (a *App) investmentShowRatesAction(ctx *cli.Context) error {
	if a.investmentService == nil {
		return fmt.Errorf("investment service not available")
	}

	project := ctx.String("project")

	costModel, err := a.investmentService.GetCostModel(ctx.Context, project)
	if err != nil {
		return fmt.Errorf("failed to get cost model: %w", err)
	}

	fmt.Printf("💰 Engineer Rates for %s\n", project)
	fmt.Printf("════════════════════════════════════════════════\n")
	fmt.Printf("Currency: %s | Working Hours/Day: %.1f | Overhead: %.1fx\n\n",
		costModel.Currency, costModel.WorkingHoursPerDay, costModel.OverheadMultiplier)

	if len(costModel.EngineerRates) > 0 {
		fmt.Printf("👥 Individual Engineer Rates:\n")
		for name, rate := range costModel.EngineerRates {
			fmt.Printf("  %s (%s): %.2f %s/hour\n",
				name, rate.Level, rate.HourlyRate, costModel.Currency)
		}
		fmt.Println()
	}

	fmt.Printf("📊 Default Rates by Level:\n")
	for level, rate := range costModel.DefaultRatesByLevel {
		fmt.Printf("  %s: %.2f %s/hour\n", level, rate, costModel.Currency)
	}

	fmt.Printf("\n🏢 Infrastructure Costs (Monthly):\n")
	fmt.Printf("  Cloud: %.2f %s\n", costModel.InfrastructureCosts.CloudCostsPerMonth, costModel.Currency)
	fmt.Printf("  Tooling: %.2f %s\n", costModel.InfrastructureCosts.ToolingCostsPerMonth, costModel.Currency)
	fmt.Printf("  Licenses: %.2f %s\n", costModel.InfrastructureCosts.LicenseCostsPerMonth, costModel.Currency)
	fmt.Printf("  Total: %.2f %s/month\n", costModel.GetTotalMonthlyCost(), costModel.Currency)

	return nil
}

// parseEngineerLevel maps a flag string to the domain enum. Unknown
// values silently fall back to Mid, mirroring the original behaviour
// (an interactive CLI is forgiving of typos rather than failing hard).
func parseEngineerLevel(s string) investmentdomain.EngineerLevel {
	switch strings.ToLower(s) {
	case "junior":
		return investmentdomain.Junior
	case "senior":
		return investmentdomain.Senior
	case "staff":
		return investmentdomain.Staff
	case "principal":
		return investmentdomain.Principal
	default:
		return investmentdomain.Mid
	}
}
