/**
 * GitHub Actions output and logging utilities.
 */

import { appendFileSync } from "node:fs";

export function setOutput(name, value) {
  const outputFile = process.env.GITHUB_OUTPUT;
  if (outputFile) {
    if (value.includes("\n")) {
      const delimiter = `ghadelimiter_${Date.now()}`;
      appendFileSync(
        outputFile,
        `${name}<<${delimiter}\n${value}\n${delimiter}\n`,
      );
    } else {
      appendFileSync(outputFile, `${name}=${value}\n`);
    }
  } else {
    console.log(`::set-output name=${name}::${value}`);
  }
}

export function logInfo(message) {
  console.log(message);
}

export function logWarning(message) {
  console.log(`::warning::${message}`);
}

export function logError(message) {
  console.log(`::error::${message}`);
}

export function logGroup(title) {
  console.log(`::group::${title}`);
}

export function logEndgroup() {
  console.log("::endgroup::");
}
