package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	investmentdomain "github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// TestInvestmentServiceAdapter_ShowRates_HappyPath exercises ShowRates'
// post-validation branches (engineer rates loop, default rates loop,
// and the optional infrastructure costs branch). The validation +
// service-error branches are already covered in investment_adapter_test.go.
func TestInvestmentServiceAdapter_ShowRates_HappyPath(t *testing.T) {
	ctx := context.Background()

	costModel, err := investmentdomain.NewCostModel(investmentdomain.USD, 8.0, 1.5)
	require.NoError(t, err)
	costModel.EngineerRates["Alice"] = investmentdomain.EngineerRate{
		Name: "Alice", HourlyRate: 150.0, Level: investmentdomain.Senior, Team: "payments",
	}
	costModel.DefaultRatesByLevel[investmentdomain.Senior] = 130.0
	costModel.DefaultRatesByLevel[investmentdomain.Mid] = 100.0

	t.Run("returns the cost model shape without infrastructure costs by default", func(t *testing.T) {
		svc := &stubInvestmentService{rate: func(context.Context, string) (*investmentdomain.CostModel, error) {
			return costModel, nil
		}}
		got, err := (&InvestmentServiceAdapter{service: svc}).ShowRates(ctx, "FN")
		require.NoError(t, err)
		m, ok := got.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "FN", m["project"])
		assert.Equal(t, investmentdomain.USD, m["currency"])
		assert.InDelta(t, 8.0, m["working_hours_per_day"], 1e-9)
		assert.InDelta(t, 1.5, m["overhead_multiplier"], 1e-9)

		engineerRates, ok := m["engineer_rates"].(map[string]string)
		require.True(t, ok)
		assert.Contains(t, engineerRates, "Alice (senior)")
		assert.Contains(t, engineerRates["Alice (senior)"], "150.00 USD/hour")

		defaultRates, ok := m["default_rates_by_level"].(map[string]string)
		require.True(t, ok)
		assert.Contains(t, defaultRates["senior"], "130.00 USD/hour")
		assert.Contains(t, defaultRates["mid"], "100.00 USD/hour")

		_, hasInfra := m["infrastructure_costs"]
		assert.False(t, hasInfra, "infrastructure_costs should be omitted when all monthly costs are zero")
	})

	t.Run("returns infrastructure_costs map when any monthly cost is set", func(t *testing.T) {
		costModelWithInfra, err := investmentdomain.NewCostModel(investmentdomain.USD, 8.0, 1.5)
		require.NoError(t, err)
		costModelWithInfra.InfrastructureCosts.CloudCostsPerMonth = 500
		costModelWithInfra.InfrastructureCosts.ToolingCostsPerMonth = 100
		costModelWithInfra.InfrastructureCosts.LicenseCostsPerMonth = 50

		svc := &stubInvestmentService{rate: func(context.Context, string) (*investmentdomain.CostModel, error) {
			return costModelWithInfra, nil
		}}
		got, err := (&InvestmentServiceAdapter{service: svc}).ShowRates(ctx, "FN")
		require.NoError(t, err)
		m, ok := got.(map[string]interface{})
		require.True(t, ok)
		infra, ok := m["infrastructure_costs"].(map[string]string)
		require.True(t, ok)
		assert.Contains(t, infra["cloud_per_month"], "500.00")
		assert.Contains(t, infra["tooling_per_month"], "100.00")
		assert.Contains(t, infra["license_per_month"], "50.00")
		assert.Contains(t, infra["total_per_month"], "650.00", "GetTotalMonthlyCost should sum the three")
	})
}
