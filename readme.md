# action-yaml-update

GitHub Action to automate updating YAML files with full format and comment preservation. Supports Helm values, Kustomize configs, and any YAML file.

Uses [`ruamel.yaml`](https://yaml.readthedocs.io/) for round-trip YAML processing — comments, formatting, and quote styles are preserved.

## usage

### key mode (explicit paths)

```yaml
- name: update yaml values
  uses: DND-IT/action-yaml-update@v1
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

### image mode (search by image name)

```yaml
- name: update image tag
  uses: DND-IT/action-yaml-update@v1
  with:
    files: |
      deploy/values.yaml
      deploy/kustomization.yaml
    mode: image
    image_name: webapp
    image_tag: v2.0.0
```

### dry run

```yaml
- name: preview changes
  id: preview
  uses: DND-IT/action-yaml-update@v1
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
        uses: DND-IT/action-yaml-update@v1
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
| `files` | yes | — | Newline-separated YAML file paths to update |
| `mode` | no | `key` | Update mode: `key` (explicit paths) or `image` (search by image name) |
| `keys` | no | — | Newline-separated dot-notation key paths (mode=key) |
| `values` | no | — | Newline-separated values corresponding to keys (mode=key) |
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

## image mode

In image mode, the action recursively searches the YAML tree for:

- **Helm-style**: `repository`/`tag` pairs where `repository` ends with the image name
- **Kustomize-style**: `name`/`newTag` pairs where `name` ends with the image name

## development

```bash
# install dependencies
uv sync --dev

# run tests
uv run pytest --cov --cov-report=term-missing

# lint
uv run ruff check src/ tests/
uv run ruff format --check src/ tests/

# build docker image locally
docker build -t action-yaml-update .
```

## license

MIT

## support

Maintained by **group:default/dai**
