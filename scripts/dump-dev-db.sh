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

# Docker Compose connection details (see docker-compose.yml)
HOST="127.0.0.1"
PORT="3306"
USERNAME="root"
DATABASE="openmrp"
export MYSQL_PWD="Testing123!"

print_status "Preparing to generate Goose migration (db/migrations/0001_initial.sql)..."

# Test connection first
print_status "Testing database connection..."
MYSQL_AUTH=("-h" "$HOST" "-P" "$PORT" "-u" "$USERNAME" "--protocol=tcp")

if ! mysql "${MYSQL_AUTH[@]}" "$DATABASE" -e "SELECT 1;" &> /dev/null; then
    print_error "Failed to connect to database. Is 'docker compose up' running?"
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
# goose_db_version is goose's own bookkeeping table. Goose creates it itself before
# running anything, so a migration that also creates it is at best redundant and at
# worst conflicts on a fresh database.
# --skip-comments drops the banner, the per-table "Table structure for" headers and the
# trailing "Dump completed on" timestamp: they carry the dumping client's version and the
# wall clock, so every regeneration churns the diff even when the schema is identical.
if ! mysqldump "${MYSQL_AUTH[@]}" \
    "$DATABASE" \
    --no-data \
    --triggers \
    --single-transaction \
    --skip-comments \
    --skip-set-charset \
    --ignore-table="$DATABASE.goose_db_version" \
    > "$TEMP_SCHEMA_FILE" 2>/dev/null; then
    print_error "Error creating schema dump"
    rm -f "$TEMP_SCHEMA_FILE"
    exit 1
fi

# Drop mysqldump's version-gated session SET statements for the same reason: their version
# guards move with the client (utf8 -> utf8mb4, 40101 -> 50503). None of them are load-bearing
# for a schema-only load; the schema carries no foreign keys, so create order is free.
# The pattern is deliberately narrow so /*!50003 CREATE ... TRIGGER */ bodies survive.
grep -vE '^/\*![0-9]+ SET .*\*/;$' "$TEMP_SCHEMA_FILE" | cat -s > "$TEMP_SCHEMA_FILE.clean"
mv "$TEMP_SCHEMA_FILE.clean" "$TEMP_SCHEMA_FILE"

# Generate DROP TABLE statements for Down migration (reverse order)
print_status "Generating DROP TABLE statements for down migration..."
TABLES=$(mysql "${MYSQL_AUTH[@]}" -D "$DATABASE" -e "SHOW TABLES;" -s --skip-column-names | grep -v '^goose_db_version$')
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
rm -f "$TEMP_SCHEMA_FILE" "$TEMP_SCHEMA_FILE.clean" "$TEMP_DROP_FILE"

print_status "Goose migration created successfully: $OUTPUT_FILE"
print_status "File size: $(du -h "$OUTPUT_FILE" | cut -f1)"