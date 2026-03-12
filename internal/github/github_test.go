package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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

func TestRetryTransport_NoRetryOnSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp.Body.Close()

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}

func TestRetryTransport_RetriesOn429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
}

func TestRetryTransport_Returns429AfterMaxRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("expected %d calls, got %d", maxRetries, got)
	}
}

func TestRetryTransport_PreservesBodyOnRetry(t *testing.T) {
	var calls atomic.Int32
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		lastBody = string(body)
		if n < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	resp, err := client.Post(srv.URL, "text/plain", io.NopCloser(strings.NewReader("test-body")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp.Body.Close()

	if lastBody != "test-body" {
		t.Fatalf("expected body 'test-body' on retry, got %q", lastBody)
	}
}

func TestRetryTransport_FailsImmediatelyWhenResetTooFar(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		// Reset in 1 hour — way beyond retry budget
		resetUnix := time.Now().Add(1 * time.Hour).Unix()
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	_, err := client.Get(srv.URL)
	if err == nil {
		t.Fatal("expected error when reset time exceeds retry budget")
	}
	if !strings.Contains(err.Error(), "exceeds retry budget") {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fail after first 429, not retry all 3 times
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 call (fail immediately), got %d", got)
	}
}

func TestRetryTransport_LogsRateLimitInfo(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n < 2 {
			// Reset soon — within retry budget
			resetUnix := time.Now().Add(1 * time.Second).Unix()
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRetryTransport_RespectsContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)

	// Cancel immediately so the backoff wait is interrupted
	cancel()

	_, err := client.Do(req)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
