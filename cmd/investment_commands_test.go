package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	investmentdomain "github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// stubFullInvestmentService records every call argument and returns the
// injected results / errors. One stub satisfies the full
// InvestmentService interface so each Action test seeds only the
// fields it needs.
type stubFullInvestmentService struct {
	calcAssetInput struct {
		assetName, project string
		sprints            []string
	}
	calcAssetResult *investmentdomain.Investment
	calcAssetErr    error

	calcSprintInput struct {
		project, sprint    string
		startDate, endDate time.Time
	}
	calcSprintResult *investmentdomain.Investment
	calcSprintErr    error

	listProject string
	listResult  []*investmentdomain.Investment
	listErr     error

	initProject string
	initResult  *investmentdomain.CostModel
	initErr     error

	getProject string
	getResult  *investmentdomain.CostModel
	getErr     error

	updateProject string
	updateModel   *investmentdomain.CostModel
	updateErr     error
}

func (s *stubFullInvestmentService) CalculateAssetInvestment(_ context.Context, assetName, project string, sprints []string) (*investmentdomain.Investment, error) {
	s.calcAssetInput.assetName = assetName
	s.calcAssetInput.project = project
	s.calcAssetInput.sprints = sprints
	return s.calcAssetResult, s.calcAssetErr
}

func (s *stubFullInvestmentService) CalculateSprintInvestment(_ context.Context, project, sprint string, startDate, endDate time.Time) (*investmentdomain.Investment, error) {
	s.calcSprintInput.project = project
	s.calcSprintInput.sprint = sprint
	s.calcSprintInput.startDate = startDate
	s.calcSprintInput.endDate = endDate
	return s.calcSprintResult, s.calcSprintErr
}

func (s *stubFullInvestmentService) ListInvestments(_ context.Context, project string) ([]*investmentdomain.Investment, error) {
	s.listProject = project
	return s.listResult, s.listErr
}

func (s *stubFullInvestmentService) InitializeCostModel(_ context.Context, project string) (*investmentdomain.CostModel, error) {
	s.initProject = project
	return s.initResult, s.initErr
}

func (s *stubFullInvestmentService) GetCostModel(_ context.Context, project string) (*investmentdomain.CostModel, error) {
	s.getProject = project
	return s.getResult, s.getErr
}

func (s *stubFullInvestmentService) UpdateCostModel(_ context.Context, project string, model *investmentdomain.CostModel) error {
	s.updateProject = project
	s.updateModel = model
	return s.updateErr
}

func newSampleInvestment() *investmentdomain.Investment {
	return &investmentdomain.Investment{
		AssetName:           "PaymentProcessing",
		Project:             "FN",
		Sprints:             []string{"Alpha"},
		StartDate:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:             time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		TotalCost:           investmentdomain.NewMoney(10000, investmentdomain.EUR),
		EngineerCosts:       investmentdomain.NewMoney(8000, investmentdomain.EUR),
		OverheadCosts:       investmentdomain.NewMoney(1500, investmentdomain.EUR),
		InfrastructureCosts: investmentdomain.NewMoney(500, investmentdomain.EUR),
		EngineersInvolved: []investmentdomain.EngineerInvestment{{
			Name: "alice", Level: investmentdomain.Senior, TotalHours: 80,
			HourlyRate: 100,
			TotalCost:  investmentdomain.NewMoney(8000, investmentdomain.EUR),
		}},
		WorkTypeBreakdown: map[string]investmentdomain.Money{
			"feature": investmentdomain.NewMoney(7000, investmentdomain.EUR),
		},
		CalculatedAt: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
	}
}

func newSampleCostModel(t *testing.T) *investmentdomain.CostModel {
	t.Helper()
	cm, err := investmentdomain.NewCostModel(investmentdomain.EUR, 8.0, 1.5)
	require.NoError(t, err)
	require.NoError(t, cm.SetDefaultRate(investmentdomain.Senior, 100))
	cm.InfrastructureCosts = investmentdomain.InfrastructureCosts{
		CloudCostsPerMonth:   1000,
		ToolingCostsPerMonth: 200,
		LicenseCostsPerMonth: 300,
	}
	return cm
}

// investment calculate

func TestApp_investmentCalculateAction_NoService(t *testing.T) {
	t.Parallel()
	a := &App{}
	err := a.investmentCalculateAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "investment service not available")
}

func TestApp_investmentCalculateAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{calcAssetErr: errors.New("boom")}}
	ctx := newContextWithFlags(t, map[string]string{"asset": "Foo", "project": "FN"}, nil)
	err := a.investmentCalculateAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate investment")
}

func TestApp_investmentCalculateAction_Success(t *testing.T) {
	stub := &stubFullInvestmentService{calcAssetResult: newSampleInvestment()}
	a := &App{investmentService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"asset": "Foo", "project": "FN", "sprints": "Alpha, Beta",
	}, nil)
	out, err := captureStdout(t, func() error { return a.investmentCalculateAction(ctx) })
	require.NoError(t, err)
	assert.Equal(t, "Foo", stub.calcAssetInput.assetName)
	assert.Equal(t, "FN", stub.calcAssetInput.project)
	assert.Equal(t, []string{"Alpha", "Beta"}, stub.calcAssetInput.sprints)
	assert.Contains(t, out, "Investment Calculation for 'PaymentProcessing'")
	assert.Contains(t, out, "Project: FN")
	assert.Contains(t, out, "Sprints: Alpha")
	assert.Contains(t, out, "TOTAL INVESTMENT:")
	assert.Contains(t, out, "Engineers (1):")
	assert.Contains(t, out, "alice (senior):")
	assert.Contains(t, out, "Work Type Breakdown:")
	assert.Contains(t, out, "feature:")
}

// investment sprint

func TestApp_investmentSprintAction_NoService(t *testing.T) {
	t.Parallel()
	a := &App{}
	err := a.investmentSprintAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_investmentSprintAction_BadStartDate(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "sprint": "Alpha", "start-date": "yesterday",
	}, nil)
	err := a.investmentSprintAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start date format")
}

func TestApp_investmentSprintAction_BadEndDate(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "sprint": "Alpha",
		"start-date": "2026-01-01", "end-date": "later",
	}, nil)
	err := a.investmentSprintAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid end date format")
}

func TestApp_investmentSprintAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{calcSprintErr: errors.New("boom")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN", "sprint": "Alpha"}, nil)
	err := a.investmentSprintAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate sprint investment")
}

func TestApp_investmentSprintAction_Success(t *testing.T) {
	stub := &stubFullInvestmentService{calcSprintResult: newSampleInvestment()}
	a := &App{investmentService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "sprint": "Alpha",
		"start-date": "2026-01-01", "end-date": "2026-01-14",
	}, nil)
	out, err := captureStdout(t, func() error { return a.investmentSprintAction(ctx) })
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), stub.calcSprintInput.startDate)
	assert.Equal(t, time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC), stub.calcSprintInput.endDate)
	assert.Contains(t, out, "Sprint Investment Calculation")
	assert.Contains(t, out, "Sprint: Alpha")
	assert.Contains(t, out, "Total Investment:")
}

// investment list

func TestApp_investmentListAction_NoService(t *testing.T) {
	t.Parallel()
	a := &App{}
	err := a.investmentListAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_investmentListAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{listErr: errors.New("boom")}}
	err := a.investmentListAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list investments")
}

func TestApp_investmentListAction_EmptyAllProjects(t *testing.T) {
	a := &App{investmentService: &stubFullInvestmentService{}}
	out, err := captureStdout(t, func() error {
		return a.investmentListAction(newContextWithFlags(t, nil, nil))
	})
	require.NoError(t, err)
	assert.Contains(t, out, "No investment calculations found")
	assert.NotContains(t, out, "for project")
}

func TestApp_investmentListAction_EmptyForProject(t *testing.T) {
	a := &App{investmentService: &stubFullInvestmentService{}}
	out, err := captureStdout(t, func() error {
		return a.investmentListAction(newContextWithFlags(t, map[string]string{"project": "FN"}, nil))
	})
	require.NoError(t, err)
	assert.Contains(t, out, "No investment calculations found for project FN")
}

func TestApp_investmentListAction_Success(t *testing.T) {
	stub := &stubFullInvestmentService{
		listResult: []*investmentdomain.Investment{newSampleInvestment()},
	}
	a := &App{investmentService: stub}
	out, err := captureStdout(t, func() error {
		return a.investmentListAction(newContextWithFlags(t, map[string]string{"project": "FN"}, nil))
	})
	require.NoError(t, err)
	assert.Equal(t, "FN", stub.listProject)
	assert.Contains(t, out, "Investment Calculations for FN (1 found):")
	assert.Contains(t, out, "PaymentProcessing (FN)")
	assert.Contains(t, out, "Sprints: Alpha")
}

// investment init-cost-model

func TestApp_investmentInitCostModelAction_NoService(t *testing.T) {
	t.Parallel()
	a := &App{}
	err := a.investmentInitCostModelAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_investmentInitCostModelAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{initErr: errors.New("boom")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.investmentInitCostModelAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize cost model")
}

func TestApp_investmentInitCostModelAction_Success(t *testing.T) {
	stub := &stubFullInvestmentService{initResult: newSampleCostModel(t)}
	a := &App{investmentService: stub}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.investmentInitCostModelAction(ctx) })
	require.NoError(t, err)
	assert.Equal(t, "FN", stub.initProject)
	assert.Contains(t, out, "Cost model initialized for project FN")
	assert.Contains(t, out, "Currency: EUR")
	assert.Contains(t, out, "Working hours per day: 8.0")
	assert.Contains(t, out, "Default engineer rates:")
}

// investment set-engineer-rate

func TestApp_investmentSetEngineerRateAction_NoService(t *testing.T) {
	t.Parallel()
	a := &App{}
	err := a.investmentSetEngineerRateAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_investmentSetEngineerRateAction_InvalidRate(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "engineer": "alice", "rate": "not-a-number",
	}, nil)
	err := a.investmentSetEngineerRateAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid rate")
}

func TestApp_investmentSetEngineerRateAction_GetModelErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{getErr: errors.New("boom")}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "engineer": "alice", "rate": "75",
	}, nil)
	err := a.investmentSetEngineerRateAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cost model")
}

func TestApp_investmentSetEngineerRateAction_UpdateErrorWraps(t *testing.T) {
	t.Parallel()
	stub := &stubFullInvestmentService{
		getResult: newSampleCostModel(t),
		updateErr: errors.New("disk"),
	}
	a := &App{investmentService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "engineer": "alice", "rate": "75",
	}, nil)
	err := a.investmentSetEngineerRateAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save cost model")
}

func TestApp_investmentSetEngineerRateAction_Success(t *testing.T) {
	stub := &stubFullInvestmentService{getResult: newSampleCostModel(t)}
	a := &App{investmentService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "engineer": "alice", "rate": "75.5", "level": "senior",
	}, nil)
	out, err := captureStdout(t, func() error { return a.investmentSetEngineerRateAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Set rate for alice: 75.50 EUR/hour (senior level)")
	require.NotNil(t, stub.updateModel)
	rate, err := stub.updateModel.GetEngineerRate("alice")
	require.NoError(t, err)
	assert.InDelta(t, 75.5, rate, 0.001)
}

// investment show-rates

func TestApp_investmentShowRatesAction_NoService(t *testing.T) {
	t.Parallel()
	a := &App{}
	err := a.investmentShowRatesAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_investmentShowRatesAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{investmentService: &stubFullInvestmentService{getErr: errors.New("boom")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.investmentShowRatesAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cost model")
}

func TestApp_investmentShowRatesAction_SuccessWithRates(t *testing.T) {
	cm := newSampleCostModel(t)
	require.NoError(t, cm.AddEngineerRate(investmentdomain.EngineerRate{
		Name: "alice", HourlyRate: 100, Level: investmentdomain.Senior,
	}))
	a := &App{investmentService: &stubFullInvestmentService{getResult: cm}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.investmentShowRatesAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Engineer Rates for FN")
	assert.Contains(t, out, "Currency: EUR")
	assert.Contains(t, out, "Individual Engineer Rates:")
	assert.Contains(t, out, "alice (senior)")
	assert.Contains(t, out, "Default Rates by Level:")
	assert.Contains(t, out, "Infrastructure Costs (Monthly):")
	assert.Contains(t, out, "Cloud:")
}

func TestApp_investmentShowRatesAction_SuccessNoIndividualRates(t *testing.T) {
	a := &App{investmentService: &stubFullInvestmentService{getResult: newSampleCostModel(t)}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.investmentShowRatesAction(ctx) })
	require.NoError(t, err)
	assert.NotContains(t, out, "Individual Engineer Rates:")
	assert.Contains(t, out, "Default Rates by Level:")
}

func TestParseEngineerLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want investmentdomain.EngineerLevel
	}{
		{"junior", investmentdomain.Junior},
		{"JUNIOR", investmentdomain.Junior},
		{"mid", investmentdomain.Mid},
		{"senior", investmentdomain.Senior},
		{"staff", investmentdomain.Staff},
		{"principal", investmentdomain.Principal},
		{"", investmentdomain.Mid},
		{"unknown", investmentdomain.Mid},
	}
	for _, c := range cases {
		c := c
		t.Run(c.in+"->"+string(c.want), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, parseEngineerLevel(c.in))
		})
	}
}
