package model

import (
	"testing"
)

func BenchmarkAssetCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewAsset("Test Asset", "Test Description")
	}
}

func BenchmarkAssetUpdateDescription(b *testing.B) {
	asset, _ := NewAsset("Test Asset", "Test Description")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = asset.UpdateDescription("New Description")
	}
}

func BenchmarkAssetAddContributionType(b *testing.B) {
	asset, _ := NewAsset("Test Asset", "Test Description")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = asset.AddContributionType("development")
	}
}

func BenchmarkAssetTaskCountOperations(b *testing.B) {
	asset, _ := NewAsset("Test Asset", "Test Description")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asset.IncrementTaskCount()
		asset.DecrementTaskCount()
	}
}

func BenchmarkAssetIDGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateID("Test Asset")
	}
}

func BenchmarkAssetConcurrentOperations(b *testing.B) {
	asset, _ := NewAsset("Test Asset", "Test Description")
	b.ResetTimer()

	// Create a channel to coordinate goroutines
	done := make(chan bool)

	// Run b.N operations across multiple goroutines
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			select {
			case <-done:
				return
			default:
				asset.IncrementTaskCount()
				asset.DecrementTaskCount()
				_ = asset.AddContributionType("development")
			}
		}
	})

	close(done)
}
