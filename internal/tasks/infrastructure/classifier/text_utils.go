package classifier

import (
	"regexp"
	"strings"
	"unicode"
)

// TextMatcher provides advanced text matching capabilities for asset classification
type TextMatcher struct {
	// Regex patterns for different text formats
	camelCasePattern *regexp.Regexp
	snakeCasePattern *regexp.Regexp
	kebabCasePattern *regexp.Regexp
}

// NewTextMatcher creates a new text matcher with precompiled patterns
func NewTextMatcher() *TextMatcher {
	return &TextMatcher{
		camelCasePattern: regexp.MustCompile(`([a-z])([A-Z])`), // matches camelCase boundaries
		snakeCasePattern: regexp.MustCompile(`[_-]+`),          // matches snake_case and kebab-case separators
		kebabCasePattern: regexp.MustCompile(`[-_]+`),          // matches kebab-case and snake_case separators
	}
}

// MatchesAssetName checks if the content contains the asset name in any common format
func (tm *TextMatcher) MatchesAssetName(content, assetName string) bool {
	// Handle empty inputs
	if content == "" || assetName == "" {
		return false
	}

	contentLower := strings.ToLower(content)
	assetNameLower := strings.ToLower(assetName)

	// 1. Exact match
	if strings.Contains(contentLower, assetNameLower) {
		return true
	}

	// 2. Generate all possible variations of the asset name
	variations := tm.generateNameVariations(assetNameLower)

	// 3. Check each variation
	for _, variation := range variations {
		if strings.Contains(contentLower, variation) {
			return true
		}
	}

	return false
}

// MatchesKeyword checks if content contains a keyword in any common format
func (tm *TextMatcher) MatchesKeyword(content, keyword string) bool {
	// Handle empty inputs
	if content == "" || keyword == "" {
		return false
	}

	contentLower := strings.ToLower(content)
	keywordLower := strings.ToLower(keyword)

	// 1. Exact match
	if strings.Contains(contentLower, keywordLower) {
		return true
	}

	// 2. Generate variations for multi-word keywords
	if strings.Contains(keywordLower, " ") {
		variations := tm.generateNameVariations(keywordLower)
		for _, variation := range variations {
			if strings.Contains(contentLower, variation) {
				return true
			}
		}
	}

	return false
}

// generateNameVariations creates all common variations of a name
func (tm *TextMatcher) generateNameVariations(name string) []string {
	variations := []string{name} // Start with original

	// Split the name into words
	words := tm.splitIntoWords(name)
	if len(words) <= 1 {
		return variations
	}

	// Add space-separated version if the original doesn't have spaces
	if !strings.Contains(name, " ") {
		variations = append(variations, strings.Join(words, " "))
	}

	// Generate different case variations
	variations = append(variations, tm.toCamelCase(words))
	variations = append(variations, tm.toPascalCase(words))
	variations = append(variations, tm.toSnakeCase(words))
	variations = append(variations, tm.toKebabCase(words))
	variations = append(variations, tm.toConcatenated(words))
	variations = append(variations, tm.toUpperConcatenated(words))

	// Remove duplicates and empty strings
	return tm.uniqueNonEmpty(variations)
}

// splitIntoWords splits a name into individual words handling various separators
func (tm *TextMatcher) splitIntoWords(name string) []string {
	// Handle different separators
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Handle camelCase by adding spaces before uppercase letters
	name = tm.camelCasePattern.ReplaceAllString(name, "${1} ${2}")

	// Split by spaces and filter empty strings
	words := strings.Fields(name)

	// Convert to lowercase
	for i, word := range words {
		words[i] = strings.ToLower(word)
	}

	return words
}

// toCamelCase converts words to camelCase (first word lowercase, rest PascalCase)
func (tm *TextMatcher) toCamelCase(words []string) string {
	if len(words) == 0 {
		return ""
	}

	result := words[0]
	for i := 1; i < len(words); i++ {
		result += tm.capitalize(words[i])
	}

	return result
}

// toPascalCase converts words to PascalCase (all words capitalized)
func (tm *TextMatcher) toPascalCase(words []string) string {
	if len(words) == 0 {
		return ""
	}

	result := ""
	for _, word := range words {
		result += tm.capitalize(word)
	}

	return result
}

// toSnakeCase converts words to snake_case
func (tm *TextMatcher) toSnakeCase(words []string) string {
	return strings.Join(words, "_")
}

// toKebabCase converts words to kebab-case
func (tm *TextMatcher) toKebabCase(words []string) string {
	return strings.Join(words, "-")
}

// toConcatenated converts words to concatenated lowercase
func (tm *TextMatcher) toConcatenated(words []string) string {
	return strings.Join(words, "")
}

// toUpperConcatenated converts words to concatenated uppercase
func (tm *TextMatcher) toUpperConcatenated(words []string) string {
	return strings.ToUpper(strings.Join(words, ""))
}

// capitalize capitalizes the first letter of a word
func (tm *TextMatcher) capitalize(word string) string {
	if len(word) == 0 {
		return word
	}

	runes := []rune(word)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// uniqueNonEmpty removes duplicates and empty strings from a slice
func (tm *TextMatcher) uniqueNonEmpty(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Default text matcher instance
var defaultTextMatcher = NewTextMatcher()

// MatchesAssetNameEnhanced is a convenience function using the default text matcher
func MatchesAssetNameEnhanced(content, assetName string) bool {
	return defaultTextMatcher.MatchesAssetName(content, assetName)
}

// MatchesKeywordEnhanced is a convenience function using the default text matcher
func MatchesKeywordEnhanced(content, keyword string) bool {
	return defaultTextMatcher.MatchesKeyword(content, keyword)
}
