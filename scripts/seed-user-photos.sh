#!/usr/bin/env bash

# Seed User Photos to S3
# Uploads the seeded account-users' avatar images to the user-photos S3 bucket
# using the exact key convention core-service expects: {account_id}/{user_id}.png
# (the key extension is always .png regardless of source format — see
# userSvcImpl.UploadUserPhoto / accountUserSvcImpl.resolveImageURL).
#
# The seed SQL (shared/db/seed/0004_auth.sql) sets user.image_url for these users,
# which is the authoritative "avatar exists" signal; this script puts the matching
# bytes in S3 so the presigned GET URLs resolve to a real image.
#
# Usage: ./scripts/seed-user-photos.sh
#
# Env overrides (defaults target the dev bucket + seeded Acme account):
#   USER_PHOTOS_BUCKET  S3 bucket (default: augno-user-photos-dev-us-east-2)
#   SEED_ACCOUNT_ID     account id prefix (default: ac_01k0a5smf9ekb8rqg12555zjqa)
#   AWS_REGION          AWS region (default: us-east-2)

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

USER_PHOTOS_BUCKET="${USER_PHOTOS_BUCKET:-augno-user-photos-dev-us-east-2}"
SEED_ACCOUNT_ID="${SEED_ACCOUNT_ID:-ac_01k0a5smf9ekb8rqg12555zjqa}"
AWS_REGION="${AWS_REGION:-us-east-2}"

ASSET_DIR="$REPO_ROOT/shared/db/seed/assets/user-photos"

if ! command -v aws &> /dev/null; then
    error "aws CLI is not installed."
    exit 1
fi

if [ ! -d "$ASSET_DIR" ]; then
    error "Asset directory not found: $ASSET_DIR"
    exit 1
fi

info "Uploading seed user photos to s3://${USER_PHOTOS_BUCKET}/${SEED_ACCOUNT_ID}/"

shopt -s nullglob
found=0
for src in "$ASSET_DIR"/*.jpg; do
    found=1
    user_id="$(basename "$src" .jpg)"
    key="${SEED_ACCOUNT_ID}/${user_id}.png"
    info "  ${user_id} -> s3://${USER_PHOTOS_BUCKET}/${key}"
    aws s3 cp "$src" "s3://${USER_PHOTOS_BUCKET}/${key}" \
        --region "$AWS_REGION" \
        --content-type "image/jpeg" \
        --only-show-errors
done

if [ "$found" -eq 0 ]; then
    error "No .jpg images found in $ASSET_DIR"
    exit 1
fi

info "User photo seed complete."
