// Package github provides GitHub API operations.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/go-github/v68/github"
)

// PRData contains pull request information.
type PRData struct {
	Number  int
	HTMLURL string
	NodeID  string
}

// CreatePullRequest creates a new pull request.
func CreatePullRequest(ctx context.Context, apiURL, token, owner, repo, title, body, head, base string) (*PRData, error) {
	client := newClient(ctx, apiURL, token)

	pr, _, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &head,
		Base:  &base,
	})
	if err != nil {
		return nil, fmt.Errorf("create pull request: %w", err)
	}

	return &PRData{
		Number:  pr.GetNumber(),
		HTMLURL: pr.GetHTMLURL(),
		NodeID:  pr.GetNodeID(),
	}, nil
}

// AddLabels adds labels to a pull request.
func AddLabels(ctx context.Context, apiURL, token, owner, repo string, prNumber int, labels []string) error {
	client := newClient(ctx, apiURL, token)

	_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNumber, labels)
	if err != nil {
		return fmt.Errorf("add labels: %w", err)
	}
	return nil
}

// RequestReviewers requests reviewers for a pull request.
func RequestReviewers(ctx context.Context, apiURL, token, owner, repo string, prNumber int, reviewers []string) error {
	client := newClient(ctx, apiURL, token)

	_, _, err := client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
		Reviewers: reviewers,
	})
	if err != nil {
		return fmt.Errorf("request reviewers: %w", err)
	}
	return nil
}

// EnableAutoMerge enables auto-merge on a pull request via GraphQL.
func EnableAutoMerge(ctx context.Context, graphqlURL, token, prNodeID, mergeMethod string) error {
	query := `mutation EnableAutoMerge($pullRequestId: ID!, $mergeMethod: PullRequestMergeMethod!) {
		enablePullRequestAutoMerge(input: {
			pullRequestId: $pullRequestId,
			mergeMethod: $mergeMethod
		}) {
			pullRequest {
				autoMergeRequest {
					enabledAt
				}
			}
		}
	}`

	variables := map[string]string{
		"pullRequestId": prNodeID,
		"mergeMethod":   mergeMethod,
	}

	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graphql request failed with status %d", resp.StatusCode)
	}

	return nil
}

func newClient(ctx context.Context, apiURL, token string) *github.Client {
	client := github.NewClient(nil).WithAuthToken(token)
	if apiURL != "" && apiURL != "https://api.github.com" {
		client, _ = client.WithEnterpriseURLs(apiURL, apiURL)
	}
	return client
}
