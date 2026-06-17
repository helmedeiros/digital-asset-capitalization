package classifier

import (
	"fmt"
	"testing"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// BenchmarkInheritFromParent_LabelPath pins the fast path: parent
// task already carries cap-asset-* and cap-development labels, so the
// helper resolves entirely from the label scan and never calls
// assetClassifier / workTypeClassifier. This is the common case in
// batch sprint classification and should stay allocation-light.
func BenchmarkInheritFromParent_LabelPath(b *testing.B) {
	parent := &taskdomain.Task{
		Key:    "FN-1",
		Labels: []string{"cap-asset-cabin-markup", "cap-development"},
	}
	chain := &ComprehensiveClassificationChainWithInheritance{
		taskLookup: map[string]*taskdomain.Task{"FN-1": parent},
	}
	subtask := &taskdomain.Task{Key: "FN-2", Epic: "FN-1"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = chain.inheritFromParent(subtask)
	}
}

// BenchmarkInheritFromParent_FallbackPath stresses the slower branch:
// parent has no cap-* labels so the helper falls through to invoking
// the asset and work-type classifiers. With a stub set, this measures
// the orchestration cost without LLM round-trip.
func BenchmarkInheritFromParent_FallbackPath(b *testing.B) {
	parent := &taskdomain.Task{Key: "FN-1", Summary: "Parent"}
	stubAsset := &stubAssetClassifier{
		result: &ports.AssetClassificationResult{
			Task:       parent,
			Asset:      &assetdomain.Asset{Name: "Payments"},
			Confidence: 0.5,
			Reason:     "stub",
		},
	}
	stubWT := &stubWorkTypeClassifier{wt: taskdomain.WorkTypeMaintenance}
	chain := &ComprehensiveClassificationChainWithInheritance{
		assetClassifier:    stubAsset,
		workTypeClassifier: stubWT,
		taskLookup:         map[string]*taskdomain.Task{"FN-1": parent},
	}
	subtask := &taskdomain.Task{Key: "FN-2", Epic: "FN-1"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = chain.inheritFromParent(subtask)
	}
}

// BenchmarkInheritFromParent_TaskLookupCold simulates a realistic
// batch scenario: many parents in the lookup map, one subtask
// resolves into one of them. Useful to spot map-lookup regressions
// (e.g. if taskLookup were ever swapped for a slice).
func BenchmarkInheritFromParent_TaskLookupCold(b *testing.B) {
	const epicCount = 200
	lookup := make(map[string]*taskdomain.Task, epicCount)
	for i := 0; i < epicCount; i++ {
		key := fmt.Sprintf("FN-%d", i)
		lookup[key] = &taskdomain.Task{
			Key:    key,
			Labels: []string{"cap-asset-search", "cap-development"},
		}
	}
	chain := &ComprehensiveClassificationChainWithInheritance{taskLookup: lookup}
	subtask := &taskdomain.Task{Key: "FN-9999", Epic: "FN-150"} // lookup hit roughly mid-map
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = chain.inheritFromParent(subtask)
	}
}
