# Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.0] - 2026-02-20

### (dfe81e5) lsi: add Windows build support

### (cad4974) doc: update CHANGELOG format, fix targeted release.sh builds

- Extract commit messages for CHANGELOG entries
- Add 'changes' argument to release.sh for dry-run preview
- redirect all logging to stderr
- cast stat.Dev to uint64 for darwin

### (c251f29) cli: add shell completion subcommand with auto-detection

- Add 'completion' subcommand supporting bash, zsh, fish, and PowerShell

### (5f3e50b) all: GNU-style flags with flaggy and automated release infrastructure

- Migrate from stdlib flag package to flaggy for GNU conventions
- Add automated release.sh script for building multi-platform binaries

### (e9ca50d) all: major refactor for simplified arch, greater test cov


## [Unreleased]
