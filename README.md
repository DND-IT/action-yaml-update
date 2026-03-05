# action-yaml-update

GitHub Action to automate updating YAML files with full format and comment preservation. Supports Helm values, Kustomize configs, and any YAML file.

Built in Go using [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) for round-trip YAML processing — comments, formatting, and quote styles are preserved.

## usage

### key mode (explicit paths)

```yaml
- name: update yaml values
  uses: DND-IT/action-yaml-update@v0
  with:
    files: |
      deploy/values.yaml
    keys: |
      webapp.image.tag
      webapp.replicas
    values: |
      v2.0.0
      5
```

### key mode with single value

When all keys should get the same value, use `value` (singular) instead of repeating:

```yaml
- name: update all tags
  uses: DND-IT/action-yaml-update@v0
  with:
    files: deploy/values.yaml
    keys: |
      api.image_tag
      api.metadata.labels.datadog.version
      frontend.image_tag
      frontend.metadata.labels.datadog.version
    value: ${{ github.event.release.tag_name }}
```

### marker mode (comment-based)

Tag values in your YAML with `# x-yaml-update` comments. The action finds and updates all marked values:

```yaml
# deploy/values.yaml
api:
  image_tag: v1.0.0 # x-yaml-update
  metadata:
    labels:
      datadog:
        version: v1.0.0 # x-yaml-update
frontend:
  image_tag: v1.0.0 # x-yaml-update
```

```yaml
- name: update marked values
  uses: DND-IT/action-yaml-update@v0
  with:
    files: deploy/values.yaml
    mode: marker
    value: ${{ github.event.release.tag_name }}
```

Use a custom marker with the `marker` input:

```yaml
- name: update with custom marker
  uses: DND-IT/action-yaml-update@v0
  with:
    files: deploy/values.yaml
    mode: marker
    marker: my-tracking-id
    value: ${{ github.event.release.tag_name }}
```

Markers also support an ID suffix (`# x-yaml-update:image-tag`) which still matches the base marker.

### image mode (search by image name)

```yaml
- name: update image tag
  uses: DND-IT/action-yaml-update@v0
  with:
    files: |
      deploy/values.yaml
      deploy/kustomization.yaml
    mode: image
    image_name: webapp
    image_tag: v2.0.0
```

### file discovery with `files_from`

Instead of listing every file, point at a directory and let the action discover YAML files:

```yaml
- name: update all env values
  uses: DND-IT/action-yaml-update@v0
  with:
    files_from: deploy/charts/myapp/envs
    files_filter: values.yaml
    mode: image
    image_name: myapp
    image_tag: ${{ github.sha }}
```

`files_from` recursively searches for `.yml` and `.yaml` files. Use `files_filter` to narrow by filename (e.g. `values.yaml`). Both `files` and `files_from` can be combined — results are merged and deduplicated.

### dry run

```yaml
- name: preview changes
  id: preview
  uses: DND-IT/action-yaml-update@v0
  with:
    files: deploy/values.yaml
    keys: app.version
    values: v2.0.0
    dry_run: 'true'

- name: show diff
  run: echo "${{ steps.preview.outputs.diff }}"
```

### full example with PR

```yaml
name: update deployment
on:
  workflow_dispatch:
    inputs:
      tag:
        description: 'Image tag to deploy'
        required: true

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: update image tag
        id: update
        uses: DND-IT/action-yaml-update@v0
        with:
          files: deploy/values.yaml
          mode: image
          image_name: webapp
          image_tag: ${{ github.event.inputs.tag }}
          create_pr: 'true'
          pr_title: 'chore: deploy webapp ${{ github.event.inputs.tag }}'
          pr_labels: deployment, automated
          auto_merge: 'true'
          merge_method: SQUASH

      - name: output PR url
        if: steps.update.outputs.pr_url
        run: echo "PR created: ${{ steps.update.outputs.pr_url }}"
```

## inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `files` | no | — | Newline-separated YAML file paths to update |
| `files_from` | no | — | Directory to recursively search for YAML files (`.yml`/`.yaml`) |
| `files_filter` | no | — | Filename filter for `files_from` discovery (e.g. `values.yaml`) |
| `mode` | no | `key` | Update mode: `key`, `image`, or `marker` |
| `keys` | no | — | Newline-separated dot-notation key paths (mode=key) |
| `values` | no | — | Newline-separated values corresponding to keys (mode=key) |
| `value` | no | — | Single value applied to all keys (mode=key) or all marker matches (mode=marker) |
| `marker` | no | `x-yaml-update` | Comment marker to match in YAML (mode=marker) |
| `image_name` | no | — | Image name suffix to search for (mode=image) |
| `image_tag` | no | — | New tag value (mode=image) |
| `create_pr` | no | `true` | Create PR vs. direct commit |
| `target_branch` | no | repo default | Base branch for PR or direct commit target |
| `pr_branch` | no | auto-generated | Branch name for PR |
| `pr_title` | no | `chore: update YAML values` | PR title |
| `pr_body` | no | auto-generated | PR body |
| `pr_labels` | no | — | Comma-separated labels |
| `pr_reviewers` | no | — | Comma-separated reviewer usernames |
| `commit_message` | no | `chore: update YAML values` | Commit message |
| `token` | no | `${{ github.token }}` | GitHub token for git push + API calls |
| `auto_merge` | no | `false` | Enable auto-merge on PR |
| `merge_method` | no | `SQUASH` | MERGE, SQUASH, or REBASE |
| `dry_run` | no | `false` | Preview changes without modifying anything |
| `git_user_name` | no | `github-actions[bot]` | Git committer name |
| `git_user_email` | no | `41898282+github-actions[bot]@...` | Git committer email |

> **Note:** At least one of `files` or `files_from` must be provided.

## outputs

| Output | Description |
|--------|-------------|
| `changed` | `true`/`false` whether files were modified |
| `changed_files` | Newline-separated list of modified files |
| `pr_number` | PR number (empty if no PR) |
| `pr_url` | PR URL (empty if no PR) |
| `commit_sha` | SHA of the created commit |
| `diff` | Summary of changes |

## key path syntax

Dot-notation paths to traverse nested YAML structures:

- `app.version` — nested key
- `webapp.image.tag` — deeply nested
- `images.0.newTag` — list index access

Values are type-coerced to match the existing value type (int, bool, float, string).

## image mode

In image mode, the action recursively searches the YAML tree for:

- **Helm-style**: `repository`/`tag` pairs where `repository` ends with the image name
- **Kustomize-style**: `name`/`newTag` pairs where `name` ends with the image name

All matching occurrences in the document are updated.

## development

```bash
# run tests
go test ./...

# run tests with coverage
go test -cover ./...

# lint
golangci-lint run

# build binary
go build -o yaml-update ./cmd/yaml-update

# build docker image locally
docker build -t action-yaml-update .
```

## license

MIT

## support

Maintained by **group:default/dai**
