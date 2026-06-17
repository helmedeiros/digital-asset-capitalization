package confluence

import (
	"strings"
	"testing"
)

// BenchmarkFormatAsContent_Paragraph exercises the single-line wrap
// path. Realistic input shape: an asset's 'Why' field with ~120 chars
// of prose.
func BenchmarkFormatAsContent_Paragraph(b *testing.B) {
	in := "Lower carrier cabin markup unlocks revenue on existing routes by aligning price-elasticity with realised demand patterns."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatAsContent(in)
	}
}

// BenchmarkFormatAsContent_BulletList exercises the multi-line bullet
// rendering with leading bullet-marker stripping. Realistic input
// shape: an asset's 'Benefits' field with 6 short bullets.
func BenchmarkFormatAsContent_BulletList(b *testing.B) {
	in := "- Lower customer-acquisition cost\n- Higher attach rate on ancillaries\n- Faster route launch (auto-tuned markup)\n- Reduced manual analyst toil\n- Improved A/B experimentation cadence\n- Cleaner data lineage for finance"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatAsContent(in)
	}
}

// BenchmarkFormatAsContent_Large stresses the linear-scan portions
// (HTML escape + line split) at a size larger than any real asset
// field today. If a future change accidentally drops in quadratic
// behaviour (e.g. via repeated strings.ReplaceAll), this will spike.
func BenchmarkFormatAsContent_Large(b *testing.B) {
	line := "Item with <special> & 'characters' that need HTML escaping inside the loop"
	in := strings.Repeat(line+"\n", 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatAsContent(in)
	}
}
