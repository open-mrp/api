#!/bin/bash

# Seed Database Script
# This script seeds the development database with data from ./db/seed/dev_database_dump.sql

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Load environment variables from .env file
if [ -f ".env" ]; then
    while IFS= read -r line; do
        if [[ -n "$line" && ! "$line" =~ ^[[:space:]]*# ]]; then
            export "$line"
        fi
    done < .env
fi

if [ -z "$DB_URL" ]; then
    print_error "DB_URL not found in environment. Ensure .env sets it."
    exit 1
fi

print_status "Parsing connection details from DB_URL..."

# Parse DB_URL: user:password@tcp(host:port)/database
USERNAME=$(echo "$DB_URL" | sed -n 's/^\([^:]*\):.*/\1/p')
PASSWORD=$(echo "$DB_URL" | sed -n 's/^[^:]*:\([^@]*\)@.*/\1/p')
HOST=$(echo "$DB_URL" | sed -n 's/.*@tcp(\([^:]*\):.*/\1/p')
PORT=$(echo "$DB_URL" | sed -n 's/.*@tcp([^:]*:\([^)]*\)).*/\1/p')
DATABASE=$(echo "$DB_URL" | sed -n 's/.*@tcp([^)]*)\/\([^?]*\).*/\1/p')

if [ -z "$USERNAME" ] || [ -z "$HOST" ] || [ -z "$PORT" ] || [ -z "$DATABASE" ]; then
    print_error "Invalid DB_URL format. Expected: user:password@tcp(host:port)/database"
    exit 1
fi

print_status "Seeding database '$DATABASE' on $HOST:$PORT as user '$USERNAME'..."

MYSQL_AUTH=("-h" "$HOST" "-P" "$PORT" "-u" "$USERNAME")
if [ -n "$PASSWORD" ]; then
    MYSQL_AUTH+=("-p$PASSWORD")
fi

if [ ! -f "./db/seed/dev_database_dump.sql" ]; then
    print_error "Seed file not found: ./db/seed/dev_database_dump.sql"
    exit 1
fi

if mysql "${MYSQL_AUTH[@]}" "$DATABASE" < ./db/seed/dev_database_dump.sql; then
    print_status "Seed data imported successfully!"
else
    print_error "Failed to import seed data."
    exit 1
fi
