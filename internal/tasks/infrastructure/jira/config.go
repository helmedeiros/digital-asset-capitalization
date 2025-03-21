package jira

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	envJiraBaseURL = "JIRA_BASE_URL"
	envJiraEmail   = "JIRA_EMAIL"
	envJiraToken   = "JIRA_TOKEN"
)

// Config holds the configuration for Jira API
type Config struct {
	baseURL string
	email   string
	token   string
}

// ConfigFactory is a function type for creating new Jira configurations
type ConfigFactory func() (*Config, error)

// NewConfig is the default implementation of ConfigFactory
var NewConfig ConfigFactory = newConfig

// newConfig creates a new Jira configuration instance
func newConfig() (*Config, error) {
	baseURL := os.Getenv("JIRA_BASE_URL")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_TOKEN")

	config := &Config{
		baseURL: baseURL,
		email:   email,
		token:   token,
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Jira configuration: %w", err)
	}

	return config, nil
}

// Validate checks if all required configuration values are present and valid
func (c *Config) Validate() error {
	if err := c.validateBaseURL(); err != nil {
		return err
	}
	if err := c.validateCredentials(); err != nil {
		return err
	}
	return nil
}

// validateBaseURL checks if the base URL is present and valid
func (c *Config) validateBaseURL() error {
	if c.baseURL == "" {
		return ErrMissingBaseURL
	}

	parsedURL, err := url.Parse(c.baseURL)
	if err != nil || !strings.HasPrefix(parsedURL.Scheme, "http") {
		return ErrInvalidBaseURL
	}

	return nil
}

// validateCredentials checks if the email and token are present
func (c *Config) validateCredentials() error {
	if c.email == "" {
		return ErrMissingEmail
	}
	if c.token == "" {
		return ErrMissingToken
	}
	return nil
}

// GetBaseURL returns the configured Jira base URL
func (c *Config) GetBaseURL() string {
	return c.baseURL
}

// GetEmail returns the configured Jira user email
func (c *Config) GetEmail() string {
	return c.email
}

// GetToken returns the configured Jira API token
func (c *Config) GetToken() string {
	return c.token
}

// GetAuthHeader returns the base64 encoded authentication header for Jira API
func (c *Config) GetAuthHeader() string {
	authString := fmt.Sprintf("%s:%s", c.email, c.token)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(authString))
}
