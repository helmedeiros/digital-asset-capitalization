package llama

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError string
	}{
		{
			name: "valid configuration",
			config: Config{
				BaseURL: "http://localhost:11434",
			},
		},
		{
			name: "empty base URL",
			config: Config{
				BaseURL: "",
			},
			expectedError: "OLLAMA_API_URL environment variable must be set",
		},
		{
			name: "from environment variables",
			config: Config{
				BaseURL: "http://custom-ollama:11434",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "from environment variables" {
				os.Setenv("OLLAMA_API_URL", "http://custom-ollama:11434")
				defer os.Unsetenv("OLLAMA_API_URL")
			}

			client, err := NewClient(tt.config)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tt.config.BaseURL, client.baseURL)
		})
	}
}

func TestEnrichContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		field         string
		asset         *domain.Asset
		mockResponse  string
		mockStatus    int
		expectedError string
	}{
		{
			name:    "successful enrichment",
			content: "Test content",
			field:   "description",
			asset: &domain.Asset{
				Name:     "Test Asset",
				Why:      "Test Why",
				Benefits: "Test Benefits",
				How:      "Test How",
				Metrics:  "Test Metrics",
			},
			mockResponse: `{"response": "Enriched content"}`,
			mockStatus:   http.StatusOK,
		},
		{
			name:    "API error",
			content: "Test content",
			field:   "description",
			asset: &domain.Asset{
				Name:     "Test Asset",
				Why:      "Test Why",
				Benefits: "Test Benefits",
				How:      "Test How",
				Metrics:  "Test Metrics",
			},
			mockResponse:  `{"error": "API error"}`,
			mockStatus:    http.StatusInternalServerError,
			expectedError: "API request failed with status 500: {\"error\": \"API error\"}",
		},
		{
			name:    "empty response",
			content: "Test content",
			field:   "description",
			asset: &domain.Asset{
				Name:     "Test Asset",
				Why:      "Test Why",
				Benefits: "Test Benefits",
				How:      "Test How",
				Metrics:  "Test Metrics",
			},
			mockResponse:  `{"response": ""}`,
			mockStatus:    http.StatusOK,
			expectedError: "no response from Ollama",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/generate", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewClient(Config{BaseURL: server.URL})
			require.NoError(t, err)

			result, err := client.EnrichContent(tt.content, tt.field, tt.asset)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "Enriched content", result)
		})
	}
}

func TestClose(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "http://localhost:11434"})
	require.NoError(t, err)

	err = client.Close()
	require.NoError(t, err)
}

func TestDefaultConfig(t *testing.T) {
	t.Run("should use default URL when env var not set", func(t *testing.T) {
		// Ensure env var is not set
		originalURL := os.Getenv("OLLAMA_API_URL")
		os.Unsetenv("OLLAMA_API_URL")
		defer func() {
			if originalURL != "" {
				os.Setenv("OLLAMA_API_URL", originalURL)
			}
		}()

		config := DefaultConfig()

		assert.Equal(t, "http://localhost:11434", config.BaseURL, "Should use default localhost URL")
	})

	t.Run("should use env var when set", func(t *testing.T) {
		// Set custom env var
		originalURL := os.Getenv("OLLAMA_API_URL")
		customURL := "http://custom-ollama:8080"
		os.Setenv("OLLAMA_API_URL", customURL)
		defer func() {
			if originalURL != "" {
				os.Setenv("OLLAMA_API_URL", originalURL)
			} else {
				os.Unsetenv("OLLAMA_API_URL")
			}
		}()

		config := DefaultConfig()

		assert.Equal(t, customURL, config.BaseURL, "Should use custom URL from env var")
	})

	t.Run("should handle empty env var", func(t *testing.T) {
		// Set empty env var
		originalURL := os.Getenv("OLLAMA_API_URL")
		os.Setenv("OLLAMA_API_URL", "")
		defer func() {
			if originalURL != "" {
				os.Setenv("OLLAMA_API_URL", originalURL)
			} else {
				os.Unsetenv("OLLAMA_API_URL")
			}
		}()

		config := DefaultConfig()

		assert.Equal(t, "http://localhost:11434", config.BaseURL, "Should use default URL when env var is empty")
	})

	t.Run("should return config struct with baseURL field", func(t *testing.T) {
		config := DefaultConfig()

		// Verify it's a proper Config struct
		assert.IsType(t, Config{}, config, "Should return Config struct")
		assert.NotEmpty(t, config.BaseURL, "BaseURL should not be empty")
	})
}
