package config

import "errors"

var (
	ErrMissingBaseURL = errors.New("JIRA_BASE_URL environment variable is not set")
	ErrMissingEmail   = errors.New("JIRA_EMAIL environment variable is not set")
	ErrMissingToken   = errors.New("JIRA_TOKEN environment variable is not set")
	ErrInvalidBaseURL = errors.New("JIRA_BASE_URL must be a valid URL")
)
