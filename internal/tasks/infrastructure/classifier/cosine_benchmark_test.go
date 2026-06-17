package classifier

import (
	"math/rand/v2"
	"testing"
)

// BenchmarkCosineSimilarity_768 pins the hot loop that the embedding
// classifier runs for every (task, asset) pair. 768-dimensional
// vectors match what nomic-embed-text returns; this is the realistic
// shape we care about. If a future change regresses this by more
// than ~2x it almost certainly slows the classifier visibly on real
// sprint loads (hundreds of tasks * dozens of assets).
func BenchmarkCosineSimilarity_768(b *testing.B) {
	a := randomVec(768, 1)
	v := randomVec(768, 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cosineSimilarity(a, v)
	}
}

// BenchmarkCosineSimilarity_1536 covers the larger OpenAI-style
// embedding shape so a future model swap doesn't regress silently.
func BenchmarkCosineSimilarity_1536(b *testing.B) {
	a := randomVec(1536, 1)
	v := randomVec(1536, 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cosineSimilarity(a, v)
	}
}

// BenchmarkCosineSimilarity_LengthMismatch is the early-exit branch.
// Pinning it ensures the guard stays branch-predictable.
func BenchmarkCosineSimilarity_LengthMismatch(b *testing.B) {
	a := randomVec(768, 1)
	v := randomVec(769, 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cosineSimilarity(a, v)
	}
}

// randomVec builds a deterministic-per-seed pseudo-random vector.
// Using math/rand/v2 with an explicit seed avoids cross-run variance
// while still touching every element so the cache doesn't fold the
// loop down to a trivial constant.
func randomVec(n int, seed uint64) []float64 {
	r := rand.New(rand.NewPCG(seed, seed^0xdeadbeef))
	out := make([]float64, n)
	for i := range out {
		out[i] = r.Float64()
	}
	return out
}
