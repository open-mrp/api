package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func table(name string, extra string) string {
	return fmt.Sprintf("CREATE TABLE `%s` (\n  `id` bigint NOT NULL,\n  `note` varchar(64) NOT NULL DEFAULT 'a;b'%s,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci", name, extra)
}

// v2.18.1 changed 19 tables in one release and PlanetScale refused the deploy request. The same shape here has to come out as parts of at most 10, each carrying only its own tables' DDL.
func TestBuildPlan_SplitsPastTheTableLimit(t *testing.T) {
	var from, to []string
	for i := 1; i <= 19; i++ {
		name := fmt.Sprintf("t%02d", i)
		from = append(from, table(name, ""))
		to = append(to, table(name, ",\n  `added` int DEFAULT NULL"))
	}
	from = append(from, table("untouched", ""))
	to = append(to, table("untouched", ""))
	to = append(to, table("brand_new", ""))

	plan, err := buildPlan(from, to, 10)
	require.NoError(t, err)
	require.Equal(t, 20, plan.Tables)
	require.Len(t, plan.Parts, 2)

	seen := map[string]bool{}
	for _, part := range plan.Parts {
		require.LessOrEqual(t, len(part.Tables), 10)
		require.Len(t, part.Statements, len(part.Tables))
		for _, name := range part.Tables {
			require.False(t, seen[name], "table %s in two parts", name)
			seen[name] = true
			require.True(t, anyContains(part.Statements, "`"+name+"`"), "part with %s has no DDL for it", name)
		}
	}
	require.False(t, seen["untouched"])
	require.True(t, seen["brand_new"])
}

// A release at or under the limit is one part, which the release script deploys from the original branch exactly as before.
func TestBuildPlan_SmallReleaseIsOnePart(t *testing.T) {
	plan, err := buildPlan(
		[]string{table("a", ""), table("b", "")},
		[]string{table("a", ",\n  `x` int DEFAULT NULL"), table("b", "")},
		10,
	)
	require.NoError(t, err)
	require.Equal(t, 1, plan.Tables)
	require.Len(t, plan.Parts, 1)
	require.Equal(t, []string{"a"}, plan.Parts[0].Tables)
}

// A view and the table it reads change together or the view's part fails on a column the other part has not added yet.
func TestBuildPlan_KeepsDependentDiffsTogether(t *testing.T) {
	from := []string{table("a", ""), table("b", ""), table("c", "")}
	to := []string{
		table("a", ",\n  `x` int DEFAULT NULL"),
		table("b", ",\n  `x` int DEFAULT NULL"),
		table("c", ",\n  `x` int DEFAULT NULL"),
		"CREATE VIEW `v` AS SELECT `a`.`id`, `a`.`x` FROM `a`",
	}

	plan, err := buildPlan(from, to, 2)
	require.NoError(t, err)
	require.Len(t, plan.Parts, 2)
	partWith := func(name string) int {
		for i, p := range plan.Parts {
			for _, n := range p.Tables {
				if n == name {
					return i
				}
			}
		}
		return -1
	}
	require.Equal(t, partWith("a"), partWith("v"))

	_, err = buildPlan(from, to, 1)
	require.ErrorContains(t, err, "cannot be split")
}

func TestStatementsRoundTripThroughFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "part.sql")
	in := []string{table("a", ""), "ALTER TABLE `a` ADD COLUMN `y` varchar(8) DEFAULT ';'"}
	require.NoError(t, writeStatements(path, in))
	out, err := readStatements(path)
	require.NoError(t, err)
	require.Equal(t, in, out)
}

func TestMysqlConfig_ReadsPscaleConnectURL(t *testing.T) {
	cfg, err := mysqlConfig("mysql://root@127.0.0.1:3306/augno_core")
	require.NoError(t, err)
	require.Equal(t, "root", cfg.User)
	require.Equal(t, "127.0.0.1:3306", cfg.Addr)
	require.Equal(t, "augno_core", cfg.DBName)

	cfg, err = mysqlConfig("mysql://u:p%40ss@db.example:3307/x?tls=true")
	require.NoError(t, err)
	require.Equal(t, "p@ss", cfg.Passwd)
	require.Equal(t, "true", cfg.TLSConfig)
}

func anyContains(statements []string, s string) bool {
	for _, stmt := range statements {
		if strings.Contains(stmt, s) {
			return true
		}
	}
	return false
}
