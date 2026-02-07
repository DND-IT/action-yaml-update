/**
 * GitHub API operations using Octokit.
 */

import { Octokit } from "@octokit/rest";
import { graphql } from "@octokit/graphql";

export async function createPullRequest(
  apiUrl,
  token,
  owner,
  repo,
  title,
  body,
  head,
  base,
) {
  const octokit = new Octokit({ auth: token, baseUrl: apiUrl });

  const response = await octokit.pulls.create({
    owner,
    repo,
    title,
    body,
    head,
    base,
  });

  return response.data;
}

export async function addLabels(
  apiUrl,
  token,
  owner,
  repo,
  issueNumber,
  labels,
) {
  const octokit = new Octokit({ auth: token, baseUrl: apiUrl });

  await octokit.issues.addLabels({
    owner,
    repo,
    issue_number: issueNumber,
    labels,
  });
}

export async function requestReviewers(
  apiUrl,
  token,
  owner,
  repo,
  prNumber,
  reviewers,
) {
  const octokit = new Octokit({ auth: token, baseUrl: apiUrl });

  await octokit.pulls.requestReviewers({
    owner,
    repo,
    pull_number: prNumber,
    reviewers,
  });
}

export async function enableAutoMerge(
  graphqlUrl,
  token,
  prNodeId,
  mergeMethod,
) {
  const graphqlWithAuth = graphql.defaults({
    baseUrl: graphqlUrl.replace("/graphql", ""),
    headers: {
      authorization: `token ${token}`,
    },
  });

  const mutation = `
    mutation EnableAutoMerge($pullRequestId: ID!, $mergeMethod: PullRequestMergeMethod!) {
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
    }
  `;

  await graphqlWithAuth(mutation, {
    pullRequestId: prNodeId,
    mergeMethod: mergeMethod.toUpperCase(),
  });
}
