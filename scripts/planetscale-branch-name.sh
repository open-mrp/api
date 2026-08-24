#!/usr/bin/env bash

# planetscale-branch-name.sh
# Shared derivation of a release's PlanetScale branch name. Sourced by the prepare and deploy scripts
# so both reach the same name from the same version — that agreement is what lets the deploy step find
# the deploy request without any state handed to it by the PR run.

# PlanetScale branch names take lowercase alphanumerics and dashes, so a version like v1.2.0 has to be
# flattened to 1-2-0.
planetscale_release_branch() {
    local version="${1#v}"
    local slug

    slug="$(echo "$version" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"

    if [ -z "$slug" ]; then
        echo "Could not derive a branch name from version '$1'" >&2
        return 1
    fi

    echo "release-$slug"
}
