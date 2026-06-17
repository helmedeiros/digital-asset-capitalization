package domain_test

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// ExampleNewMoney shows the common shape: build a Money, print it
// using the String() method so the formatting contract stays pinned.
func ExampleNewMoney() {
	cost := domain.NewMoney(199.99, domain.EUR)
	fmt.Println(cost)
	// Output: 199.99 EUR
}

// ExampleMoney_Add demonstrates same-currency addition; mismatched
// currencies fall through to the receiver unchanged rather than
// silently converting.
func ExampleMoney_Add() {
	dev := domain.NewMoney(1500, domain.EUR)
	infra := domain.NewMoney(250, domain.EUR)
	total := dev.Add(infra)
	fmt.Println(total)
	// Output: 1750.00 EUR
}

// ExampleMoney_Multiply is how the cost model applies an overhead
// multiplier — the Currency travels through unchanged.
func ExampleMoney_Multiply() {
	base := domain.NewMoney(1000, domain.USD)
	withOverhead := base.Multiply(2.0)
	fmt.Println(withOverhead)
	// Output: 2000.00 USD
}

// ExampleNewCostModel shows the minimum valid construction: working
// hours must be > 0 and overhead must be >= 1.0.
func ExampleNewCostModel() {
	cm, err := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("currency=%s workingHours=%.1f overhead=%.1fx\n",
		cm.Currency, cm.WorkingHoursPerDay, cm.OverheadMultiplier)
	// Output: currency=EUR workingHours=8.0 overhead=2.0x
}

// ExampleCostModel_InferEngineerLevel falls back to fixed rate ranges
// when no defaults are configured: 30/hour maps to Junior.
func ExampleCostModel_InferEngineerLevel() {
	cm, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	level := cm.InferEngineerLevel(30.0)
	fmt.Println(level)
	// Output: junior
}
