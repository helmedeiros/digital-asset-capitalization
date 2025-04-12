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

// Config holds the configuration for the JIRA client
type Config struct {
	BaseURL string
	Email   string
	Token   string
}

// ConfigFactory is a function type for creating new Jira configurations
type ConfigFactory func() (*Config, error)

// NewConfig is the default implementation of ConfigFactory
var NewConfig ConfigFactory = newConfig

// newConfig creates a new Jira configuration instance
func newConfig() (*Config, error) {
	baseURL := os.Getenv(envJiraBaseURL)
	email := os.Getenv(envJiraEmail)
	token := os.Getenv(envJiraToken)

	config := &Config{
		BaseURL: baseURL,
		Email:   email,
		Token:   token,
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
	if c.BaseURL == "" {
		return ErrMissingBaseURL
	}

	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil || !strings.HasPrefix(parsedURL.Scheme, "http") {
		return ErrInvalidBaseURL
	}

	return nil
}

// validateCredentials checks if the email and token are present
func (c *Config) validateCredentials() error {
	if c.Email == "" {
		return ErrMissingEmail
	}
	if c.Token == "" {
		return ErrMissingToken
	}
	return nil
}

// GetBaseURL returns the configured Jira base URL
func (c *Config) GetBaseURL() string {
	return c.BaseURL
}

// GetEmail returns the configured Jira user email
func (c *Config) GetEmail() string {
	return c.Email
}

// GetToken returns the configured Jira API token
func (c *Config) GetToken() string {
	return c.Token
}

// GetAuthHeader returns the base64 encoded authentication header for Jira API
func (c *Config) GetAuthHeader() string {
	authString := fmt.Sprintf("%s:%s", c.Email, c.Token)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(authString))
}
