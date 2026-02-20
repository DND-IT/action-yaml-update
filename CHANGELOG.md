# Changelog

## [0.3.2](https://github.com/DND-IT/action-yaml-update/compare/v0.3.1...v0.3.2) (2026-02-20)


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
