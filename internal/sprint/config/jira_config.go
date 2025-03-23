package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Environment variable names for Jira configuration
const (
	envJiraBaseURL = "JIRA_BASE_URL"
	envJiraEmail   = "JIRA_EMAIL"
	envJiraToken   = "JIRA_TOKEN"
)

// JiraConfig holds the configuration for Jira API authentication and connection
type JiraConfig struct {
	baseURL string
	email   string
	token   string
}

// NewJiraConfig creates a new JiraConfig instance from environment variables
func NewJiraConfig() (*JiraConfig, error) {
	config := &JiraConfig{
		baseURL: os.Getenv(envJiraBaseURL),
		email:   os.Getenv(envJiraEmail),
		token:   os.Getenv(envJiraToken),
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Jira configuration: %w", err)
	}

	return config, nil
}

// Validate checks if all required configuration values are present and valid
func (c *JiraConfig) Validate() error {
	if err := c.validateBaseURL(); err != nil {
		return err
	}
	if err := c.validateCredentials(); err != nil {
		return err
	}

	return nil
}

// validateBaseURL checks if the base URL is present and valid
func (c *JiraConfig) validateBaseURL() error {
	if c.baseURL == "" {
		return ErrMissingBaseURL
	}

	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return ErrInvalidBaseURL
	}
	if !strings.HasPrefix(parsedURL.Scheme, "http") {
		return ErrInvalidBaseURL
	}

	return nil
}

// validateCredentials checks if the email and token are present
func (c *JiraConfig) validateCredentials() error {
	if c.email == "" {
		return ErrMissingEmail
	}
	if c.token == "" {
		return ErrMissingToken
	}

	return nil
}

// GetBaseURL returns the configured Jira base URL
func (c *JiraConfig) GetBaseURL() string {
	return c.baseURL
}

// GetEmail returns the configured Jira user email
func (c *JiraConfig) GetEmail() string {
	return c.email
}

// GetAuthHeader returns the base64 encoded authentication header for Jira API
func (c *JiraConfig) GetAuthHeader() string {
	authString := fmt.Sprintf("%s:%s", c.email, c.token)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(authString))
}
