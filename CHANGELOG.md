# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Flaggy integration for GNU-style flags (both short and long forms)
- Auto-generated help output with `-h`/`--help`
- Shell completion subcommand with support for bash, zsh, fish, and PowerShell
- Automatic shell detection from environment variables ($SHELL, etc.)
- Comprehensive flag parsing tests with 30+ test cases
- Comprehensive completion tests covering all shells and auto-detection
- Manual help handling with clean output
- Release automation with `release.sh` script
- Multi-platform binary packaging (9 OS/arch combinations)
- Comprehensive release documentation in `RELEASING.md`
- Project guidelines for AI agents in `AGENTS.md`

### Changed
- Migrated from stdlib `flag` package to `flaggy` v1.8.0
- Separated flag parsing into `flags.go`
- Separated completion logic into `completion.go`
- Updated documentation for new flag syntax and completion feature
- Enhanced help message to include completion subcommand

### Improved
- User experience with both `-v` and `--version` flag forms
- Flag clustering support (e.g., `-lugs`)
- Equals syntax for flags (`--timeout=30s`)
- Test coverage maintained at 88.5%
- Shell completion improves CLI workflow efficiency
