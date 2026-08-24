package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const migrationsDir = "../../../../../shared/db/migrations"

// A purge stops at the first table that errors and every table listed after it is left behind, so a name that drifted out of the schema silently turns every account purge into a partial one. scheduled_message survived here for months after scheduled messages became message rows.
func TestPurgeTargets_MatchSchema(t *testing.T) {
	t.Parallel()

	schema := loadSchemaColumns(t)

	for _, target := range purgeTargets {
		columns, ok := schema[target.Table]
		if !ok {
			t.Errorf("purge target table %q does not exist in the schema", target.Table)
			continue
		}
		if !columns[target.Column] {
			t.Errorf("purge target %s.%s: table exists but has no %q column", target.Table, target.Column, target.Column)
		}
	}
}

var (
	createTableRe = regexp.MustCompile("(?m)^CREATE TABLE (?:IF NOT EXISTS )?`([^`]+)` \\(")
	columnRe      = regexp.MustCompile("^\\s+`([^`]+)` ")
	dropTableRe   = regexp.MustCompile("(?mi)^DROP TABLE (?:IF EXISTS )?`([^`]+)`")
	alterTableRe  = regexp.MustCompile("(?mi)^ALTER TABLE `([^`]+)`")
	addColumnRe   = regexp.MustCompile("(?i)ADD COLUMN `([^`]+)`")
	dropColumnRe  = regexp.MustCompile("(?i)DROP COLUMN `([^`]+)`")
	gooseDownRe   = regexp.MustCompile(`(?m)^-- \+goose Down`)
)

// loadSchemaColumns replays the Up half of every migration into table name -> column set. It reads the
// whole directory rather than the baseline alone because the baseline is frozen at the goose cutover:
// tables and columns added since live in the migrations layered on top of it.
func loadSchemaColumns(t *testing.T) map[string]map[string]bool {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no migrations found in %s", migrationsDir)
	}
	sort.Strings(files)

	schema := make(map[string]map[string]bool)

	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		// Down statements undo what Up did; replaying both leaves the schema empty.
		up := string(raw)
		if loc := gooseDownRe.FindStringIndex(up); loc != nil {
			up = up[:loc[0]]
		}

		applyMigration(schema, up)
	}

	if len(schema) == 0 {
		t.Fatalf("parsed no tables from %s", migrationsDir)
	}

	return schema
}

// applyMigration handles the DDL subset these migrations use: table creation, table drops, and column
// add/drop. Anything else does not move a purge target's (table, column) pair. Statements are applied
// in file order because the baseline drops each table immediately before recreating it.
func applyMigration(schema map[string]map[string]bool, sql string) {
	for _, statement := range strings.Split(sql, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		if m := createTableRe.FindStringSubmatch(statement); m != nil {
			columns := make(map[string]bool)
			for line := range strings.SplitSeq(statement, "\n") {
				if c := columnRe.FindStringSubmatch(line); c != nil {
					columns[c[1]] = true
				}
			}
			schema[m[1]] = columns
			continue
		}

		if m := dropTableRe.FindStringSubmatch(statement); m != nil {
			delete(schema, m[1])
			continue
		}

		if m := alterTableRe.FindStringSubmatch(statement); m != nil {
			columns, ok := schema[m[1]]
			if !ok {
				continue
			}
			for _, add := range addColumnRe.FindAllStringSubmatch(statement, -1) {
				columns[add[1]] = true
			}
			for _, drop := range dropColumnRe.FindAllStringSubmatch(statement, -1) {
				delete(columns, drop[1])
			}
		}
	}
}
