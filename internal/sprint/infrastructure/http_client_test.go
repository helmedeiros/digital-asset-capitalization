package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClient_Get(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		authToken      string
		wantErr        bool
	}{
		{
			name:           "successful request",
			serverResponse: `{"issues": []}`,
			serverStatus:   http.StatusOK,
			authToken:      "Bearer test-token",
			wantErr:        false,
		},
		{
			name:           "server error",
			serverResponse: "Internal Server Error",
			serverStatus:   http.StatusInternalServerError,
			authToken:      "Bearer test-token",
			wantErr:        true,
		},
		{
			name:           "not found",
			serverResponse: "Not Found",
			serverStatus:   http.StatusNotFound,
			authToken:      "Bearer test-token",
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			serverResponse: "Unauthorized",
			serverStatus:   http.StatusUnauthorized,
			authToken:      "Bearer invalid-token",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader != tt.authToken {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
					return
				}
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := NewHTTPClient(server.URL, tt.authToken)

			_, err := client.Get(server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPClient.Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPClient_GetJiraIssues(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		authToken      string
		wantErr        bool
	}{
		{
			name: "successful request with issues",
			serverResponse: `{
				"issues": [
					{
						"key": "TEST-1",
						"fields": {
							"summary": "Test Issue",
							"status": {"name": "In Progress"}
						}
					}
				]
			}`,
			serverStatus: http.StatusOK,
			authToken:    "Bearer test-token",
			wantErr:      false,
		},
		{
			name:           "invalid JSON response",
			serverResponse: "Invalid JSON",
			serverStatus:   http.StatusOK,
			authToken:      "Bearer test-token",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader != tt.authToken {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
					return
				}
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := NewHTTPClient(server.URL, tt.authToken)

			issues, err := client.GetJiraIssues(server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPClient.GetJiraIssues() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(issues) == 0 {
				t.Error("HTTPClient.GetJiraIssues() returned empty issues slice for successful request")
			}
		})
	}
}
