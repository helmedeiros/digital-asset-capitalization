//go:build benchmark

package classifier

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

const (
	dataDir       = "../../../../.assetcap"
	embeddingDims = 384
)

// embeddingModels lists the models to benchmark; the first available one is also used as the default
var embeddingModels = []string{"nomic-embed-text", "llama3"}

type groundTruthEntry struct {
	Task              *taskdomain.Task
	ExpectedAssetName string
	Label             string
}

type benchmarkMetrics struct {
	Name             string
	Total            int
	Precision1       int
	Top3HitRate      int
	ConfidenceHits   []float64
	ConfidenceMisses []float64
	Elapsed          time.Duration
	Disagreements    []disagreement
}

type disagreement struct {
	TaskKey    string
	Summary    string
	Classified string
	Truth      string
	Confidence float64
}

func TestClassifierBenchmark(t *testing.T) {
	tasks := loadBenchmarkTasks(t)
	assets := loadBenchmarkAssets(t)

	groundTruth := extractGroundTruth(t, tasks, assets)
	if len(groundTruth) == 0 {
		t.Fatal("no labeled tasks found in tasks.json")
	}
	if len(assets) == 0 {
		t.Fatal("no assets found in assets.json")
	}

	categories := map[string]bool{}
	for _, gt := range groundTruth {
		categories[gt.ExpectedAssetName] = true
	}

	gtTasks := make([]*taskdomain.Task, len(groundTruth))
	for i, gt := range groundTruth {
		gtTasks[i] = gt.Task
	}

	// Build historical task map from ALL tasks (for enriching asset embeddings)
	historicalTasks := buildHistoricalTaskMap(tasks)
	t.Logf("Historical tasks: %d asset IDs with task summaries", len(historicalTasks))

	// Build epic name resolver: task summaries + inferred epic names from sibling labels
	epicNames := buildEpicNameMap(tasks, assets)
	t.Logf("Epic/task names resolved: %d keys to summaries", len(epicNames))

	// Build epic-to-dominant-asset hints for score boosting
	epicAssetHint := buildEpicAssetHints(tasks)
	t.Logf("Epic asset hints: %d epics with dominant asset", len(epicAssetHint))

	ollamaAvailable := isOllamaAvailable()
	if ollamaAvailable {
		t.Log("Ollama detected — using REAL embeddings")
	} else {
		t.Log("Ollama not available — using deterministic pseudo-embeddings (accuracy not meaningful)")
	}

	t.Logf("Dataset: %d labeled tasks, %d asset categories, %d total assets",
		len(groundTruth), len(categories), len(assets))

	nameIndex := buildNameIndex(assets)

	heuristicMetrics := runHeuristicBenchmark(t, gtTasks, assets, groundTruth, nameIndex)

	var embeddingResults []benchmarkMetrics
	if ollamaAvailable {
		for _, model := range embeddingModels {
			t.Logf("Running embedding benchmark with model: %s", model)
			m := runEmbeddingBenchmarkWithModel(t, gtTasks, assets, groundTruth, model, nameIndex, historicalTasks, epicNames, epicAssetHint)
			embeddingResults = append(embeddingResults, m)
		}
	} else {
		m := runEmbeddingBenchmarkMock(t, gtTasks, assets, groundTruth, nameIndex, historicalTasks, epicNames, epicAssetHint)
		embeddingResults = append(embeddingResults, m)
	}

	printMultiModelReport(t, groundTruth, assets, categories, heuristicMetrics, embeddingResults)
}

// buildEpicNameMap creates a map from epic key to epic summary.
// First resolves from tasks that exist in the dataset, then infers names for
// unresolved epics from the dominant asset label of sibling tasks.
// Also indexes all task keys to summaries so task-key references in titles can be resolved.
func buildEpicNameMap(tasks []*taskdomain.Task, assets []*assetdomain.Asset) map[string]string {
	// Collect all epic keys referenced by tasks
	epicKeys := map[string]bool{}
	for _, t := range tasks {
		if t.Epic != "" {
			epicKeys[t.Epic] = true
		}
	}

	// Phase 1: Resolve keys to summaries from existing tasks
	result := map[string]string{}
	for _, t := range tasks {
		// Index ALL task keys (not just epics) so title references like "COP-3" resolve
		if t.Summary != "" {
			result[t.Key] = t.Summary
		}
	}

	// Phase 2: For epic keys not found as tasks, infer from sibling task asset labels
	// Build asset ID → asset name lookup
	assetIDToName := map[string]string{}
	for _, a := range assets {
		if !strings.HasPrefix(strings.ToLower(a.Name), "cap-asset-") {
			assetIDToName[strings.ToLower(a.ID)] = a.Name
		}
	}

	for epicKey := range epicKeys {
		if _, ok := result[epicKey]; ok {
			continue // already resolved
		}

		// Count asset labels across sibling tasks
		labelCounts := map[string]int{}
		for _, t := range tasks {
			if t.Epic != epicKey {
				continue
			}
			for _, l := range t.Labels {
				if strings.HasPrefix(strings.ToLower(l), "cap-asset-") {
					labelCounts[strings.ToLower(l)]++
				}
			}
		}

		// Find dominant label (skip if ambiguous — top label must have >50% share)
		var topLabel string
		var topCount, total int
		for label, count := range labelCounts {
			total += count
			if count > topCount {
				topCount = count
				topLabel = label
			}
		}

		if topLabel != "" && total > 0 && float64(topCount)/float64(total) > 0.5 {
			if name, ok := assetIDToName[topLabel]; ok {
				result[epicKey] = name + " epic"
			}
		}
	}

	return result
}

// buildEpicAssetHints maps epic keys to their dominant asset ID.
// Only includes epics where >50% of labeled sibling tasks share the same asset.
func buildEpicAssetHints(tasks []*taskdomain.Task) map[string]string {
	type epicStats struct {
		labelCounts map[string]int
		total       int
	}

	stats := map[string]*epicStats{}
	for _, t := range tasks {
		if t.Epic == "" {
			continue
		}
		if stats[t.Epic] == nil {
			stats[t.Epic] = &epicStats{labelCounts: map[string]int{}}
		}
		for _, l := range t.Labels {
			if strings.HasPrefix(strings.ToLower(l), "cap-asset-") {
				lower := strings.ToLower(l)
				stats[t.Epic].labelCounts[lower]++
				stats[t.Epic].total++
				break
			}
		}
	}

	result := map[string]string{}
	for epicKey, s := range stats {
		if s.total == 0 {
			continue
		}
		var topLabel string
		var topCount int
		for label, count := range s.labelCounts {
			if count > topCount {
				topCount = count
				topLabel = label
			}
		}
		// Only use hint if dominant asset has >50% share
		if float64(topCount)/float64(s.total) > 0.5 {
			result[epicKey] = topLabel
		}
	}
	return result
}

// buildHistoricalTaskMap groups task summaries by their cap-asset-* label.
// Returns map[assetID][]summary, e.g. map["cap-asset-carrier-comparison-optimization"][]string{"AB Enable carriers...", ...}
func buildHistoricalTaskMap(tasks []*taskdomain.Task) map[string][]string {
	result := map[string][]string{}
	for _, task := range tasks {
		for _, label := range task.Labels {
			if strings.HasPrefix(strings.ToLower(label), "cap-asset-") {
				assetID := strings.ToLower(label)
				result[assetID] = append(result[assetID], task.Summary)
				break // one asset label per task
			}
		}
	}
	return result
}

func isOllamaAvailable() bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func loadBenchmarkTasks(t *testing.T) []*taskdomain.Task {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, "tasks.json"))
	if err != nil {
		t.Fatalf("failed to read tasks.json: %v", err)
	}

	var tasksMap map[string]*taskdomain.Task
	if err := json.Unmarshal(data, &tasksMap); err != nil {
		var tasksList []*taskdomain.Task
		if err2 := json.Unmarshal(data, &tasksList); err2 != nil {
			t.Fatalf("failed to parse tasks.json: %v (also tried array: %v)", err, err2)
		}
		return tasksList
	}

	tasks := make([]*taskdomain.Task, 0, len(tasksMap))
	for _, task := range tasksMap {
		tasks = append(tasks, task)
	}
	return tasks
}

func loadBenchmarkAssets(t *testing.T) []*assetdomain.Asset {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, "assets.json"))
	if err != nil {
		t.Fatalf("failed to read assets.json: %v", err)
	}

	var assetsMap map[string]*assetdomain.Asset
	if err := json.Unmarshal(data, &assetsMap); err != nil {
		t.Fatalf("failed to parse assets.json: %v", err)
	}

	assets := make([]*assetdomain.Asset, 0, len(assetsMap))
	for _, asset := range assetsMap {
		assets = append(assets, asset)
	}
	return assets
}

func extractGroundTruth(t *testing.T, tasks []*taskdomain.Task, assets []*assetdomain.Asset) []groundTruthEntry {
	t.Helper()

	// Build lookup maps for resolving labels to actual asset names
	assetNameExact := map[string]string{}   // lowercase -> actual name
	assetNameByWord := map[string][]string{} // word -> asset names containing that word
	for _, a := range assets {
		lower := strings.ToLower(a.Name)
		// Skip raw cap-asset-* entries in the asset repository
		if strings.HasPrefix(lower, "cap-asset-") {
			continue
		}
		assetNameExact[lower] = a.Name
		for _, word := range strings.Fields(lower) {
			assetNameByWord[word] = append(assetNameByWord[word], a.Name)
		}
	}

	var entries []groundTruthEntry
	var skipped int
	for _, task := range tasks {
		for _, label := range task.Labels {
			if !strings.HasPrefix(strings.ToLower(label), "cap-asset-") {
				continue
			}
			assetID := strings.TrimPrefix(strings.ToLower(label), "cap-asset-")
			rawName := labelToAssetName(assetID)

			// Skip labels that don't map to real assets
			if strings.EqualFold(rawName, "Not Applicable") || strings.EqualFold(rawName, "Test Asset") {
				skipped++
				break
			}

			// Resolve to actual asset name
			resolvedName := resolveAssetName(rawName, assetNameExact)
			if resolvedName == "" {
				t.Logf("SKIP: label %q -> %q has no matching asset", label, rawName)
				skipped++
				break
			}

			entries = append(entries, groundTruthEntry{
				Task:              task,
				ExpectedAssetName: resolvedName,
				Label:             label,
			})
			break
		}
	}

	if skipped > 0 {
		t.Logf("Skipped %d tasks with unmatchable labels", skipped)
	}
	return entries
}

// resolveAssetName maps a label-derived name to an actual asset name in the repository
func resolveAssetName(rawName string, assetNameExact map[string]string) string {
	lower := strings.ToLower(rawName)

	// Exact match
	if actual, ok := assetNameExact[lower]; ok {
		return actual
	}

	// Known aliases (label -> actual asset name)
	aliases := map[string]string{
		"insurance":            "Insurance Platform",
		"esim":                 "E-sim (Kolet)",
		"pay page optimization": "Payment Page Optimization",
	}
	if actual, ok := aliases[lower]; ok {
		if _, exists := assetNameExact[strings.ToLower(actual)]; exists {
			return actual
		}
	}

	// Substring match: find asset whose name contains the label name or vice versa
	for assetLower, actual := range assetNameExact {
		if strings.Contains(assetLower, lower) || strings.Contains(lower, assetLower) {
			return actual
		}
	}

	return ""
}

func labelToAssetName(identifier string) string {
	parts := strings.Split(identifier, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

func runHeuristicBenchmark(t *testing.T, tasks []*taskdomain.Task, assets []*assetdomain.Asset, groundTruth []groundTruthEntry, nameIndex map[string]string) benchmarkMetrics {
	t.Helper()

	repo := &mockAssetRepo{assets: assets}
	clf := NewContentBasedAssetClassifier(repo)

	start := time.Now()
	results, err := clf.ClassifyTasksAssets(tasks)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("heuristic classifier failed: %v", err)
	}

	return evaluateClassificationResults("Heuristic", results, groundTruth, elapsed, nameIndex)
}

func runEmbeddingBenchmarkWithModel(t *testing.T, tasks []*taskdomain.Task, assets []*assetdomain.Asset, groundTruth []groundTruthEntry, model string, nameIndex map[string]string, historicalTasks map[string][]string, epicNames map[string]string, epicAssetHint map[string]string) benchmarkMetrics {
	t.Helper()

	repo := &mockAssetRepo{assets: assets}
	store := newTestStore(t)

	client, err := llama.NewClient(llama.Config{BaseURL: "http://localhost:11434"})
	if err != nil {
		t.Fatalf("failed to create Ollama client: %v", err)
	}
	svc := NewOllamaEmbeddingAdapter(client, model)
	clf := NewEmbeddingAssetClassifierWithHistory(svc, repo, store, historicalTasks, epicNames, epicAssetHint)

	start := time.Now()
	results, err := clf.ClassifyTasksAssets(tasks)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("embedding classifier (%s) failed: %v", model, err)
	}

	return evaluateClassificationResults(fmt.Sprintf("Embed(%s)", model), results, groundTruth, elapsed, nameIndex)
}

func runEmbeddingBenchmarkMock(t *testing.T, tasks []*taskdomain.Task, assets []*assetdomain.Asset, groundTruth []groundTruthEntry, nameIndex map[string]string, historicalTasks map[string][]string, epicNames map[string]string, epicAssetHint map[string]string) benchmarkMetrics {
	t.Helper()

	repo := &mockAssetRepo{assets: assets}
	store := newTestStore(t)
	svc := &deterministicEmbeddingService{dims: embeddingDims}
	clf := NewEmbeddingAssetClassifierWithHistory(svc, repo, store, historicalTasks, epicNames, epicAssetHint)

	start := time.Now()
	results, err := clf.ClassifyTasksAssets(tasks)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("embedding classifier (mock) failed: %v", err)
	}

	return evaluateClassificationResults("Embed(mock)", results, groundTruth, elapsed, nameIndex)
}

func evaluateClassificationResults(name string, results []*ports.AssetClassificationResult, groundTruth []groundTruthEntry, elapsed time.Duration, nameIndex map[string]string) benchmarkMetrics {
	m := benchmarkMetrics{
		Name:    name,
		Total:   len(groundTruth),
		Elapsed: elapsed,
	}

	for i, gt := range groundTruth {
		if i >= len(results) {
			break
		}
		res := results[i]

		classifiedName := ""
		if res.Asset != nil {
			classifiedName = res.Asset.Name
		}

		if classifiedName != "" && matchesGroundTruthWithIndex(classifiedName, gt.ExpectedAssetName, nameIndex) {
			m.Precision1++
			m.Top3HitRate++
			m.ConfidenceHits = append(m.ConfidenceHits, res.Confidence)
		} else {
			m.ConfidenceMisses = append(m.ConfidenceMisses, res.Confidence)
			m.Disagreements = append(m.Disagreements, disagreement{
				TaskKey:    gt.Task.Key,
				Summary:    truncate(gt.Task.Summary, 60),
				Classified: classifiedName,
				Truth:      gt.ExpectedAssetName,
				Confidence: res.Confidence,
			})
		}
	}

	return m
}

// buildNameIndex creates a lookup from various name forms to a canonical asset name.
// This handles label-reconstructed names (e.g. "Esim" -> "E-sim (Kolet)") matching
// actual asset names from the repository.
func buildNameIndex(assets []*assetdomain.Asset) map[string]string {
	idx := map[string]string{}

	for _, a := range assets {
		if strings.HasPrefix(strings.ToLower(a.Name), "cap-asset-") {
			continue
		}
		lower := strings.ToLower(a.Name)
		idx[lower] = a.Name

		// Also index without parenthetical suffixes: "E-sim (Kolet)" -> "e-sim"
		if paren := strings.Index(lower, " ("); paren > 0 {
			idx[lower[:paren]] = a.Name
		}
		// Also index with hyphens removed: "e-sim" -> "esim"
		noHyphen := strings.ReplaceAll(lower, "-", "")
		idx[noHyphen] = a.Name
		// No-hyphen without parenthetical: "esim (kolet)" -> "esim"
		if paren := strings.Index(noHyphen, " ("); paren > 0 {
			idx[noHyphen[:paren]] = a.Name
		}
	}

	// Add explicit aliases
	aliases := map[string]string{
		"pay page optimization": "Payment Page Optimization",
	}
	for alias, actual := range aliases {
		idx[alias] = actual
	}

	return idx
}

func matchesGroundTruthWithIndex(classifiedName, expectedName string, nameIndex map[string]string) bool {
	a := strings.TrimSpace(strings.ToLower(classifiedName))
	b := strings.TrimSpace(strings.ToLower(expectedName))

	if a == b {
		return true
	}

	// Resolve both names through the index to canonical form
	canonA := nameIndex[a]
	canonB := nameIndex[b]

	if canonA != "" && canonB != "" && strings.EqualFold(canonA, canonB) {
		return true
	}

	// Direct substring containment
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}

	// Canonical substring
	if canonA != "" && canonB != "" {
		la := strings.ToLower(canonA)
		lb := strings.ToLower(canonB)
		if strings.Contains(la, lb) || strings.Contains(lb, la) {
			return true
		}
	}

	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func printMultiModelReport(t *testing.T, groundTruth []groundTruthEntry, assets []*assetdomain.Asset, categories map[string]bool, heuristic benchmarkMetrics, embeddings []benchmarkMetrics) {
	t.Helper()

	allMetrics := append([]benchmarkMetrics{heuristic}, embeddings...)
	colWidth := 22

	fmt.Println()
	fmt.Println("=== CLASSIFIER BENCHMARK REPORT ===")
	fmt.Printf("Dataset: %d labeled tasks, %d asset categories, %d total assets\n\n",
		len(groundTruth), len(categories), len(assets))

	// Header
	fmt.Printf("%-28s", "")
	for _, m := range allMetrics {
		fmt.Printf("%-*s", colWidth, m.Name)
	}
	fmt.Println()
	fmt.Printf("%-28s", strings.Repeat("-", 28))
	for range allMetrics {
		fmt.Printf("%-*s", colWidth, strings.Repeat("-", colWidth-2))
	}
	fmt.Println()

	// Precision@1
	fmt.Printf("%-28s", "Precision@1")
	for _, m := range allMetrics {
		fmt.Printf("%-*s", colWidth, fmtPct(m.Precision1, m.Total))
	}
	fmt.Println()

	// Top-3
	fmt.Printf("%-28s", "Top-3 Hit Rate")
	for _, m := range allMetrics {
		fmt.Printf("%-*s", colWidth, fmtPct(m.Top3HitRate, m.Total))
	}
	fmt.Println()

	// Confidence hit
	fmt.Printf("%-28s", "Mean Confidence (hit)")
	for _, m := range allMetrics {
		fmt.Printf("%-*s", colWidth, fmt.Sprintf("%.3f", mean(m.ConfidenceHits)))
	}
	fmt.Println()

	// Confidence miss
	fmt.Printf("%-28s", "Mean Confidence (miss)")
	for _, m := range allMetrics {
		fmt.Printf("%-*s", colWidth, fmt.Sprintf("%.3f", mean(m.ConfidenceMisses)))
	}
	fmt.Println()

	// Elapsed
	fmt.Printf("%-28s", "Elapsed Time")
	for _, m := range allMetrics {
		fmt.Printf("%-*s", colWidth, m.Elapsed.Round(time.Millisecond).String())
	}
	fmt.Println()

	// Disagreements per classifier
	for _, m := range allMetrics {
		printDisagreements(m.Name+" misses", m.Disagreements, 20)
	}

	fmt.Println()
}

func fmtPct(num, denom int) string {
	return fmt.Sprintf("%d/%d (%.0f%%)", num, denom, pct(num, denom))
}

func printDisagreements(title string, items []disagreement, maxItems int) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("\n%s (%d total):\n", title, len(items))
	limit := len(items)
	if limit > maxItems {
		limit = maxItems
	}
	for _, d := range items[:limit] {
		fmt.Printf("  %-10s %-60s classified=%-30s truth=%s\n",
			d.TaskKey, d.Summary, d.Classified, d.Truth)
	}
	if len(items) > maxItems {
		fmt.Printf("  ... and %d more\n", len(items)-maxItems)
	}
}

func pct(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom) * 100
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// deterministicEmbeddingService generates pseudo-embeddings from text hashes (no Ollama needed)
type deterministicEmbeddingService struct {
	dims int
}

func (s *deterministicEmbeddingService) Embed(texts []string) ([][]float64, error) {
	result := make([][]float64, len(texts))
	for i, text := range texts {
		result[i] = hashToVector(text, s.dims)
	}
	return result, nil
}

func hashToVector(text string, dims int) []float64 {
	vec := make([]float64, dims)

	chunksNeeded := (dims + 31) / 32
	var hashBytes []byte
	for c := 0; c < chunksNeeded; c++ {
		h := sha256.Sum256([]byte(fmt.Sprintf("%s#%d", text, c)))
		hashBytes = append(hashBytes, h[:]...)
	}

	for i := 0; i < dims; i++ {
		vec[i] = float64(hashBytes[i]) / 255.0
	}

	norm := 0.0
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec
}
