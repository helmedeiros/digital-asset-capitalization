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

func TestHTTPClient_GetBoards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"values": [{"id": 1, "name": "Board 1", "type": "scrum"} ]}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "Bearer test-token")
	boards, err := client.GetBoards(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(boards) != 1 || boards[0].Name != "Board 1" {
		t.Errorf("unexpected boards: %+v", boards)
	}
}

func TestHTTPClient_GetSprints(t *testing.T) {
	t.Run("valid sprints", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"values": [
				{"id": 123, "name": "Sprint 1", "state": "active", "startDate": "2024-01-01", "endDate": "2024-01-15", "goal": "Goal 1"},
				{"id": "456", "name": "Sprint 2", "state": "closed", "startDate": "2024-02-01", "endDate": "2024-02-15", "goal": "Goal 2"}
			]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "Bearer test-token")
		sprints, err := client.GetSprints(server.URL)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(sprints) != 2 {
			t.Errorf("expected 2 sprints, got %d", len(sprints))
		}
		if sprints[0].ID != "123" || sprints[1].ID != "456" {
			t.Errorf("unexpected sprint IDs: %+v", sprints)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"values": [ {`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "Bearer test-token")
		_, err := client.GetSprints(server.URL)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}

func TestSprint_UnmarshalJSON(t *testing.T) {
	jsons := []struct {
		input    string
		expected string
	}{
		{`{"id": 123, "name": "SprintNum"}`, "123"},
		{`{"id": "abc", "name": "SprintStr"}`, "abc"},
	}
	for _, tc := range jsons {
		var s Sprint
		err := s.UnmarshalJSON([]byte(tc.input))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if s.ID != tc.expected {
			t.Errorf("expected ID %q, got %q", tc.expected, s.ID)
		}
	}
}
