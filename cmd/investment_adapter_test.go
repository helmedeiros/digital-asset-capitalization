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

// stubInvestmentService satisfies InvestmentServicePort with injectable
// function fields, mirroring the small stub pattern used for the
// prompt-handler tests. Each method short-circuits to a zero value
// when no stub is wired.
type stubInvestmentService struct {
	calc func(context.Context, string, string, []string) (*investmentdomain.Investment, error)
	list func(context.Context, string) ([]*investmentdomain.Investment, error)
	rate func(context.Context, string) (*investmentdomain.CostModel, error)
}

func (s *stubInvestmentService) CalculateAssetInvestment(ctx context.Context, asset, project string, sprints []string) (*investmentdomain.Investment, error) {
	if s.calc != nil {
		return s.calc(ctx, asset, project, sprints)
	}
	return nil, nil
}

func (s *stubInvestmentService) ListInvestments(ctx context.Context, project string) ([]*investmentdomain.Investment, error) {
	if s.list != nil {
		return s.list(ctx, project)
	}
	return nil, nil
}

func (s *stubInvestmentService) GetCostModel(ctx context.Context, project string) (*investmentdomain.CostModel, error) {
	if s.rate != nil {
		return s.rate(ctx, project)
	}
	return nil, nil
}

func sampleInvestment(asset string) *investmentdomain.Investment {
	return &investmentdomain.Investment{
		AssetName: asset,
		Project:   "FN",
		Sprints:   []string{"S1"},
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		TotalCost: investmentdomain.Money{Amount: 1000, Currency: investmentdomain.USD},
		EngineersInvolved: []investmentdomain.EngineerInvestment{
			{Name: "Alice", Level: investmentdomain.Senior, TotalHours: 10, TotalCost: investmentdomain.Money{Amount: 500, Currency: investmentdomain.USD}},
			{Name: "Bob", Level: investmentdomain.Mid, TotalHours: 8, TotalCost: investmentdomain.Money{Amount: 500, Currency: investmentdomain.USD}},
		},
		WorkTypeBreakdown: map[string]investmentdomain.Money{
			"cap-development": {Amount: 750, Currency: investmentdomain.USD},
			"cap-maintenance": {Amount: 250, Currency: investmentdomain.USD},
		},
		CalculatedAt: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
	}
}

func TestInvestmentServiceAdapter_CalculateInvestment(t *testing.T) {
	ctx := context.Background()

	t.Run("empty asset rejected", func(t *testing.T) {
		_, err := (&InvestmentServiceAdapter{service: &stubInvestmentService{}}).
			CalculateInvestment(ctx, "", "FN", []string{"S1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("empty project rejected", func(t *testing.T) {
		_, err := (&InvestmentServiceAdapter{service: &stubInvestmentService{}}).
			CalculateInvestment(ctx, "X", "", []string{"S1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("empty sprint list rejected", func(t *testing.T) {
		_, err := (&InvestmentServiceAdapter{service: &stubInvestmentService{}}).
			CalculateInvestment(ctx, "X", "FN", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one sprint")
	})

	t.Run("service error wraps", func(t *testing.T) {
		svc := &stubInvestmentService{calc: func(context.Context, string, string, []string) (*investmentdomain.Investment, error) {
			return nil, errors.New("boom")
		}}
		_, err := (&InvestmentServiceAdapter{service: svc}).
			CalculateInvestment(ctx, "X", "FN", []string{"S1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to calculate investment")
	})

	t.Run("happy path returns full result with breakdowns", func(t *testing.T) {
		svc := &stubInvestmentService{calc: func(_ context.Context, asset, _ string, _ []string) (*investmentdomain.Investment, error) {
			return sampleInvestment(asset), nil
		}}
		got, err := (&InvestmentServiceAdapter{service: svc}).
			CalculateInvestment(ctx, "Payment Gateway", "FN", []string{"S1"})
		require.NoError(t, err)
		m, ok := got.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "Payment Gateway", m["asset"])
		assert.Equal(t, "FN", m["project"])
		assert.InDelta(t, 18.0, m["total_hours"], 1e-9, "10 + 8 engineer hours")

		breakdown, ok := m["engineer_breakdown"].([]map[string]interface{})
		require.True(t, ok)
		require.Len(t, breakdown, 2)
		assert.Equal(t, "Alice", breakdown[0]["name"])

		workTypes, ok := m["work_type_breakdown"].([]map[string]interface{})
		require.True(t, ok)
		require.Len(t, workTypes, 2)
	})
}

func TestInvestmentServiceAdapter_ListInvestments(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project rejected", func(t *testing.T) {
		_, err := (&InvestmentServiceAdapter{service: &stubInvestmentService{}}).
			ListInvestments(ctx, "")
		require.Error(t, err)
	})

	t.Run("service error wraps", func(t *testing.T) {
		svc := &stubInvestmentService{list: func(context.Context, string) ([]*investmentdomain.Investment, error) {
			return nil, errors.New("repo down")
		}}
		_, err := (&InvestmentServiceAdapter{service: svc}).ListInvestments(ctx, "FN")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list investments")
	})

	t.Run("empty result returns no-investments message", func(t *testing.T) {
		svc := &stubInvestmentService{list: func(context.Context, string) ([]*investmentdomain.Investment, error) {
			return nil, nil
		}}
		got, err := (&InvestmentServiceAdapter{service: svc}).ListInvestments(ctx, "FN")
		require.NoError(t, err)
		m, ok := got.(map[string]string)
		require.True(t, ok)
		assert.Contains(t, m["message"], "No investments found")
	})

	t.Run("populated result transforms and sums totals", func(t *testing.T) {
		svc := &stubInvestmentService{list: func(context.Context, string) ([]*investmentdomain.Investment, error) {
			return []*investmentdomain.Investment{sampleInvestment("A"), sampleInvestment("B")}, nil
		}}
		got, err := (&InvestmentServiceAdapter{service: svc}).ListInvestments(ctx, "FN")
		require.NoError(t, err)
		m, ok := got.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 2, m["total_investments"])
		assert.Equal(t, "$2000.00", m["total_value"], "two sample investments at $1000 each")
		invs, ok := m["investments"].([]map[string]interface{})
		require.True(t, ok)
		require.Len(t, invs, 2)
		assert.Equal(t, "A", invs[0]["asset"])
		assert.Equal(t, []string{"S1"}, invs[0]["sprints"])
	})
}

func TestInvestmentServiceAdapter_ShowRates(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project rejected", func(t *testing.T) {
		_, err := (&InvestmentServiceAdapter{service: &stubInvestmentService{}}).
			ShowRates(ctx, "")
		require.Error(t, err)
	})

	t.Run("service error wraps", func(t *testing.T) {
		svc := &stubInvestmentService{rate: func(context.Context, string) (*investmentdomain.CostModel, error) {
			return nil, errors.New("not found")
		}}
		_, err := (&InvestmentServiceAdapter{service: svc}).ShowRates(ctx, "FN")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get cost model")
	})
}
