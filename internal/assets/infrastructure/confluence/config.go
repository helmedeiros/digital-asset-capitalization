package confluence

import (
	"os"
)

// Config holds the configuration for the Confluence adapter
type Config struct {
	// BaseURL is the base URL of your Confluence instance
	BaseURL string
	// SpaceKey is the Confluence space key (e.g. MZN)
	SpaceKey string
	// Label is the label to filter pages by (e.g. cap-asset)
	Label string
	// Token is the Confluence token for authentication
	Token string
	// Username is the Confluence username for authentication
	Username string
	// MaxResults is the maximum number of results to fetch per page
	MaxResults int
	// Debug enables debug logging
	Debug bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxResults: 25,
		Username:   os.Getenv("JIRA_EMAIL"),
		Token:      os.Getenv("JIRA_TOKEN"),
		BaseURL:    os.Getenv("JIRA_BASE_URL"),
		Debug:      false,
	}
}
