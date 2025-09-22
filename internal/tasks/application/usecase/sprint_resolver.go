package usecase

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	sprintPorts "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

const (
	reasonExactMatch = "exact match"
)

// SprintResolver resolves sprint names to actual sprint entities with interactive selection
type SprintResolver struct {
	sprintPort    sprintPorts.JiraPort
	selectionPort ports.SprintSelectionPort
}

// NewSprintResolver creates a new sprint resolver
func NewSprintResolver(sprintPort sprintPorts.JiraPort, selectionPort ports.SprintSelectionPort) *SprintResolver {
	return &SprintResolver{
		sprintPort:    sprintPort,
		selectionPort: selectionPort,
	}
}

// ResolveSprint resolves a sprint name to a sprint entity with interactive fallback
// Returns resolved sprint name for use in task fetching, or error if resolution fails
func (sr *SprintResolver) ResolveSprint(_ context.Context, project, sprintName string) (string, error) {
	if sprintName == "" {
		return "", fmt.Errorf("sprint name is required")
	}

	// Step 1: Try exact match first
	exactSprint, err := sr.sprintPort.GetSprintByName(project, sprintName)
	if err == nil && exactSprint != nil {
		return exactSprint.Name, nil
	}

	// Step 2: If exact match fails, get all sprints and try fuzzy matching
	allSprints, err := sr.sprintPort.GetSprintsForProject(project, []string{})
	if err != nil {
		return "", fmt.Errorf("failed to get sprints for project %s: %w", project, err)
	}

	if len(allSprints) == 0 {
		return "", fmt.Errorf("no sprints found for project %s", project)
	}

	// Step 3: Apply fuzzy matching to find candidates
	candidates := sr.findSprintCandidates(allSprints, sprintName)

	if len(candidates) == 0 {
		return "", fmt.Errorf("no sprints matching '%s' found in project %s", sprintName, project)
	}

	// Step 4: If single candidate, return it directly
	if len(candidates) == 1 {
		fmt.Printf("🔍 Found similar sprint: %s (ID: %s)\n", candidates[0].Sprint.Name, candidates[0].Sprint.ID)
		return candidates[0].Sprint.Name, nil
	}

	// Step 5: Multiple candidates - use interactive selection
	selectedSprint, err := sr.selectionPort.SelectSprint(candidates)
	if err != nil {
		return "", fmt.Errorf("failed to select sprint: %w", err)
	}

	if selectedSprint == nil {
		return "", fmt.Errorf("sprint selection cancelled by user")
	}

	return selectedSprint.Name, nil
}

// findSprintCandidates finds sprint candidates using fuzzy matching
func (sr *SprintResolver) findSprintCandidates(sprints []sprintPorts.Sprint, targetName string) []ports.SprintCandidate {
	candidates := make([]ports.SprintCandidate, 0)
	normalizedTarget := sr.normalizeSprintName(targetName)

	for _, sprint := range sprints {
		normalizedSprint := sr.normalizeSprintName(sprint.Name)

		// Check for different types of matches
		var matchReason string
		matched := false

		// 1. Exact match after normalization
		if normalizedSprint == normalizedTarget {
			matchReason = reasonExactMatch
			matched = true
		} else if strings.Contains(normalizedSprint, normalizedTarget) {
			// 2. Contains match
			matchReason = fmt.Sprintf("name contains '%s'", targetName)
			matched = true
		} else if strings.Contains(normalizedTarget, normalizedSprint) {
			// 3. Reverse contains (user typed more than sprint name)
			matchReason = fmt.Sprintf("partial match with '%s'", sprint.Name)
			matched = true
		} else if sr.calculateSimilarity(normalizedSprint, normalizedTarget) >= 0.6 {
			// 4. Similarity-based match (60% threshold)
			matchReason = fmt.Sprintf("similar to '%s'", targetName)
			matched = true
		}

		if matched {
			candidates = append(candidates, ports.SprintCandidate{
				Sprint: sprint,
				Reason: matchReason,
			})
		}
	}

	// Sort candidates by relevance (exact matches first, then by name)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Reason == reasonExactMatch && candidates[j].Reason != reasonExactMatch {
			return true
		}
		if candidates[j].Reason == reasonExactMatch && candidates[i].Reason != reasonExactMatch {
			return false
		}
		return candidates[i].Sprint.Name < candidates[j].Sprint.Name
	})

	return candidates
}

// normalizeSprintName normalizes sprint names for comparison by removing emojis and extra whitespace
func (sr *SprintResolver) normalizeSprintName(name string) string {
	// Remove emoji characters
	name = sr.removeEmojis(name)

	// Convert to lowercase and trim whitespace
	name = strings.ToLower(strings.TrimSpace(name))

	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	name = spaceRegex.ReplaceAllString(name, " ")

	return name
}

// removeEmojis removes emoji characters from a string
func (sr *SprintResolver) removeEmojis(input string) string {
	// Remove flag emojis (regional indicators)
	flagRegex := regexp.MustCompile(`[\x{1F1E6}-\x{1F1FF}]{2}`)
	input = flagRegex.ReplaceAllString(input, "")

	// Remove other emoji ranges
	emojiRegex := regexp.MustCompile(`[\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F1E0}-\x{1F1FF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]`)
	input = emojiRegex.ReplaceAllString(input, "")

	// Remove any remaining non-ASCII characters that might be emojis
	result := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1 // Remove non-ASCII characters
		}
		return r
	}, input)

	return strings.TrimSpace(result)
}

// calculateSimilarity calculates similarity between two strings using Levenshtein distance
func (sr *SprintResolver) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	distance := sr.levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (sr *SprintResolver) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}

	for j := 1; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = minInt(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// minInt returns the minimum of three integers
func minInt(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
