package config

import (
	"net/url"
	"os"
	"strings"
)

type JiraConfig struct {
	BaseURL string
	Email   string
	Token   string
}

func NewJiraConfig() (*JiraConfig, error) {
	config := &JiraConfig{
		BaseURL: os.Getenv("JIRA_BASE_URL"),
		Email:   os.Getenv("JIRA_EMAIL"),
		Token:   os.Getenv("JIRA_TOKEN"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *JiraConfig) Validate() error {
	if c.BaseURL == "" {
		return ErrMissingBaseURL
	}
	if c.Email == "" {
		return ErrMissingEmail
	}
	if c.Token == "" {
		return ErrMissingToken
	}

	// Validate URL format
	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil || !strings.HasPrefix(parsedURL.Scheme, "http") {
		return ErrInvalidBaseURL
	}

	return nil
}
