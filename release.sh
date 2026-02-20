#!/usr/bin/env bash
#
# release.sh - Create and publish a new release of lsi
#
# Usage:
#   ./release.sh [major|minor|patch|prerelease]
#
# Default: minor
#
# Requirements:
#   - git
#   - gh (GitHub CLI)
#   - svu (semantic version utility)
#   - go (1.21+)
#
# This script will:
#   1. Verify workspace is clean (no uncommitted changes)
#   2. Compute next version using svu
#   3. Verify version/tag doesn't already exist
#   4. Update CHANGELOG.md with commits since last release
#   5. Build distribution packages for all platforms
#   6. Create GitHub release with packages attached

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="lsi"
DIST_DIR="dist"
PLATFORMS=(
	"linux/amd64"
	"linux/arm64"
	"linux/386"
	"darwin/amd64"
	"darwin/arm64"
	"windows/amd64"
	"windows/386"
	"freebsd/amd64"
	"openbsd/amd64"
)

# Helper functions
log_info() {
	echo -e "${BLUE}==>${NC} $*"
}

log_success() {
	echo -e "${GREEN}✓${NC} $*"
}

log_warn() {
	echo -e "${YELLOW}⚠${NC} $*"
}

log_error() {
	echo -e "${RED}✗${NC} $*" >&2
}

die() {
	log_error "$@"
	exit 1
}

# Check required commands
check_requirements() {
	local missing=()

	for cmd in git gh svu go; do
		if ! command -v "$cmd" &>/dev/null; then
			missing+=("$cmd")
		fi
	done

	if [ ${#missing[@]} -gt 0 ]; then
		die "Missing required commands: ${missing[*]}\nInstall them before running this script."
	fi

	log_success "All required commands found"
}

# Verify workspace is clean
check_workspace() {
	log_info "Checking workspace status..."

	if ! git diff-index --quiet HEAD --; then
		die "Workspace has uncommitted changes. Commit or stash them first."
	fi

	if [ -n "$(git ls-files --others --exclude-standard)" ]; then
		die "Workspace has untracked files. Add or ignore them first."
	fi

	log_success "Workspace is clean"
}

# Compute next version using svu
compute_version() {
	local bump_type="${1:-minor}"

	log_info "Computing next version (bump: $bump_type)..."

	# Validate bump type
	case "$bump_type" in
	major | minor | patch | prerelease) ;;
	*)
		die "Invalid bump type: $bump_type. Must be one of: major, minor, patch, prerelease"
		;;
	esac

	# Get current version
	local current_version
	current_version=$(svu current 2>/dev/null || echo "v0.0.0")
	log_info "Current version: $current_version"

	# Compute next version
	local next_version
	next_version=$(svu "$bump_type")

	if [ -z "$next_version" ]; then
		die "Failed to compute next version"
	fi

	# Ensure version starts with 'v'
	if [[ ! "$next_version" =~ ^v ]]; then
		next_version="v$next_version"
	fi

	log_success "Next version: $next_version"
	echo "$next_version"
}

# Check if tag already exists
check_tag_exists() {
	local version="$1"

	log_info "Checking if tag $version already exists..."

	if git rev-parse "$version" >/dev/null 2>&1; then
		die "Tag $version already exists"
	fi

	log_success "Tag $version is unique"
}

# Get previous version tag
get_previous_version() {
	git describe --tags --abbrev=0 2>/dev/null || echo ""
}

# Update CHANGELOG.md with commits since last release
update_changelog() {
	local version="$1"
	local prev_version
	prev_version=$(get_previous_version)

	log_info "Updating CHANGELOG.md..."

	# Create CHANGELOG.md if it doesn't exist
	if [ ! -f CHANGELOG.md ]; then
		log_info "Creating CHANGELOG.md..."
		cat >CHANGELOG.md <<EOF
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

EOF
	fi

	# Get commit messages since last release
	local commits
	if [ -n "$prev_version" ]; then
		log_info "Getting commits since $prev_version..."
		commits=$(git log "${prev_version}..HEAD" --pretty=format:"- %s" --no-merges)
	else
		log_info "No previous version found, getting all commits..."
		commits=$(git log --pretty=format:"- %s" --no-merges)
	fi

	if [ -z "$commits" ]; then
		log_warn "No commits found since last release"
		commits="- Initial release"
	fi

	# Create temporary file with new entry
	local temp_file
	temp_file=$(mktemp)

	# Extract everything before the first version entry (header)
	local header_lines
	header_lines=$(grep -n "^## \[" CHANGELOG.md | head -1 | cut -d: -f1 || echo "")

	if [ -n "$header_lines" ]; then
		# Has existing versions
		head -n $((header_lines - 1)) CHANGELOG.md >"$temp_file"
	else
		# No existing versions, copy whole file
		cat CHANGELOG.md >"$temp_file"
	fi

	# Add new version entry
	{
		echo "## [$version] - $(date +%Y-%m-%d)"
		echo ""
		echo "$commits"
		echo ""
	} >>"$temp_file"

	# Append rest of the changelog
	if [ -n "$header_lines" ]; then
		tail -n +$header_lines CHANGELOG.md >>"$temp_file"
	fi

	# Replace original
	mv "$temp_file" CHANGELOG.md

	log_success "CHANGELOG.md updated"
}

# Build binary for specific platform
build_binary() {
	local goos="$1"
	local goarch="$2"
	local version="$3"

	local binary_name="$PROJECT_NAME"
	if [ "$goos" = "windows" ]; then
		binary_name="${PROJECT_NAME}.exe"
	fi

	local platform_dir="${DIST_DIR}/${PROJECT_NAME}-${version}-${goos}-${goarch}"
	mkdir -p "$platform_dir"

	log_info "Building for $goos/$goarch..."

	# Build binary
	GOOS="$goos" GOARCH="$goarch" go build \
		-ldflags="-s -w -X main.version=$version" \
		-o "${platform_dir}/${binary_name}" \
		. || die "Build failed for $goos/$goarch"

	# Copy additional files
	cp LICENSE "$platform_dir/"
	cp README.md "$platform_dir/"
	cp CHANGELOG.md "$platform_dir/"

	log_success "Built $goos/$goarch"
}

# Create distribution archives
create_archives() {
	local version="$1"

	log_info "Creating distribution archives..."

	cd "$DIST_DIR" || die "Failed to enter $DIST_DIR"

	for dir in ${PROJECT_NAME}-${version}-*; do
		if [ ! -d "$dir" ]; then
			continue
		fi

		# Determine archive format based on platform
		if [[ "$dir" == *"windows"* ]]; then
			# Use zip for Windows
			local archive="${dir}.zip"
			log_info "Creating $archive..."
			zip -rq "$archive" "$dir" || die "Failed to create $archive"
		else
			# Use tar.gz for Unix-like systems
			local archive="${dir}.tar.gz"
			log_info "Creating $archive..."
			tar -czf "$archive" "$dir" || die "Failed to create $archive"
		fi

		log_success "Created $archive"
	done

	cd - >/dev/null
}

# Build all distribution packages
build_distributions() {
	local version="$1"

	log_info "Building distribution packages..."

	# Clean and create dist directory
	rm -rf "$DIST_DIR"
	mkdir -p "$DIST_DIR"

	# Build for each platform
	for platform in "${PLATFORMS[@]}"; do
		local goos="${platform%/*}"
		local goarch="${platform#*/}"
		build_binary "$goos" "$goarch" "$version"
	done

	# Create archives
	create_archives "$version"

	log_success "All distributions built"
}

# Create GitHub release
create_github_release() {
	local version="$1"

	log_info "Creating GitHub release $version..."

	# Extract changelog entry for this version
	local release_notes
	release_notes=$(awk "/^## \[$version\]/,/^## \[/" CHANGELOG.md | sed '1d;$d')

	if [ -z "$release_notes" ]; then
		release_notes="Release $version"
	fi

	# Get list of archives
	local archives=()
	for file in ${DIST_DIR}/*.{tar.gz,zip}; do
		if [ -f "$file" ]; then
			archives+=("$file")
		fi
	done

	if [ ${#archives[@]} -eq 0 ]; then
		die "No distribution archives found in $DIST_DIR"
	fi

	log_info "Uploading ${#archives[@]} distribution packages..."

	# Create release with gh
	gh release create "$version" \
		"${archives[@]}" \
		--title "$version" \
		--notes "$release_notes" ||
		die "Failed to create GitHub release"

	log_success "GitHub release created: $version"
}

# Commit and tag the release
commit_and_tag() {
	local version="$1"

	log_info "Committing and tagging release..."

	# Add CHANGELOG.md
	git add CHANGELOG.md

	# Commit
	git commit -m "Release $version" || die "Failed to commit"

	# Create tag
	git tag -a "$version" -m "Release $version" || die "Failed to create tag"

	# Push
	git push origin main || die "Failed to push commits"
	git push origin "$version" || die "Failed to push tag"

	log_success "Committed and tagged $version"
}

# Main function
main() {
	local bump_type="${1:-minor}"

	echo ""
	log_info "=== $PROJECT_NAME Release Script ==="
	echo ""

	# Check requirements
	check_requirements

	# Verify workspace is clean
	check_workspace

	# Compute next version
	local version
	version=$(compute_version "$bump_type")

	# Check tag doesn't exist
	check_tag_exists "$version"

	# Confirm with user
	echo ""
	log_warn "Ready to release $version"
	read -rp "Continue? (y/N): " confirm
	if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
		log_info "Release cancelled"
		exit 0
	fi
	echo ""

	# Update CHANGELOG
	update_changelog "$version"

	# Run tests
	log_info "Running tests..."
	go test ./... || die "Tests failed"
	log_success "Tests passed"

	# Build distributions
	build_distributions "$version"

	# Commit and tag
	commit_and_tag "$version"

	# Create GitHub release
	create_github_release "$version"

	# Clean up dist directory (optional)
	log_info "Cleaning up..."
	rm -rf "$DIST_DIR"

	echo ""
	log_success "=== Release $version completed successfully! ==="
	echo ""
	log_info "View release at: https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/releases/tag/$version"
	echo ""
}

# Run main function
main "$@"
