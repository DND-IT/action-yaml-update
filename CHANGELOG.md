# Changelog

## [0.3.7](https://github.com/DND-IT/action-yaml-update/compare/v0.3.6...v0.3.7) (2026-03-12)


### Features

* add 429 retry with exponential backoff and PR coverage reporting ([#18](https://github.com/DND-IT/action-yaml-update/issues/18)) ([08b0a57](https://github.com/DND-IT/action-yaml-update/commit/08b0a577378296104faf10773e20cdc82f42e148))

## [0.3.6](https://github.com/DND-IT/action-yaml-update/compare/v0.3.5...v0.3.6) (2026-03-10)


### Bug Fixes

* update existing PR instead of failing when branch already has an open PR ([09972de](https://github.com/DND-IT/action-yaml-update/commit/09972de6280d687c720066e96a1d00e6c6459fd3))

## [0.3.5](https://github.com/DND-IT/action-yaml-update/compare/v0.3.4...v0.3.5) (2026-03-10)


### Bug Fixes

* fetch remote branch before force-with-lease push to prevent stale ref rejection ([1eda453](https://github.com/DND-IT/action-yaml-update/commit/1eda4532dc1c22bc63fbc33bdccc61eaaca6f645))

## [0.3.4](https://github.com/DND-IT/action-yaml-update/compare/v0.3.3...v0.3.4) (2026-03-05)


### Features

* add singular value input and marker mode for comment-based updates ([#12](https://github.com/DND-IT/action-yaml-update/issues/12)) ([7586c6c](https://github.com/DND-IT/action-yaml-update/commit/7586c6cfbfddda97cde85da6ef57d55bf7b2a097))

## [0.3.3](https://github.com/DND-IT/action-yaml-update/compare/v0.3.2...v0.3.3) (2026-03-04)


### Features

* force push deploy branches to always reflect HEAD + changes ([a2d8a42](https://github.com/DND-IT/action-yaml-update/commit/a2d8a42f7b1028a470fe37230a4c4d8f86dafd86))

## [0.3.2](https://github.com/DND-IT/action-yaml-update/compare/v0.3.1...v0.3.2) (2026-02-23)


### Features

* add files_from directory discovery and fix blank line removal ([7092fc1](https://github.com/DND-IT/action-yaml-update/commit/7092fc1281a777228fcfdd5cc326656cd16ce9bf))


### Bug Fixes

* retry push with rebase when remote has new commits ([41b04e0](https://github.com/DND-IT/action-yaml-update/commit/41b04e09a9f6be4db173fc0aba4d63c83f5e67c2))

## [0.3.1](https://github.com/DND-IT/action-yaml-update/compare/v0.3.0...v0.3.1) (2026-02-20)


### Bug Fixes

* mark workspace as safe directory before local git config ([fdca406](https://github.com/DND-IT/action-yaml-update/commit/fdca4066a723052a22e55b436ce63915931ae92f))

## [0.3.0](https://github.com/DND-IT/action-yaml-update/compare/v0.2.0...v0.3.0) (2026-02-13)


### ⚠ BREAKING CHANGES

* action.yml now references a versioned container image instead of :main tag. Container images are tagged with semver versions (e.g. 0.3.0, 0.3, 0) on each release.

### Features

* use semver container tags and pin action to release version ([c62568b](https://github.com/DND-IT/action-yaml-update/commit/c62568bc07e60996ceb49380eaca88761da5668c))

## [0.2.0](https://github.com/DND-IT/action-yaml-update/compare/v0.1.1...v0.2.0) (2026-02-12)


### ⚠ BREAKING CHANGES

* Internal implementation changed, but action.yml interface remains the same - no changes needed for users.

### Features

* rewrite action in Go ([#6](https://github.com/DND-IT/action-yaml-update/issues/6)) ([44e2663](https://github.com/DND-IT/action-yaml-update/commit/44e2663ae69c7c672cded0811c54d8765edfeb07))

## [0.1.1](https://github.com/DND-IT/action-yaml-update/compare/v0.1.0...v0.1.1) (2026-02-06)


### Bug Fixes

* ref docker registry ([bdbf940](https://github.com/DND-IT/action-yaml-update/commit/bdbf940a916fab8448d7851dfdbc1497e980b080))

## 0.1.0 (2026-02-06)


### Features

* initial release ([7569dfa](https://github.com/DND-IT/action-yaml-update/commit/7569dfa0d76d2ec5bc3d11d81037f35171803cd5))
