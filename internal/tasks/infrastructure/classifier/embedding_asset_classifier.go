package classifier

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"sync"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	assetports "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

var taskKeyPattern = regexp.MustCompile(`[A-Z]+-\d+`)

const maxTaskTextLen = 2000

// EmbeddingAssetClassifier implements asset classification using embedding vectors and cosine similarity
type EmbeddingAssetClassifier struct {
	embeddingService ports.EmbeddingService
	assetRepo        assetports.AssetRepository
	store            *EmbeddingStore
	cachedAssets     []*assetdomain.Asset
	assetsMu         sync.Once
	assetsErr        error
	// historicalTasks maps asset ID (e.g. "cap-asset-carrier-comparison-optimization")
	// to task summaries previously classified to that asset. Used to enrich asset embeddings.
	historicalTasks map[string][]string
	// epicNames maps epic keys (e.g. "COP-2") to their human-readable summaries.
	// Used to replace opaque JIRA keys with meaningful text in task embeddings.
	epicNames map[string]string
	// epicAssetHint maps epic keys to their dominant asset ID (e.g. "COP-19" → "cap-asset-carrier-comparison-optimization").
	// Used to boost the cosine score of the likely asset based on epic neighborhood.
	epicAssetHint map[string]string
}

// NewEmbeddingAssetClassifier creates a new embedding-based asset classifier
func NewEmbeddingAssetClassifier(embeddingService ports.EmbeddingService, assetRepo assetports.AssetRepository, store *EmbeddingStore) ports.AssetClassifier {
	return &EmbeddingAssetClassifier{
		embeddingService: embeddingService,
		assetRepo:        assetRepo,
		store:            store,
	}
}

// NewEmbeddingAssetClassifierWithHistory creates an embedding classifier enriched with historical task associations.
// historicalTasks maps asset ID (cap-asset-*) to a slice of task summaries previously classified to that asset.
// epicNames maps epic keys (e.g. "COP-2") to their summaries for resolving opaque JIRA keys.
// epicAssetHint maps epic keys to dominant asset IDs for score boosting (can be nil).
func NewEmbeddingAssetClassifierWithHistory(embeddingService ports.EmbeddingService, assetRepo assetports.AssetRepository, store *EmbeddingStore, historicalTasks map[string][]string, epicNames map[string]string, epicAssetHint map[string]string) ports.AssetClassifier {
	return &EmbeddingAssetClassifier{
		embeddingService: embeddingService,
		assetRepo:        assetRepo,
		store:            store,
		historicalTasks:  historicalTasks,
		epicNames:        epicNames,
		epicAssetHint:    epicAssetHint,
	}
}

// loadAssets fetches and caches all assets
func (c *EmbeddingAssetClassifier) loadAssets() ([]*assetdomain.Asset, error) {
	c.assetsMu.Do(func() {
		c.cachedAssets, c.assetsErr = c.assetRepo.FindAll()
	})
	return c.cachedAssets, c.assetsErr
}

// ClassifyTaskAsset classifies a single task by embedding similarity
func (c *EmbeddingAssetClassifier) ClassifyTaskAsset(task *taskdomain.Task) (*ports.AssetClassificationResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	results, err := c.ClassifyTasksAssets([]*taskdomain.Task{task})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &ports.AssetClassificationResult{
			Task:       task,
			Confidence: 0,
			Reason:     "no classification result",
		}, nil
	}

	return results[0], nil
}

// ClassifyTasksAssets classifies multiple tasks using batch embedding and cosine similarity
func (c *EmbeddingAssetClassifier) ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	if len(tasks) == 0 {
		return []*ports.AssetClassificationResult{}, nil
	}

	allAssets, err := c.loadAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	// Filter out cap-asset-* named entries that pollute the candidate pool
	assets := filterRealAssets(allAssets)

	if len(assets) == 0 {
		results := make([]*ports.AssetClassificationResult, len(tasks))
		for i, task := range tasks {
			results[i] = &ports.AssetClassificationResult{
				Task:       task,
				Confidence: 0,
				Reason:     "no assets available for embedding classification",
			}
		}
		return results, nil
	}

	// Step 1: Ensure all asset embeddings are cached
	assetVectors, err := c.ensureAssetEmbeddings(assets)
	if err != nil {
		return nil, fmt.Errorf("failed to compute asset embeddings: %w", err)
	}

	// Step 2: Embed all task texts in a single batch call
	taskTexts := make([]string, len(tasks))
	for i, task := range tasks {
		taskTexts[i] = buildTaskTextWithEpics(task, c.epicNames)
	}

	taskVectors, err := c.embeddingService.Embed(taskTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed tasks: %w", err)
	}

	// Step 3: For each task, find best matching asset by cosine similarity with epic boost
	results := make([]*ports.AssetClassificationResult, len(tasks))
	for i, task := range tasks {
		if i >= len(taskVectors) {
			results[i] = &ports.AssetClassificationResult{
				Task:       task,
				Confidence: 0,
				Reason:     "missing task embedding",
			}
			continue
		}

		bestAsset, bestScore := c.findBestMatchWithEpicBoost(taskVectors[i], assets, assetVectors, task)
		if bestAsset == nil {
			results[i] = &ports.AssetClassificationResult{
				Task:       task,
				Confidence: 0,
				Reason:     "no matching asset found via embeddings",
			}
			continue
		}

		// Normalize cosine similarity (typically 0.5-1.0 range) to a 0-1 confidence
		confidence := normalizeScore(bestScore)

		results[i] = &ports.AssetClassificationResult{
			Task:       task,
			Asset:      bestAsset,
			Confidence: confidence,
			Reason:     fmt.Sprintf("embedding similarity: %.3f (normalized: %.2f)", bestScore, confidence),
		}
	}

	// Step 4: Persist any new asset embeddings
	if err := c.store.Save(); err != nil {
		log.Printf("Warning: failed to save embedding cache: %v", err)
	}

	return results, nil
}

// ensureAssetEmbeddings returns embedding vectors for all assets, computing missing ones via the embedding service
func (c *EmbeddingAssetClassifier) ensureAssetEmbeddings(assets []*assetdomain.Asset) ([][]float64, error) {
	vectors := make([][]float64, len(assets))
	var missingIndices []int
	var missingTexts []string

	for i, asset := range assets {
		text := buildAssetTextWithHistory(asset, c.historicalTasks)
		hash := HashText(text)

		if vec, ok := c.store.Get(asset.Name, hash); ok {
			vectors[i] = vec
		} else {
			missingIndices = append(missingIndices, i)
			missingTexts = append(missingTexts, text)
		}
	}

	if len(missingTexts) == 0 {
		return vectors, nil
	}

	log.Printf("Computing embeddings for %d assets (cached: %d)", len(missingTexts), len(assets)-len(missingTexts))

	newVectors, err := c.embeddingService.Embed(missingTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed assets: %w", err)
	}

	for j, idx := range missingIndices {
		if j >= len(newVectors) {
			break
		}
		vectors[idx] = newVectors[j]

		text := buildAssetTextWithHistory(assets[idx], c.historicalTasks)
		c.store.Set(assets[idx].Name, newVectors[j], HashText(text))
	}

	return vectors, nil
}

// buildAssetText composes a rich text representation of an asset for embedding (without history)
func buildAssetText(asset *assetdomain.Asset) string {
	return buildAssetTextWithHistory(asset, nil)
}

// buildAssetTextWithHistory composes a rich text representation of an asset for embedding,
// optionally enriched with summaries of historically classified tasks.
func buildAssetTextWithHistory(asset *assetdomain.Asset, historicalTasks map[string][]string) string {
	parts := []string{"Asset: " + asset.Name}

	if asset.Description != "" {
		parts = append(parts, asset.Description)
	}
	if asset.Why != "" {
		parts = append(parts, asset.Why)
	}
	if asset.Benefits != "" {
		parts = append(parts, asset.Benefits)
	}
	if asset.How != "" {
		parts = append(parts, asset.How)
	}
	if asset.Metrics != "" {
		parts = append(parts, asset.Metrics)
	}
	if len(asset.Keywords) > 0 {
		parts = append(parts, "Keywords: "+strings.Join(asset.Keywords, " "))
	}

	// Append historical task summaries to anchor the asset in real task language
	if historicalTasks != nil {
		if summaries, ok := historicalTasks[asset.ID]; ok && len(summaries) > 0 {
			parts = append(parts, "Related tasks: "+strings.Join(summaries, "; "))
		}
	}

	return strings.Join(parts, ". ")
}

// buildTaskText composes task text for embedding, truncated to maxTaskTextLen (without epic resolution)
func buildTaskText(task *taskdomain.Task) string {
	return buildTaskTextWithEpics(task, nil)
}

// buildTaskTextWithEpics composes task text for embedding, resolving epic keys and
// task-key references in the summary to human-readable names when possible.
func buildTaskTextWithEpics(task *taskdomain.Task, epicNames map[string]string) string {
	// Resolve task-key references in summary (e.g. "Re-run COP-39 experiment" → "Re-run AB 10% Voucher test on SRP banner experiment")
	summary := task.Summary
	if epicNames != nil {
		summary = taskKeyPattern.ReplaceAllStringFunc(summary, func(key string) string {
			if key == task.Key {
				return key // don't replace self-reference
			}
			if name, ok := epicNames[key]; ok {
				return name
			}
			return key
		})
	}

	text := "Task: " + summary
	if task.Description != "" {
		text += " " + task.Description
	}
	if task.Epic != "" {
		epicText := task.Epic
		if epicNames != nil {
			if name, ok := epicNames[task.Epic]; ok {
				epicText = name
			}
		}
		text += " Epic: " + epicText
	}
	if task.Project != "" {
		text += " Project: " + task.Project
	}
	// Include non-cap-asset labels (e.g., cap-development, cap-maintenance)
	// Exclude cap-asset-* labels to avoid leaking ground truth
	for _, label := range task.Labels {
		if !strings.HasPrefix(label, "cap-asset-") && label != "" {
			text += " " + label
		}
	}

	if len(text) > maxTaskTextLen {
		text = text[:maxTaskTextLen]
	}

	return text
}

// filterRealAssets removes entries whose Name starts with "cap-asset-" from the candidate pool.
// These are duplicate/alias entries that pollute cosine similarity results.
func filterRealAssets(assets []*assetdomain.Asset) []*assetdomain.Asset {
	filtered := make([]*assetdomain.Asset, 0, len(assets))
	for _, a := range assets {
		if !strings.HasPrefix(strings.ToLower(a.Name), "cap-asset-") {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

const epicBoostWeight = 0.05

// findBestMatchWithEpicBoost finds the best matching asset, boosting the score of the
// epic's dominant asset if the task's epic has a strong mapping.
func (c *EmbeddingAssetClassifier) findBestMatchWithEpicBoost(query []float64, assets []*assetdomain.Asset, assetVectors [][]float64, task *taskdomain.Task) (*assetdomain.Asset, float64) {
	if c.epicAssetHint == nil || task.Epic == "" {
		return findBestMatch(query, assets, assetVectors)
	}

	hintAssetID, hasHint := c.epicAssetHint[task.Epic]
	if !hasHint {
		return findBestMatch(query, assets, assetVectors)
	}

	// Get top candidates and apply epic boost
	top := FindTopNMatches(query, assets, assetVectors, len(assets))
	if len(top) == 0 {
		return nil, -1.0
	}

	bestIdx := 0
	bestScore := top[0].Score
	for j, sa := range top {
		score := sa.Score
		if strings.EqualFold(sa.Asset.ID, hintAssetID) {
			score += epicBoostWeight
		}
		if score > bestScore {
			bestScore = score
			bestIdx = j
		}
	}

	return top[bestIdx].Asset, top[bestIdx].Score
}

// ScoredAsset pairs an asset with its similarity score
type ScoredAsset struct {
	Asset *assetdomain.Asset
	Score float64
}

// findBestMatch returns the asset with highest cosine similarity to the query vector
func findBestMatch(query []float64, assets []*assetdomain.Asset, assetVectors [][]float64) (*assetdomain.Asset, float64) {
	top := FindTopNMatches(query, assets, assetVectors, 1)
	if len(top) == 0 {
		return nil, -1.0
	}
	return top[0].Asset, top[0].Score
}

// FindTopNMatches returns the top N assets ranked by cosine similarity to the query vector
func FindTopNMatches(query []float64, assets []*assetdomain.Asset, assetVectors [][]float64, n int) []ScoredAsset {
	if n <= 0 || len(query) == 0 {
		return nil
	}

	type entry struct {
		idx   int
		score float64
	}

	var scored []entry
	for i, vec := range assetVectors {
		if len(vec) == 0 {
			continue
		}
		scored = append(scored, entry{idx: i, score: cosineSimilarity(query, vec)})
	}

	// Simple selection: sort descending by score
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if n > len(scored) {
		n = len(scored)
	}

	result := make([]ScoredAsset, n)
	for i := 0; i < n; i++ {
		result[i] = ScoredAsset{
			Asset: assets[scored[i].idx],
			Score: scored[i].score,
		}
	}
	return result
}

// cosineSimilarity computes the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}

	return dot / denom
}

// normalizeScore maps raw cosine similarity (typically 0.5-1.0 for related texts) to a 0-1 confidence scale
func normalizeScore(raw float64) float64 {
	// Empirical mapping: cosine sim of 0.5 -> confidence 0, 1.0 -> confidence 1.0
	normalized := (raw - 0.5) * 2.0
	if normalized < 0 {
		return 0
	}
	if normalized > 1.0 {
		return 1.0
	}
	return normalized
}
