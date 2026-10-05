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

# A release that changes more tables than one deploy request allows is split into parts, each on its
# own branch: release-1-2-0-p1, release-1-2-0-p2, ...
planetscale_release_part_branch() {
    echo "$1-p$2"
}

# Each part's deploy request carries this marker in its notes. The plan id ties the parts of one prepare
# run together, so the deploy step can tell a complete set from one a re-run left half rebuilt.
planetscale_plan_marker() {
    echo "[release-plan id=$1 part=$2/$3]"
}

# Reads "<id> <part> <total>" back out of a deploy request's notes; prints nothing if there is no marker.
planetscale_parse_plan_marker() {
    echo "$1" | sed -nE 's/.*\[release-plan id=([^ ]+) part=([0-9]+)\/([0-9]+)\].*/\1 \2 \3/p' | head -1
}
