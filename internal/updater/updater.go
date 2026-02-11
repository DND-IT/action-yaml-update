// Package updater provides format-preserving YAML update operations.
package updater

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Change represents a single value change made to the YAML.
type Change struct {
	Key string
	Old any
	New any
}

// Document wraps a yaml.Node with detected indentation.
type Document struct {
	Root   *yaml.Node
	Indent int
}

// LoadYAML parses YAML content into a Document for format-preserving editing.
func LoadYAML(content []byte) (*Document, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	indent := detectIndent(content)

	return &Document{
		Root:   &node,
		Indent: indent,
	}, nil
}

// DumpYAML serializes a Document back to bytes, preserving formatting.
func DumpYAML(doc *Document) ([]byte, error) {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(doc.Indent)

	if err := enc.Encode(doc.Root); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func detectIndent(content []byte) int {
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		indent := len(line) - len(stripped)
		if indent > 0 {
			return indent
		}
	}
	return 2
}

// UpdateKeys updates values at dot-notation key paths.
func UpdateKeys(doc *Document, keys, values []string) ([]Change, error) {
	var changes []Change

	// Get the document content node
	content := doc.Root
	if doc.Root.Kind == yaml.DocumentNode && len(doc.Root.Content) > 0 {
		content = doc.Root.Content[0]
	}

	for i, keyPath := range keys {
		newValue := values[i]

		node, err := resolveKeyPath(content, keyPath)
		if err != nil {
			return nil, err
		}

		oldValue := nodeValue(node)
		coerced := coerceValue(newValue, node)

		if node.Value != coerced {
			changes = append(changes, Change{
				Key: keyPath,
				Old: oldValue,
				New: parseValue(coerced),
			})
			node.Value = coerced
			// Preserve the original tag/style when possible
			if node.Tag == "!!int" || node.Tag == "!!float" || node.Tag == "!!bool" {
				// Keep the tag if coerced value matches the type
				if _, err := strconv.Atoi(coerced); err == nil && node.Tag == "!!int" {
					// Keep !!int
				} else if _, err := strconv.ParseFloat(coerced, 64); err == nil && node.Tag == "!!float" {
					// Keep !!float
				} else if coerced == "true" || coerced == "false" {
					// Keep !!bool
				} else {
					node.Tag = "!!str"
				}
			}
		}
	}

	return changes, nil
}

// UpdateImageTags searches for image references and updates their tags.
func UpdateImageTags(doc *Document, imageName, newTag string) []Change {
	var changes []Change

	content := doc.Root
	if doc.Root.Kind == yaml.DocumentNode && len(doc.Root.Content) > 0 {
		content = doc.Root.Content[0]
	}

	walkImageTags(content, imageName, newTag, &changes, "")
	return changes
}

func resolveKeyPath(node *yaml.Node, keyPath string) (*yaml.Node, error) {
	parts := strings.Split(keyPath, ".")
	current := node

	for i, part := range parts {
		if current.Kind == yaml.MappingNode {
			found := false
			for j := 0; j < len(current.Content); j += 2 {
				if current.Content[j].Value == part {
					current = current.Content[j+1]
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("key '%s' not found in path '%s'", part, keyPath)
			}
		} else if current.Kind == yaml.SequenceNode {
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("expected integer index for list, got '%s' in path '%s'", part, keyPath)
			}
			if idx < 0 || idx >= len(current.Content) {
				return nil, fmt.Errorf("index %d out of range in path '%s'", idx, keyPath)
			}
			current = current.Content[idx]
		} else if i < len(parts)-1 {
			return nil, fmt.Errorf("cannot traverse into scalar at '%s' in path '%s'", part, keyPath)
		}
	}

	return current, nil
}

func walkImageTags(node *yaml.Node, imageName, newTag string, changes *[]Change, path string) {
	switch node.Kind {
	case yaml.MappingNode:
		// Build a map of key -> value node for easy lookup
		keyMap := make(map[string]*yaml.Node)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			keyMap[key] = node.Content[i+1]
		}

		// Check for repository/tag pattern (Helm-style)
		if repoNode, ok := keyMap["repository"]; ok {
			if tagNode, ok := keyMap["tag"]; ok {
				repoVal := repoNode.Value
				if strings.HasSuffix(repoVal, "/"+imageName) || repoVal == imageName {
					oldTag := nodeValue(tagNode)
					coerced := coerceValue(newTag, tagNode)
					if tagNode.Value != coerced {
						tagPath := "tag"
						if path != "" {
							tagPath = path + ".tag"
						}
						*changes = append(*changes, Change{
							Key: tagPath,
							Old: oldTag,
							New: parseValue(coerced),
						})
						tagNode.Value = coerced
					}
				}
			}
		}

		// Check for name/newTag pattern (Kustomize-style)
		if nameNode, ok := keyMap["name"]; ok {
			if newTagNode, ok := keyMap["newTag"]; ok {
				nameVal := nameNode.Value
				if strings.HasSuffix(nameVal, "/"+imageName) || nameVal == imageName {
					oldTag := nodeValue(newTagNode)
					coerced := coerceValue(newTag, newTagNode)
					if newTagNode.Value != coerced {
						tagPath := "newTag"
						if path != "" {
							tagPath = path + ".newTag"
						}
						*changes = append(*changes, Change{
							Key: tagPath,
							Old: oldTag,
							New: parseValue(coerced),
						})
						newTagNode.Value = coerced
					}
				}
			}
		}

		// Recurse into children
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			walkImageTags(node.Content[i+1], imageName, newTag, changes, childPath)
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			childPath := fmt.Sprintf("%s.%d", path, i)
			if path == "" {
				childPath = strconv.Itoa(i)
			}
			walkImageTags(child, imageName, newTag, changes, childPath)
		}
	}
}

func nodeValue(node *yaml.Node) any {
	switch node.Tag {
	case "!!int":
		if v, err := strconv.Atoi(node.Value); err == nil {
			return v
		}
	case "!!float":
		if v, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return v
		}
	case "!!bool":
		return node.Value == "true"
	}
	return node.Value
}

func parseValue(s string) any {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	return s
}

func coerceValue(newValue string, node *yaml.Node) string {
	switch node.Tag {
	case "!!int":
		if _, err := strconv.Atoi(newValue); err == nil {
			return newValue
		}
	case "!!float":
		if _, err := strconv.ParseFloat(newValue, 64); err == nil {
			return newValue
		}
	case "!!bool":
		lower := strings.ToLower(strings.TrimSpace(newValue))
		if lower == "true" || lower == "yes" || lower == "1" {
			return "true"
		}
		return "false"
	}
	return newValue
}

// Diff generates a simple unified diff between original and new content.
func Diff(filename string, original, updated []byte) string {
	if string(original) == string(updated) {
		return ""
	}

	origLines := strings.Split(string(original), "\n")
	newLines := strings.Split(string(updated), "\n")

	var lines []string
	lines = append(lines, fmt.Sprintf("--- %s", filename))
	lines = append(lines, fmt.Sprintf("+++ %s", filename))

	maxLen := len(origLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	var hunkOrig, hunkNew []string
	hunkStart := -1

	flushHunk := func() {
		if len(hunkOrig) > 0 || len(hunkNew) > 0 {
			lines = append(lines, fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunkStart+1, len(hunkOrig), hunkStart+1, len(hunkNew)))
			for _, line := range hunkOrig {
				lines = append(lines, "-"+line)
			}
			for _, line := range hunkNew {
				lines = append(lines, "+"+line)
			}
		}
		hunkOrig = nil
		hunkNew = nil
		hunkStart = -1
	}

	for i := 0; i < maxLen; i++ {
		var orig, curr string
		if i < len(origLines) {
			orig = origLines[i]
		}
		if i < len(newLines) {
			curr = newLines[i]
		}

		if orig != curr {
			if hunkStart == -1 {
				hunkStart = i
			}
			if i < len(origLines) {
				hunkOrig = append(hunkOrig, orig)
			}
			if i < len(newLines) {
				hunkNew = append(hunkNew, curr)
			}
		} else {
			flushHunk()
		}
	}
	flushHunk()

	return strings.Join(lines, "\n")
}
