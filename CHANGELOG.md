# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Flaggy integration for GNU-style flags (both short and long forms)
- Auto-generated help output with `-h`/`--help`
- Shell completion support (bash, zsh, fish, PowerShell, Nushell)
- Comprehensive flag parsing tests
- Manual help handling with clean output

### Changed
- Migrated from stdlib `flag` package to `flaggy` v1.8.0
- Separated flag parsing into `flags.go`
- Updated documentation for new flag syntax

### Improved
- User experience with both `-v` and `--version` flag forms
- Flag clustering support
- Equals syntax for flags (`--timeout=30s`)
- Test coverage maintained at 88.5%
