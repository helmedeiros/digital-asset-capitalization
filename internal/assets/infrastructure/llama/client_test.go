package llama

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

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
			// 500 is now retryable, so the resulting error reports the
			// retry-budget exhaustion rather than the per-attempt body.
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
			expectedError: "Ollama request failed after 3 attempts: status 500",
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

			// Tight backoff so the retryable-status case doesn't wait
			// a second of real time per test invocation.
			client, err := NewClient(Config{BaseURL: server.URL, BackoffBase: time.Microsecond})
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

func TestGenerateEmbeddings(t *testing.T) {
	tests := []struct {
		name          string
		texts         []string
		mockResponse  string
		mockStatus    int
		expectedCount int
		expectedError string
	}{
		{
			name:          "successful single embedding",
			texts:         []string{"hello world"},
			mockResponse:  `{"embeddings": [[0.1, 0.2, 0.3]]}`,
			mockStatus:    http.StatusOK,
			expectedCount: 1,
		},
		{
			name:          "successful batch embeddings",
			texts:         []string{"text one", "text two", "text three"},
			mockResponse:  `{"embeddings": [[0.1, 0.2], [0.3, 0.4], [0.5, 0.6]]}`,
			mockStatus:    http.StatusOK,
			expectedCount: 3,
		},
		{
			name:          "empty input returns empty",
			texts:         []string{},
			mockResponse:  "",
			mockStatus:    http.StatusOK,
			expectedCount: 0,
		},
		{
			name:          "API error",
			texts:         []string{"test"},
			mockResponse:  `{"error": "model not found"}`,
			mockStatus:    http.StatusNotFound,
			expectedError: "Ollama embed returned status 404",
		},
		{
			name:          "count mismatch",
			texts:         []string{"one", "two"},
			mockResponse:  `{"embeddings": [[0.1, 0.2]]}`,
			mockStatus:    http.StatusOK,
			expectedError: "expected 2 embeddings, got 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.texts) == 0 {
				client, err := NewClient(Config{BaseURL: "http://localhost:11434"})
				require.NoError(t, err)
				result, err := client.GenerateEmbeddings("llama3", tt.texts)
				require.NoError(t, err)
				assert.Empty(t, result)
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/embed", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client, err := NewClient(Config{BaseURL: server.URL, BackoffBase: time.Microsecond})
			require.NoError(t, err)

			result, err := client.GenerateEmbeddings("llama3", tt.texts)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

func TestNewClientTimeout(t *testing.T) {
	t.Run("applies configured timeout", func(t *testing.T) {
		client, err := NewClient(Config{BaseURL: "http://example.invalid", Timeout: 7 * time.Second})
		require.NoError(t, err)
		assert.Equal(t, 7*time.Second, client.httpClient.Timeout)
	})

	t.Run("falls back to DefaultTimeout when unset", func(t *testing.T) {
		client, err := NewClient(Config{BaseURL: "http://example.invalid"})
		require.NoError(t, err)
		assert.Equal(t, DefaultTimeout, client.httpClient.Timeout)
	})

	t.Run("falls back to DefaultTimeout for non-positive values", func(t *testing.T) {
		client, err := NewClient(Config{BaseURL: "http://example.invalid", Timeout: -1 * time.Second})
		require.NoError(t, err)
		assert.Equal(t, DefaultTimeout, client.httpClient.Timeout)
	})

	t.Run("aborts requests that exceed the timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(Config{BaseURL: server.URL, Timeout: 25 * time.Millisecond})
		require.NoError(t, err)

		_, err = client.GenerateEmbeddings("llama3", []string{"hello"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to call Ollama embed")
	})
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

func TestIsRetryableStatus(t *testing.T) {
	cases := []struct {
		code      int
		retryable bool
	}{
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusNotFound, false},
		{http.StatusRequestTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
	}
	for _, c := range cases {
		assert.Equal(t, c.retryable, isRetryableStatus(c.code), "status %d", c.code)
	}
}

func TestDoWithRetry_RetriesTransientThenSucceeds(t *testing.T) {
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`busy`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"ok"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, MaxAttempts: 3, BackoffBase: time.Microsecond})
	require.NoError(t, err)

	resp, err := client.doWithRetry(func() (*http.Request, error) {
		return http.NewRequest("GET", server.URL+"/anything", nil)
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int64(3), atomic.LoadInt64(&hits), "should have retried twice before success")
}

func TestDoWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, MaxAttempts: 4, BackoffBase: time.Microsecond})
	require.NoError(t, err)

	resp, err := client.doWithRetry(func() (*http.Request, error) {
		return http.NewRequest("GET", server.URL+"/anything", nil)
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "after 4 attempts")
	assert.Equal(t, int64(4), atomic.LoadInt64(&hits))
}

func TestDoWithRetry_DoesNotRetryClientErrors(t *testing.T) {
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, MaxAttempts: 5, BackoffBase: time.Microsecond})
	require.NoError(t, err)

	resp, err := client.doWithRetry(func() (*http.Request, error) {
		return http.NewRequest("GET", server.URL+"/anything", nil)
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits), "4xx must not retry")
}

func TestDoWithRetry_RetriesTransportError(t *testing.T) {
	// Spin a server up to get an address, then close it so every attempt
	// fails at the transport layer. With BackoffBase set to a single
	// microsecond the retry loop costs effectively nothing.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, MaxAttempts: 3, BackoffBase: time.Microsecond})
	require.NoError(t, err)

	resp, err := client.doWithRetry(func() (*http.Request, error) {
		return http.NewRequest("GET", server.URL+"/anything", nil)
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "after 3 attempts")
}

func TestNewClient_AppliesRetryDefaults(t *testing.T) {
	t.Run("zero MaxAttempts falls back to DefaultMaxAttempts", func(t *testing.T) {
		client, err := NewClient(Config{BaseURL: "http://example.invalid"})
		require.NoError(t, err)
		assert.Equal(t, DefaultMaxAttempts, client.maxAttempts)
		assert.Equal(t, DefaultBackoffBase, client.backoffBase)
	})

	t.Run("explicit MaxAttempts of 1 disables retries", func(t *testing.T) {
		client, err := NewClient(Config{BaseURL: "http://example.invalid", MaxAttempts: 1})
		require.NoError(t, err)
		assert.Equal(t, 1, client.maxAttempts)
	})
}
