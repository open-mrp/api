#!/usr/bin/env bash

# release-migration-changes.sh
# Shared answer to "which migrations does this release add?". Sourced by planetscale-release-branch.sh
# and release-migration-manifest.sh, so the deploy request, the manifest and the release PR comment all
# agree on it. release.yml's deploy job inlines the same rule: a manual resume runs that file from the
# default branch against an older tag, which may not carry this script.
#
# A migration is new when its goose version is absent at the previous release tag. goose tracks
# versions, not file contents, so an edit to a shipped migration (a comment, a renamed file) can never
# reach prod and must not cut a PlanetScale branch: #305 reworded two shipped migrations' comments and
# opened a deploy request with nothing in it.

# The newest release tag reachable from $1, other than $2 when given.
release_previous_tag() {
    local base_ref="$1" exclude="${2:-}"
    git tag --list 'v*' --sort=-v:refname --merged "$base_ref" | grep -vxF "${exclude:-//none//}" | head -1 || true
}

# Prints "<version> <file>" for each goose migration file read from stdin, version as a plain integer.
release_migration_versions() {
    local file version
    while IFS= read -r file; do
        [ -n "$file" ] || continue
        case "$file" in *.sql) ;; *) continue ;; esac
        version="$(basename "$file")"
        version="${version%%_*}"
        case "$version" in ''|*[!0-9]*) continue ;; esac
        echo "$((10#$version)) $file"
    done
}

# The migration files in $3 at $2 whose version $1 does not have, one per line in version order. With
# no previous tag, every migration is new.
release_new_migrations() {
    local previous_tag="$1" base_ref="$2" dir="$3" shipped

    if [ -z "$previous_tag" ]; then
        git ls-tree -r --name-only "$base_ref" -- "$dir" 2>/dev/null | release_migration_versions | sort -n | awk '{ print $2 }'
        return 0
    fi

    shipped="$(git ls-tree -r --name-only "$previous_tag" -- "$dir" 2>/dev/null | release_migration_versions | awk '{ printf "%s ", $1 }')"
    git ls-tree -r --name-only "$base_ref" -- "$dir" 2>/dev/null | release_migration_versions | sort -n |
        awk -v shipped="$shipped" 'BEGIN { n = split(shipped, s, " "); for (i = 1; i <= n; i++) seen[s[i]] = 1 } !($1 in seen) { print $2 }'
}

# Identifies exactly what a deploy request was cut from: the previous tag (which decides what is
# recorded as shipped) and the content of every new migration. A rerun with the same fingerprint would
# produce the same schema, so the deploy request already open can stand.
release_migrations_fingerprint() {
    local previous_tag="$1" base_ref="$2"
    shift 2
    {
        echo "previous=$previous_tag"
        [ "$#" -eq 0 ] || git ls-tree "$base_ref" -- "$@"
    } | shasum -a 256 | cut -c1-16
}
