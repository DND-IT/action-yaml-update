/**
 * Core YAML update logic using the yaml package for format preservation.
 */

import { parseDocument, isMap, isSeq, Scalar } from "yaml";

/**
 * Coerce a string value to match the type of the existing value.
 */
function coerceValue(newValue, existingValue) {
  if (existingValue === null || existingValue === undefined) {
    return newValue;
  }
  if (typeof existingValue === "boolean") {
    return ["true", "yes", "1"].includes(newValue.trim().toLowerCase());
  }
  if (typeof existingValue === "number") {
    if (Number.isInteger(existingValue)) {
      const parsed = parseInt(newValue, 10);
      return isNaN(parsed) ? newValue : parsed;
    }
    const parsed = parseFloat(newValue);
    return isNaN(parsed) ? newValue : parsed;
  }
  return newValue;
}

/**
 * Detect indentation from YAML content by finding nested keys.
 */
function detectIndent(content) {
  const lines = content.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const stripped = line.trimStart();
    if (!stripped || stripped.startsWith("#") || stripped.startsWith("-"))
      continue;
    if (stripped.includes(":")) {
      const indent = line.length - stripped.length;
      if (indent > 0) return indent;
    }
  }
  return 2; // default
}

/**
 * Load YAML content and return a Document that preserves formatting.
 * Returns { doc, indent } where indent is the detected indentation.
 */
export function loadYaml(content) {
  const indent = detectIndent(content);
  const doc = parseDocument(content);
  doc._detectedIndent = indent;
  return doc;
}

/**
 * Dump a YAML Document back to string, preserving formatting.
 */
export function dumpYaml(doc) {
  const indent = doc._detectedIndent || 2;
  return doc.toString({ indent });
}

/**
 * Update values at dot-notation key paths.
 * Returns list of changes made.
 */
export function updateKeys(doc, keys, values) {
  const changes = [];

  for (let i = 0; i < keys.length; i++) {
    const keyPath = keys[i];
    const newValue = values[i];

    const pathParts = keyPath.split(".").map((p) => {
      const num = parseInt(p, 10);
      return isNaN(num) ? p : num;
    });

    const oldValue = doc.getIn(pathParts);
    if (oldValue === undefined) {
      throw new Error(`Key '${keyPath}' not found`);
    }

    const coerced = coerceValue(newValue, oldValue);
    if (oldValue !== coerced) {
      doc.setIn(pathParts, coerced);
      changes.push({
        key: keyPath,
        old: oldValue,
        new: coerced,
      });
    }
  }

  return changes;
}

/**
 * Recursively walk a YAML tree looking for image repository/tag pairs.
 */
function walkImageTags(node, imageName, newTag, changes, path = "") {
  if (isMap(node)) {
    const items = node.items;

    // Check for repository/tag pattern (Helm-style)
    const repoItem = items.find((item) => item.key?.value === "repository");
    const tagItem = items.find((item) => item.key?.value === "tag");

    if (repoItem && tagItem) {
      const repoVal = String(repoItem.value?.value ?? repoItem.value);
      if (repoVal.endsWith(`/${imageName}`) || repoVal === imageName) {
        const oldTag = tagItem.value?.value ?? tagItem.value;
        const coerced = coerceValue(newTag, oldTag);
        if (oldTag !== coerced) {
          tagItem.value = new Scalar(coerced);
          const tagPath = path ? `${path}.tag` : "tag";
          changes.push({ key: tagPath, old: oldTag, new: coerced });
        }
      }
    }

    // Check for name/newTag pattern (Kustomize-style)
    const nameItem = items.find((item) => item.key?.value === "name");
    const newTagItem = items.find((item) => item.key?.value === "newTag");

    if (nameItem && newTagItem) {
      const nameVal = String(nameItem.value?.value ?? nameItem.value);
      if (nameVal.endsWith(`/${imageName}`) || nameVal === imageName) {
        const oldTag = newTagItem.value?.value ?? newTagItem.value;
        const coerced = coerceValue(newTag, oldTag);
        if (oldTag !== coerced) {
          newTagItem.value = new Scalar(coerced);
          const tagPath = path ? `${path}.newTag` : "newTag";
          changes.push({ key: tagPath, old: oldTag, new: coerced });
        }
      }
    }

    // Recurse into children
    for (const item of items) {
      const key = item.key?.value ?? item.key;
      const childPath = path ? `${path}.${key}` : String(key);
      walkImageTags(item.value, imageName, newTag, changes, childPath);
    }
  } else if (isSeq(node)) {
    node.items.forEach((item, i) => {
      const childPath = `${path}.${i}`;
      walkImageTags(item, imageName, newTag, changes, childPath);
    });
  }
}

/**
 * Search for image references matching imageName and update their tags.
 */
export function updateImageTags(doc, imageName, newTag) {
  const changes = [];
  walkImageTags(doc.contents, imageName, newTag, changes);
  return changes;
}

/**
 * Generate a human-readable diff between original and updated YAML.
 */
export function diffYaml(filePath, originalContent, doc) {
  const newContent = doc.toString();

  if (originalContent === newContent) {
    return "";
  }

  const origLines = originalContent.split("\n");
  const newLines = newContent.split("\n");

  const lines = [`--- ${filePath}`, `+++ ${filePath}`];

  // Simple diff - show changed lines
  const maxLen = Math.max(origLines.length, newLines.length);
  let inHunk = false;
  let hunkStart = -1;
  let hunkOrig = [];
  let hunkNew = [];

  const flushHunk = () => {
    if (hunkOrig.length > 0 || hunkNew.length > 0) {
      lines.push(
        `@@ -${hunkStart + 1},${hunkOrig.length} +${hunkStart + 1},${hunkNew.length} @@`,
      );
      for (const line of hunkOrig) {
        lines.push(`-${line}`);
      }
      for (const line of hunkNew) {
        lines.push(`+${line}`);
      }
    }
    hunkOrig = [];
    hunkNew = [];
    inHunk = false;
  };

  for (let i = 0; i < maxLen; i++) {
    const orig = origLines[i] ?? "";
    const curr = newLines[i] ?? "";

    if (orig !== curr) {
      if (!inHunk) {
        inHunk = true;
        hunkStart = i;
      }
      if (origLines[i] !== undefined) hunkOrig.push(orig);
      if (newLines[i] !== undefined) hunkNew.push(curr);
    } else {
      flushHunk();
    }
  }
  flushHunk();

  return lines.join("\n");
}
