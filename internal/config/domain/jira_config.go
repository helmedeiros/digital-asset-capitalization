package domain

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// JiraConfig represents the Jira configuration domain entity
type JiraConfig struct {
	baseURL string
	email   string
	token   string
}

// NewJiraConfig creates a new JiraConfig with validation
func NewJiraConfig(baseURL, email, token string) (*JiraConfig, error) {
	config := &JiraConfig{
		baseURL: strings.TrimSpace(baseURL),
		email:   strings.TrimSpace(email),
		token:   strings.TrimSpace(token),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// validate performs domain validation rules
func (c *JiraConfig) validate() error {
	if c.baseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if err := c.validateBaseURL(); err != nil {
		return err
	}

	if c.email == "" {
		return fmt.Errorf("email is required")
	}

	if err := c.validateEmail(); err != nil {
		return err
	}

	if c.token == "" {
		return fmt.Errorf("token is required")
	}

	return nil
}

// validateBaseURL validates the base URL format
func (c *JiraConfig) validateBaseURL() error {
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL format: %w", err)
	}

	if !strings.HasPrefix(parsedURL.Scheme, "http") {
		return fmt.Errorf("invalid base URL format: must start with http or https")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("invalid base URL format: missing host")
	}

	return nil
}

// validateEmail validates the email format
func (c *JiraConfig) validateEmail() error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(c.email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// BaseURL returns the base URL
func (c *JiraConfig) BaseURL() string {
	return c.baseURL
}

// Email returns the email
func (c *JiraConfig) Email() string {
	return c.email
}

// Token returns the token
func (c *JiraConfig) Token() string {
	return c.token
}

// IsValid checks if the configuration is valid
func (c *JiraConfig) IsValid() bool {
	return c.validate() == nil
}

// AuthHeader returns the base64 encoded authentication header
func (c *JiraConfig) AuthHeader() string {
	auth := fmt.Sprintf("%s:%s", c.email, c.token)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	return fmt.Sprintf("Basic %s", encoded)
}

// String returns a string representation (with masked token)
func (c *JiraConfig) String() string {
	maskedToken := "***"
	if len(c.token) > 0 {
		maskedToken = c.token[:minInt(3, len(c.token))] + "***"
	}
	return fmt.Sprintf("JiraConfig{BaseURL: %s, Email: %s, Token: %s}", c.baseURL, c.email, maskedToken)
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
