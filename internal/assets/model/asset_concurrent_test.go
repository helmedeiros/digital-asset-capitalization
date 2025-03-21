package model

import (
	"sync"
	"testing"
	"time"
)

func TestAssetConcurrentOperations(t *testing.T) {
	asset, _ := NewAsset("Test Asset", "Test Description")
	var wg sync.WaitGroup
	concurrentOperations := 100

	// Test concurrent task count operations
	t.Run("concurrent task count operations", func(t *testing.T) {
		wg.Add(concurrentOperations)
		for i := 0; i < concurrentOperations; i++ {
			go func() {
				defer wg.Done()
				asset.IncrementTaskCount()
				asset.DecrementTaskCount()
			}()
		}
		wg.Wait()

		if asset.AssociatedTaskCount != 0 {
			t.Errorf("expected task count to be 0 after concurrent operations, got %d", asset.AssociatedTaskCount)
		}
	})

	// Test concurrent contribution type additions
	t.Run("concurrent contribution type additions", func(t *testing.T) {
		wg.Add(concurrentOperations)
		for i := 0; i < concurrentOperations; i++ {
			go func() {
				defer wg.Done()
				_ = asset.AddContributionType("development")
			}()
		}
		wg.Wait()

		if len(asset.ContributionTypes) != 1 {
			t.Errorf("expected 1 contribution type after concurrent operations, got %d", len(asset.ContributionTypes))
		}
	})

	// Test concurrent description updates
	t.Run("concurrent description updates", func(t *testing.T) {
		wg.Add(concurrentOperations)
		for i := 0; i < concurrentOperations; i++ {
			go func(i int) {
				defer wg.Done()
				_ = asset.UpdateDescription("New Description")
			}(i)
		}
		wg.Wait()

		if asset.Description != "New Description" {
			t.Errorf("expected description to be 'New Description', got %s", asset.Description)
		}
	})

	// Test concurrent documentation updates
	t.Run("concurrent documentation updates", func(t *testing.T) {
		wg.Add(concurrentOperations)
		for i := 0; i < concurrentOperations; i++ {
			go func() {
				defer wg.Done()
				asset.UpdateDocumentation()
			}()
		}
		wg.Wait()

		if asset.LastDocUpdateAt.IsZero() {
			t.Error("expected LastDocUpdateAt to be set after concurrent operations")
		}
	})
}

func TestAssetConcurrentStress(t *testing.T) {
	asset, _ := NewAsset("Test Asset", "Test Description")
	var wg sync.WaitGroup
	concurrentOperations := 1000
	duration := 5 * time.Second

	// Stress test with mixed operations
	t.Run("stress test with mixed operations", func(t *testing.T) {
		done := make(chan struct{})
		start := time.Now()

		// Start multiple goroutines performing different operations
		for i := 0; i < concurrentOperations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-done:
						return
					default:
						asset.IncrementTaskCount()
						asset.DecrementTaskCount()
						_ = asset.AddContributionType("development")
						_ = asset.UpdateDescription("Stress Test")
						asset.UpdateDocumentation()
					}
				}
			}()
		}

		// Run for the specified duration
		time.Sleep(duration)
		close(done)
		wg.Wait()

		elapsed := time.Since(start)
		t.Logf("Stress test completed in %v", elapsed)
		t.Logf("Final state - Task Count: %d, Contribution Types: %d, Version: %d",
			asset.AssociatedTaskCount,
			len(asset.ContributionTypes),
			asset.Version)
	})
}
