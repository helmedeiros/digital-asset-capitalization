package model

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

		assert.Equal(t, 0, asset.AssociatedTaskCount, "expected task count to be 0 after concurrent operations")
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

		assert.Equal(t, 1, len(asset.ContributionTypes), "expected 1 contribution type after concurrent operations")
	})

	// Test concurrent description updates
	t.Run("concurrent description updates", func(t *testing.T) {
		wg.Add(concurrentOperations)
		for i := 0; i < concurrentOperations; i++ {
			go func() {
				defer wg.Done()
				_ = asset.UpdateDescription("New Description")
			}()
		}
		wg.Wait()

		assert.Equal(t, "New Description", asset.Description, "expected description to be 'New Description'")
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

		assert.False(t, asset.LastDocUpdateAt.IsZero(), "expected LastDocUpdateAt to be set after concurrent operations")
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
