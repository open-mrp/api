package repository

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const schemaDumpPath = "../../../../../shared/db/migrations/0001_initial.sql"

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
	createTableRe = regexp.MustCompile("(?m)^CREATE TABLE `([^`]+)` \\(")
	columnRe      = regexp.MustCompile("^\\s+`([^`]+)` ")
)

// loadSchemaColumns reads the mysqldump-format schema into table name -> column set.
func loadSchemaColumns(t *testing.T) map[string]map[string]bool {
	t.Helper()

	raw, err := os.ReadFile(schemaDumpPath)
	if err != nil {
		t.Fatalf("read schema dump: %v", err)
	}

	schema := make(map[string]map[string]bool)
	var current string

	for line := range strings.SplitSeq(string(raw), "\n") {
		if m := createTableRe.FindStringSubmatch(line); m != nil {
			current = m[1]
			schema[current] = make(map[string]bool)
			continue
		}
		if current == "" {
			continue
		}
		if strings.HasPrefix(line, ")") {
			current = ""
			continue
		}
		if m := columnRe.FindStringSubmatch(line); m != nil {
			schema[current][m[1]] = true
		}
	}

	if len(schema) == 0 {
		t.Fatalf("parsed no tables from %s", schemaDumpPath)
	}

	return schema
}
