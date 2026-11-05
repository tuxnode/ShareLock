# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][Keep a Changelog] and this project adheres to [Semantic Versioning][Semantic Versioning].

## [Unreleased]

### Added
- Server configuration via `.env` file and `SHARELOCK_*` environment variables (`-env` flag to override path)
- `.env.example` documented configuration template
- Docker image for the server (multi-stage, non-root)
- CI job to build and push the Docker image to GHCR on release tags
- File listing: `list` command and `ListFiles` method
- Session persistence: credentials cached to disk after `init`/`login`
- Logout command to clear cached session
- Debugging workflow section to AGENTS.md

### Fixed
- `ListFiles` no longer panics when user is not logged in
- `FileSender` typo corrected in netstream module
- `UserlibKeyStore` cross-test key contamination: each instance now maintains a per-instance cache to prevent stale keys from previous tests
- Improved error messages with username/filename context across all services

### Changed
- Migration from CLAUDE.md to AGENTS.md with more compact, repo-specific guidance

### Removed
- Legacy duplicate struct definitions from encryption.go

## [v0.2.0] - 2021-03-29
### Changed
- Updated [userlib][userlib] dependency to `v0.2.0`.

---

## [Released]

## [v0.1.0] - 2021-02-21
CHANGELOG did not exist in this release.

---

<!-- Links -->
[Keep a Changelog]: https://keepachangelog.com/
[Semantic Versioning]: https://semver.org/
[userlib]: https://github.com/cs161-staff/project2-userlib/blob/master/CHANGELOG.md

<!-- Versions -->
[Unreleased]: https://github.com/cs161-staff/project2-starter-code/compare/v0.2.0...HEAD
[Released]: https://github.com/cs161-staff/project2-starter-code/releases
[v0.2.0]: https://github.com/cs161-staff/project2-starter-code/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/cs161-staff/project2-starter-code/releases/v0.1.0
