#!/usr/bin/env bash
#
# Version utilities for the OpenMRP API
#
# Version format:
#   Stable:  <minor>.<patch>.<codename>          (e.g., 1.2.forge)
#   Preview: <minor>.<patch>.<codename>-preview.<n>  (e.g., 1.3.forge-preview.1)
#
# Usage:
#   source ./scripts/version.sh
#   validate_version "1.0.forge"
#   parse_version "1.0.forge"
#

set -euo pipefail

# Regex patterns
STABLE_PATTERN='^([0-9]+)\.([0-9]+)\.([a-z][a-z0-9-]*)$'
PREVIEW_PATTERN='^([0-9]+)\.([0-9]+)\.([a-z][a-z0-9-]*)-preview\.([0-9]+)$'

# Validate a version string
# Returns 0 if valid, 1 if invalid
validate_version() {
    local version="$1"

    if [[ "$version" =~ $STABLE_PATTERN ]]; then
        return 0
    elif [[ "$version" =~ $PREVIEW_PATTERN ]]; then
        return 0
    else
        return 1
    fi
}

# Check if a version is a preview version
# Returns 0 if preview, 1 if stable
is_preview() {
    local version="$1"

    if [[ "$version" =~ $PREVIEW_PATTERN ]]; then
        return 0
    else
        return 1
    fi
}

# Check if a version is a stable version
# Returns 0 if stable, 1 if preview
is_stable() {
    local version="$1"

    if [[ "$version" =~ $STABLE_PATTERN ]]; then
        return 0
    else
        return 1
    fi
}

# Parse a version string and export components
# Sets: VERSION_MINOR, VERSION_PATCH, VERSION_CODENAME, VERSION_PREVIEW (0 if stable)
parse_version() {
    local version="$1"

    # Use sed for portable parsing (works in both bash and zsh)
    if is_preview "$version"; then
        # Preview format: minor.patch.codename-preview.n
        VERSION_MINOR=$(echo "$version" | sed -E 's/^([0-9]+)\..*/\1/')
        VERSION_PATCH=$(echo "$version" | sed -E 's/^[0-9]+\.([0-9]+)\..*/\1/')
        VERSION_CODENAME=$(echo "$version" | sed -E 's/^[0-9]+\.[0-9]+\.([a-z][a-z0-9-]*)-preview\.[0-9]+$/\1/')
        VERSION_PREVIEW=$(echo "$version" | sed -E 's/^.*-preview\.([0-9]+)$/\1/')
    elif is_stable "$version"; then
        # Stable format: minor.patch.codename
        VERSION_MINOR=$(echo "$version" | sed -E 's/^([0-9]+)\..*/\1/')
        VERSION_PATCH=$(echo "$version" | sed -E 's/^[0-9]+\.([0-9]+)\..*/\1/')
        VERSION_CODENAME=$(echo "$version" | sed -E 's/^[0-9]+\.[0-9]+\.([a-z][a-z0-9-]*)$/\1/')
        VERSION_PREVIEW="0"
    else
        echo "Error: Invalid version format: $version" >&2
        return 1
    fi

    export VERSION_MINOR VERSION_PATCH VERSION_CODENAME VERSION_PREVIEW
}

# Convert version to git tag (add v prefix)
version_to_tag() {
    local version="$1"
    echo "v${version}"
}

# Convert git tag to version (remove v prefix)
tag_to_version() {
    local tag="$1"
    echo "${tag#v}"
}

# Compare two versions
# Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
compare_versions() {
    local v1="$1"
    local v2="$2"

    parse_version "$v1"
    local v1_minor="$VERSION_MINOR"
    local v1_patch="$VERSION_PATCH"
    local v1_codename="$VERSION_CODENAME"
    local v1_preview="$VERSION_PREVIEW"

    parse_version "$v2"
    local v2_minor="$VERSION_MINOR"
    local v2_patch="$VERSION_PATCH"
    local v2_codename="$VERSION_CODENAME"
    local v2_preview="$VERSION_PREVIEW"

    # Compare minor
    if (( v1_minor < v2_minor )); then
        echo "-1"
        return
    elif (( v1_minor > v2_minor )); then
        echo "1"
        return
    fi

    # Compare patch
    if (( v1_patch < v2_patch )); then
        echo "-1"
        return
    elif (( v1_patch > v2_patch )); then
        echo "1"
        return
    fi

    # Compare codename (string comparison)
    if [[ "$v1_codename" < "$v2_codename" ]]; then
        echo "-1"
        return
    elif [[ "$v1_codename" > "$v2_codename" ]]; then
        echo "1"
        return
    fi

    # Same minor.patch.codename - compare preview
    # Stable (preview=0) is greater than any preview
    if (( v1_preview == 0 && v2_preview > 0 )); then
        echo "1"
        return
    elif (( v1_preview > 0 && v2_preview == 0 )); then
        echo "-1"
        return
    elif (( v1_preview < v2_preview )); then
        echo "-1"
        return
    elif (( v1_preview > v2_preview )); then
        echo "1"
        return
    fi

    echo "0"
}

# Get the current version from Go source (shared/version/version.go)
get_current_version() {
    local version
    version=$(go run ./cmd/print-version)

    if ! validate_version "$version"; then
        echo "Error: Invalid version from Go source: $version" >&2
        return 1
    fi

    echo "$version"
}

# Get the next preview version
# If stable: bump patch and add -preview.1
# If preview: increment preview number
get_next_preview() {
    local version="$1"

    parse_version "$version"

    if is_stable "$version"; then
        # Bump patch and add preview.1
        local new_patch=$((VERSION_PATCH + 1))
        echo "${VERSION_MINOR}.${new_patch}.${VERSION_CODENAME}-preview.1"
    else
        # Increment preview number
        local new_preview=$((VERSION_PREVIEW + 1))
        echo "${VERSION_MINOR}.${VERSION_PATCH}.${VERSION_CODENAME}-preview.${new_preview}"
    fi
}

# Get the stable version from a preview
# Strips the -preview.N suffix
get_stable_from_preview() {
    local version="$1"

    if ! is_preview "$version"; then
        echo "Error: Not a preview version: $version" >&2
        return 1
    fi

    parse_version "$version"
    echo "${VERSION_MINOR}.${VERSION_PATCH}.${VERSION_CODENAME}"
}

# Print version info
print_version_info() {
    local version="$1"

    if ! validate_version "$version"; then
        echo "Error: Invalid version: $version" >&2
        return 1
    fi

    parse_version "$version"

    echo "Version: $version"
    echo "  Minor:    $VERSION_MINOR"
    echo "  Patch:    $VERSION_PATCH"
    echo "  Codename: $VERSION_CODENAME"

    if is_preview "$version"; then
        echo "  Preview:  $VERSION_PREVIEW"
        echo "  Type:     preview"
    else
        echo "  Type:     stable"
    fi
}
