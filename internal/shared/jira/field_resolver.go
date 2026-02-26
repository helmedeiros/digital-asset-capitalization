package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CustomFieldIDs holds the discovered JIRA custom field IDs
type CustomFieldIDs struct {
	TPDBusinessUnit  string
	EngineeringHours string
	WorkStream       string
}

// Field represents a JIRA field definition
type Field struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Schema struct {
		Type   string `json:"type"`
		Custom string `json:"custom"`
	} `json:"schema"`
}

// FieldResolver discovers custom field IDs from the JIRA API
type FieldResolver struct {
	baseURL    string
	authHeader string
	httpClient *http.Client
	cache      *CustomFieldIDs
	mu         sync.Mutex
}

// NewFieldResolver creates a new FieldResolver
func NewFieldResolver(baseURL, authHeader string) *FieldResolver {
	return &FieldResolver{
		baseURL:    baseURL,
		authHeader: authHeader,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ResolveCustomFieldIDs discovers the custom field IDs for TPD Business Unit,
// Engineering time spent (hours), and Work Stream by name matching
func (r *FieldResolver) ResolveCustomFieldIDs() (*CustomFieldIDs, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cache != nil {
		return r.cache, nil
	}

	url := fmt.Sprintf("%s/rest/api/3/field", r.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating field request: %w", err)
	}

	req.Header.Set("Authorization", r.authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching fields: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching fields", resp.StatusCode)
	}

	var fields []Field
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil, fmt.Errorf("error decoding fields response: %w", err)
	}

	ids := &CustomFieldIDs{}
	for _, field := range fields {
		switch field.Name {
		case "TPD Business Unit":
			ids.TPDBusinessUnit = field.ID
		case "Engineering time spent (hours)":
			ids.EngineeringHours = field.ID
		case "Work Stream":
			ids.WorkStream = field.ID
		}
	}

	r.cache = ids
	return ids, nil
}
