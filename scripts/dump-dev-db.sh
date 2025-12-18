#!/bin/bash

# Dump Dev Database Script
# This script generates a Goose-ready migration at shared/db/migrations/0001_initial.sql

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

# Load environment variables from .env file (only variable assignments, not comments)
if [ -f ".env" ]; then
    while IFS= read -r line; do
        # Skip empty lines and comments
        if [[ -n "$line" && ! "$line" =~ ^[[:space:]]*# ]]; then
            # Export the variable
            export "$line"
        fi
    done < .env
fi

# Determine connection details from DB_URL (mysql://...) or DB_URL (user:pass@tcp(host:port)/db)
print_status "Preparing to generate Goose migration (db/migrations/0001_initial.sql)..."

# Prefer DB_URL if present
if [ -n "$DB_URL" ]; then
    if [[ $DB_URL =~ mysql://([^:]+):([^@]+)@([^:]+):([^/]+)/(.+) ]]; then
        USERNAME="${BASH_REMATCH[1]}"
        PASSWORD="${BASH_REMATCH[2]}"
        HOST="${BASH_REMATCH[3]}"
        PORT="${BASH_REMATCH[4]}"
        DATABASE="${BASH_REMATCH[5]}"
    else
        print_error "Invalid DB_URL format. Expected: mysql://username:password@host:port/database_name"
        exit 1
    fi
elif [ -n "$DB_URL" ]; then
    DEV_DB_URL=${DB_URL}
    USERNAME=$(echo "$DEV_DB_URL" | sed -n 's/^\([^:]*\):.*/\1/p')
    PASSWORD=$(echo "$DEV_DB_URL" | sed -n 's/^[^:]*:\([^@]*\)@.*/\1/p')
    HOST=$(echo "$DEV_DB_URL" | sed -n 's/.*@tcp(\([^:]*\):.*/\1/p')
    PORT=$(echo "$DEV_DB_URL" | sed -n 's/.*@tcp([^:]*:\([^)]*\)).*/\1/p')
    DATABASE=$(echo "$DEV_DB_URL" | sed -n 's/.*@tcp([^)]*)\/\([^?]*\).*/\1/p')
    if [ -z "$USERNAME" ] || [ -z "$HOST" ] || [ -z "$PORT" ] || [ -z "$DATABASE" ]; then
        print_error "Invalid DB_URL format. Expected: user:password@tcp(host:port)/database"
        print_error "Parsed values:"
        print_error "  User: $USERNAME"
        print_error "  Host: $HOST"
        print_error "  Port: $PORT"
        print_error "  Name: $DATABASE"
        exit 1
    fi
else
    print_error "No DB_URL or DB_URL found in environment. Ensure .env sets one of them."
    exit 1
fi

print_status "Parsed connection details:"
print_status "  Host: $HOST"
print_status "  Port: $PORT"
print_status "  Database: $DATABASE"
print_status "  User: $USERNAME"

# Test connection first
print_status "Testing database connection..."
MYSQL_AUTH=("-h" "$HOST" "-P" "$PORT" "-u" "$USERNAME")
if [ -n "$PASSWORD" ]; then
    MYSQL_AUTH+=("-p$PASSWORD")
fi

if ! mysql "${MYSQL_AUTH[@]}" "$DATABASE" -e "SELECT 1;" &> /dev/null; then
    print_error "Failed to connect to database. Please check DB_URL/DB_URL."
    exit 1
fi

print_status "Database connection successful!"

# Paths and temporary files
MIGRATIONS_DIR="shared/db/migrations"
OUTPUT_FILE="$MIGRATIONS_DIR/0001_initial.sql"
TEMP_SCHEMA_FILE="temp_schema_dump.sql"
TEMP_DROP_FILE="temp_drop_statements.sql"

# Ensure migrations directory exists
mkdir -p "$MIGRATIONS_DIR"

# Create the schema dump (schema only, include triggers; exclude routines to avoid DELIMITER issues)
print_status "Dumping schema (no data) to temporary file..."
if ! mysqldump -h "$HOST" -P "$PORT" -u "$USERNAME" ${PASSWORD:+-p"$PASSWORD"} \
    "$DATABASE" \
    --no-data \
    --triggers \
    --single-transaction \
    --set-gtid-purged=OFF \
    > "$TEMP_SCHEMA_FILE" 2>/dev/null; then
    print_error "Error creating schema dump"
    rm -f "$TEMP_SCHEMA_FILE"
    exit 1
fi

# Generate DROP TABLE statements for Down migration (reverse order)
print_status "Generating DROP TABLE statements for down migration..."
TABLES=$(mysql "${MYSQL_AUTH[@]}" -D "$DATABASE" -e "SHOW TABLES;" -s --skip-column-names)
if [ -n "$TABLES" ]; then
    echo "$TABLES" | while read -r table; do
        if [ -n "$table" ]; then
            echo "DROP TABLE IF EXISTS \`$table\`;"
        fi
    done | tail -r > "$TEMP_DROP_FILE"
else
    echo "-- No tables found" > "$TEMP_DROP_FILE"
fi

# Write Goose migration file
print_status "Writing Goose migration to $OUTPUT_FILE..."
{
    echo "-- +goose Up"
    echo ""
    cat "$TEMP_SCHEMA_FILE"
    echo ""
    echo "-- +goose Down"
    echo ""
    echo "-- Disable foreign key checks to allow dropping in any order"
    echo "SET FOREIGN_KEY_CHECKS=0;"
    cat "$TEMP_DROP_FILE"
    echo "SET FOREIGN_KEY_CHECKS=1;"
    echo ""
} > "$OUTPUT_FILE"

# Cleanup
rm -f "$TEMP_SCHEMA_FILE" "$TEMP_DROP_FILE"

print_status "Goose migration created successfully: $OUTPUT_FILE"
print_status "File size: $(du -h "$OUTPUT_FILE" | cut -f1)"