/**
 * Parse and validate GitHub Action inputs from environment variables.
 */

function getEnv(name, defaultValue = "") {
  return (
    process.env[`INPUT_${name.toUpperCase().replace(/-/g, "_")}`] ??
    defaultValue
  );
}

function parseBoolean(value) {
  return ["true", "yes", "1"].includes(value.trim().toLowerCase());
}

function parseList(value, separator = "\n") {
  if (!value.trim()) return [];
  return value
    .split(separator)
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

export class InputError extends Error {
  constructor(message) {
    super(message);
    this.name = "InputError";
  }
}

export function parseInputs() {
  const files = parseList(getEnv("files"));
  if (files.length === 0) {
    throw new InputError("'files' input is required");
  }

  const mode = getEnv("mode", "key");
  if (!["key", "image"].includes(mode)) {
    throw new InputError(`Invalid mode '${mode}'. Must be 'key' or 'image'`);
  }

  let keys = [];
  let values = [];
  let imageName = "";
  let imageTag = "";

  if (mode === "key") {
    keys = parseList(getEnv("keys"));
    values = parseList(getEnv("values"));

    if (keys.length === 0) {
      throw new InputError("'keys' input is required for mode=key");
    }
    if (values.length === 0) {
      throw new InputError("'values' input is required for mode=key");
    }
    if (keys.length !== values.length) {
      throw new InputError(
        `Number of keys (${keys.length}) must match number of values (${values.length})`,
      );
    }
  } else {
    imageName = getEnv("image_name");
    imageTag = getEnv("image_tag");

    if (!imageName) {
      throw new InputError("'image_name' input is required for mode=image");
    }
    if (!imageTag) {
      throw new InputError("'image_tag' input is required for mode=image");
    }
  }

  return {
    files,
    mode,
    keys,
    values,
    imageName,
    imageTag,
    createPr: parseBoolean(getEnv("create_pr", "true")),
    targetBranch: getEnv("target_branch"),
    prBranch: getEnv("pr_branch"),
    prTitle: getEnv("pr_title", "chore: update YAML values"),
    prBody: getEnv("pr_body"),
    prLabels: parseList(getEnv("pr_labels"), ","),
    prReviewers: parseList(getEnv("pr_reviewers"), ","),
    commitMessage: getEnv("commit_message", "chore: update YAML values"),
    token: getEnv("token") || process.env.GITHUB_TOKEN || "",
    autoMerge: parseBoolean(getEnv("auto_merge", "false")),
    mergeMethod: getEnv("merge_method", "SQUASH"),
    dryRun: parseBoolean(getEnv("dry_run", "false")),
    gitUserName: getEnv("git_user_name", "github-actions[bot]"),
    gitUserEmail: getEnv(
      "git_user_email",
      "41898282+github-actions[bot]@users.noreply.github.com",
    ),
    githubRepository: process.env.GITHUB_REPOSITORY || "",
    githubServerUrl: process.env.GITHUB_SERVER_URL || "https://github.com",
    githubApiUrl: process.env.GITHUB_API_URL || "https://api.github.com",
    githubGraphqlUrl:
      process.env.GITHUB_GRAPHQL_URL || "https://api.github.com/graphql",
  };
}
