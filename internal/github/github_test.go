package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_DefaultAPI(t *testing.T) {
	client, err := newClient("", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_ExplicitPublicAPI(t *testing.T) {
	client, err := newClient("https://api.github.com", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_EnterpriseURL(t *testing.T) {
	client, err := newClient("https://github.example.com/api/v3", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_InvalidEnterpriseURL(t *testing.T) {
	_, err := newClient("://invalid", "test-token")
	if err == nil {
		t.Fatal("expected error for invalid enterprise URL")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestEnableAutoMerge_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "bearer test-token" {
			t.Errorf("unexpected auth header: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected content-type: %s", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := payload["query"]; !ok {
			t.Error("expected 'query' in payload")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"enablePullRequestAutoMerge": map[string]any{
					"pullRequest": map[string]any{
						"autoMergeRequest": map[string]any{
							"enabledAt": "2024-01-01T00:00:00Z",
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	err := EnableAutoMerge(context.Background(), srv.URL, "test-token", "PR_node123", "SQUASH")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnableAutoMerge_GraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"message": "Pull request is not mergeable"},
				{"message": "Another error"},
			},
		})
	}))
	defer srv.Close()

	err := EnableAutoMerge(context.Background(), srv.URL, "test-token", "PR_node123", "SQUASH")
	if err == nil {
		t.Fatal("expected error for GraphQL errors response")
	}
	if got := err.Error(); got != "graphql errors: Pull request is not mergeable; Another error" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestEnableAutoMerge_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := EnableAutoMerge(context.Background(), srv.URL, "test-token", "PR_node123", "SQUASH")
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	if got := err.Error(); got != "graphql request failed with status 500" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestEnableAutoMerge_InvalidURL(t *testing.T) {
	err := EnableAutoMerge(context.Background(), "://invalid", "test-token", "PR_node123", "SQUASH")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
