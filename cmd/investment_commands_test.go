package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	investmentdomain "github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// All six investment Actions begin with the same nil-check guard.
// InvestmentService is a concrete struct (not an interface) so we
// can't easily stub the happy path without a wider refactor; the
// nil-guard branch is fully testable and worth pinning since it
// protects against the bare-App misconfiguration path.

func TestApp_investmentCalculateAction_NoServiceReturnsError(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.investmentCalculateAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "investment service not available")
}

func TestApp_investmentSprintAction_NoServiceReturnsError(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.investmentSprintAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "investment service not available")
}

func TestApp_investmentListAction_NoServiceReturnsError(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.investmentListAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "investment service not available")
}

func TestApp_investmentInitCostModelAction_NoServiceReturnsError(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.investmentInitCostModelAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "investment service not available")
}

func TestApp_investmentSetEngineerRateAction_NoServiceReturnsError(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.investmentSetEngineerRateAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "investment service not available")
}

func TestApp_investmentShowRatesAction_NoServiceReturnsError(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.investmentShowRatesAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "investment service not available")
}

// parseEngineerLevel is a pure helper.

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
