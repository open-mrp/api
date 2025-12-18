#!/bin/bash

# Database Migration Script
# This script wraps goose for database migrations

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
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

# Protection: Only allow running on localhost
if [[ "$DB_URL" != *"localhost"* ]] && [[ "$DB_URL" != *"127.0.0.1"* ]]; then
    print_error "Migrations are restricted to localhost/127.0.0.1 for safety."
    print_error "Current DB_URL: $DB_URL"
    exit 1
fi

COMMAND=$1
shift

MIGRATIONS_DIR="./db/migrations"

case "$COMMAND" in
    up)
        print_status "Running migrations up..."
        goose -dir "$MIGRATIONS_DIR" up "$@"
        ;;
    down)
        print_warning "This will rollback database migrations and potentially lose data!"
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Rolling back migrations..."
            goose -dir "$MIGRATIONS_DIR" down "$@"
        else
            print_status "Operation cancelled."
        fi
        ;;
    status)
        print_status "Checking migration status..."
        goose -dir "$MIGRATIONS_DIR" status "$@"
        ;;
    reset)
        print_warning "This will reset your database and drop all data!"
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Resetting database..."
            goose -dir "$MIGRATIONS_DIR" reset "$@"
        else
            print_status "Operation cancelled."
        fi
        ;;
    *)
        echo "Usage: $0 {up|down|status|reset} [goose_args...]"
        exit 1
        ;;
esac
