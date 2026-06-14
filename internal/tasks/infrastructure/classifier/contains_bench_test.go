package classifier

import (
	"testing"

	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// Benchmark inputs sized roughly like a real task summary plus
// description so the numbers reflect real classification load, not a
// 5-character toy string. Realistic content typically holds ~200-500
// bytes; we pick 256 plus a known suffix so the keyword hits late.
var (
	benchContent = func() string {
		const filler = "Reviewed the data pipeline integration with our new service layer; " +
			"verified the latency budget; the SRE rotation owner signed off on Friday. " +
			"There were several edge cases around retry semantics and idempotency keys."
		return filler + " Implement new API endpoint"
	}()
	benchKeywords = []string{"api", "endpoint", "service", "new", "add", "create", "implement"}
)

func BenchmarkContainsAny(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if !containsAny(benchContent, benchKeywords) {
			b.Fatal("expected match")
		}
	}
}

// BenchmarkContainsAPIKeywords measures the realistic per-task hot
// path: a task whose summary+description hits the API-keyword case,
// classified once. Useful for tracking regression when the keyword set
// or the surrounding function shape changes.
func BenchmarkContainsAPIKeywords(b *testing.B) {
	chain := &ComprehensiveClassificationChain{}
	task := &taskdomain.Task{
		Summary:     "Implement new API endpoint for the markup service",
		Description: benchContent,
	}
	for i := 0; i < b.N; i++ {
		if !chain.containsAPIKeywords(task) {
			b.Fatal("expected match")
		}
	}
}
