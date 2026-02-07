/**
 * Git operations via subprocess.
 */

import { execSync } from "node:child_process";

function run(command) {
  return execSync(command, {
    encoding: "utf8",
    stdio: ["pipe", "pipe", "pipe"],
  }).trim();
}

export function configure(userName, userEmail, token, repository, serverUrl) {
  run(`git config user.name "${userName}"`);
  run(`git config user.email "${userEmail}"`);

  // Mark workspace as safe
  const workspace = process.cwd();
  run(`git config --global --add safe.directory "${workspace}"`);

  // Set up authenticated remote if token provided
  if (token && repository) {
    const host = serverUrl.replace(/^https?:\/\//, "");
    const remoteUrl = `https://x-access-token:${token}@${host}/${repository}.git`;
    run(`git remote set-url origin "${remoteUrl}"`);
  }
}

export function getDefaultBranch() {
  try {
    // Try to get from remote HEAD
    const ref = run("git symbolic-ref refs/remotes/origin/HEAD");
    return ref.replace("refs/remotes/origin/", "");
  } catch {
    // Fallback to main
    return "main";
  }
}

export function createBranch(name, base) {
  run(`git fetch origin ${base}`);
  run(`git checkout -b ${name} origin/${base}`);
}

export function commitAndPush(files, message, branch) {
  for (const file of files) {
    run(`git add "${file}"`);
  }
  run(`git commit -m "${message.replace(/"/g, '\\"')}"`);
  run(`git push -u origin ${branch}`);
  return run("git rev-parse HEAD");
}
