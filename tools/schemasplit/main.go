// Command schemasplit splits a release's schema change into parts small enough for one PlanetScale deploy request each.
//
// A deploy request may change at most 10 tables. A release that touches more (v2.18.1 touched 19) used to be undeployable until its migrations were split across several releases by hand. This tool does that split mechanically: it diffs the schema prod has against the schema the release's migrations produce, using the same Vitess schemadiff PlanetScale computes deploy requests with, and packs the per-table DDL into parts of at most -max-tables tables. Diffs that depend on each other (a view and its table, a foreign key and its parent) always land in the same part, so every part is deployable on its own and in any order.
//
// planetscale-release-branch.sh drives it: `dump` reads a branch's schema before and after the migrations run, `plan` writes the parts, and `apply` runs one part against a branch freshly cut from prod. dump and apply connect through DATABASE_URL, which `pscale connect --execute` sets.
//
// Usage:
//
//	schemasplit dump -out before.sql
//	schemasplit plan -from before.sql -to after.sql -max-tables 10 -out parts/
//	schemasplit apply -file parts/part-1.sql
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-sql-driver/mysql"
	vtlog "vitess.io/vitess/go/vt/log"
	"vitess.io/vitess/go/vt/schemadiff"
)

// statementSeparator ends every statement in the files this tool writes. CREATE statements carry semicolons inside string literals (column comments, defaults), so a plain `;` split would cut them apart.
const statementSeparator = "\n;;\n"

// Part is one deploy request's share of the release: the tables it changes and the DDL that changes them, in execution order.
type Part struct {
	Tables     []string `json:"tables"`
	Statements []string `json:"statements"`
}

// Plan is what `plan` writes to plan.json. Tables counts every table the release changes, across all parts.
type Plan struct {
	Tables int    `json:"tables"`
	Parts  []Part `json:"parts"`
}

func main() {
	// schemadiff logs every INSTANT DDL eligibility check at info; the CI log only needs the plan.
	vtlog.SwapLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: schemasplit dump|plan|apply [flags]")
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "dump":
		err = runDump(os.Args[2:])
	case "plan":
		err = runPlan(os.Args[2:])
	case "apply":
		err = runApply(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "schemasplit:", err)
		os.Exit(1)
	}
}

func runDump(args []string) error {
	fs := flag.NewFlagSet("dump", flag.ExitOnError)
	out := fs.String("out", "", "file to write the schema to")
	_ = fs.Parse(args)
	if *out == "" {
		return errors.New("dump: -out is required")
	}

	db, err := openDatabaseURL()
	if err != nil {
		return err
	}
	defer db.Close()

	statements, err := dumpSchema(context.Background(), db)
	if err != nil {
		return err
	}
	return writeStatements(*out, statements)
}

func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	from := fs.String("from", "", "schema before the release (prod)")
	to := fs.String("to", "", "schema after the release's migrations")
	maxTables := fs.Int("max-tables", 10, "most tables one part may change")
	out := fs.String("out", "", "directory to write plan.json and part-N.sql into")
	_ = fs.Parse(args)
	if *from == "" || *to == "" || *out == "" {
		return errors.New("plan: -from, -to and -out are required")
	}

	fromSQL, err := readStatements(*from)
	if err != nil {
		return err
	}
	toSQL, err := readStatements(*to)
	if err != nil {
		return err
	}

	plan, err := buildPlan(fromSQL, toSQL, *maxTables)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	for i, part := range plan.Parts {
		if err := writeStatements(filepath.Join(*out, fmt.Sprintf("part-%d.sql", i+1)), part.Statements); err != nil {
			return err
		}
	}
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*out, "plan.json"), encoded, 0o644); err != nil {
		return err
	}

	fmt.Printf("%d tables changed, %d part(s) of at most %d tables\n", plan.Tables, len(plan.Parts), *maxTables)
	for i, part := range plan.Parts {
		fmt.Printf("  part %d: %s\n", i+1, strings.Join(part.Tables, ", "))
	}
	return nil
}

func runApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	file := fs.String("file", "", "part file written by plan")
	_ = fs.Parse(args)
	if *file == "" {
		return errors.New("apply: -file is required")
	}

	statements, err := readStatements(*file)
	if err != nil {
		return err
	}

	db, err := openDatabaseURL()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	for _, stmt := range statements {
		fmt.Println(stmt + ";")
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply: %w", err)
		}
	}
	return nil
}

// buildPlan diffs the two schemas and packs the per-table diffs into parts. Diffs linked by a schemadiff dependency are kept together; a group of linked diffs larger than maxTables cannot be split and is an error rather than a part that would fail at deploy time.
func buildPlan(fromSQL, toSQL []string, maxTables int) (*Plan, error) {
	if maxTables < 1 {
		return nil, fmt.Errorf("max-tables must be at least 1 (got %d)", maxTables)
	}

	env := schemadiff.NewTestEnv()
	fromSchema, err := schemadiff.NewSchemaFromQueries(env, fromSQL)
	if err != nil {
		return nil, fmt.Errorf("parse the schema before the release: %w", err)
	}
	toSchema, err := schemadiff.NewSchemaFromQueries(env, toSQL)
	if err != nil {
		return nil, fmt.Errorf("parse the schema after the release: %w", err)
	}

	diff, err := schemadiff.DiffSchemas(env, fromSchema, toSchema, &schemadiff.DiffHints{})
	if err != nil {
		return nil, fmt.Errorf("diff schemas: %w", err)
	}
	ordered, err := diff.OrderedDiffs(context.Background())
	if err != nil {
		return nil, fmt.Errorf("order diffs: %w", err)
	}

	// Union-find over table names: one table can have several diffs, and a dependency joins two tables' fates.
	parent := map[string]string{}
	var find func(string) string
	find = func(name string) string {
		if parent[name] == name {
			return name
		}
		parent[name] = find(parent[name])
		return parent[name]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}

	var tableOrder []string
	for _, d := range ordered {
		name := d.EntityName()
		if _, seen := parent[name]; !seen {
			parent[name] = name
			tableOrder = append(tableOrder, name)
		}
	}
	for _, dep := range diff.AllDependencies() {
		union(dep.Diff().EntityName(), dep.DependentDiff().EntityName())
	}

	// Groups keep the order their first table appears in, so parts deploy in the order the diffs would have run.
	groupTables := map[string][]string{}
	var groupOrder []string
	for _, name := range tableOrder {
		root := find(name)
		if _, ok := groupTables[root]; !ok {
			groupOrder = append(groupOrder, root)
		}
		groupTables[root] = append(groupTables[root], name)
	}

	plan := &Plan{Tables: len(tableOrder)}
	partOf := map[string]int{}
	for _, root := range groupOrder {
		tables := groupTables[root]
		if len(tables) > maxTables {
			return nil, fmt.Errorf("%d tables depend on each other and cannot be split below %d: %s", len(tables), maxTables, strings.Join(tables, ", "))
		}
		last := len(plan.Parts) - 1
		if last < 0 || len(plan.Parts[last].Tables)+len(tables) > maxTables {
			plan.Parts = append(plan.Parts, Part{})
			last++
		}
		plan.Parts[last].Tables = append(plan.Parts[last].Tables, tables...)
		for _, t := range tables {
			partOf[t] = last
		}
	}

	for _, d := range ordered {
		stmt := d.CanonicalStatementString()
		if stmt == "" {
			continue
		}
		i := partOf[d.EntityName()]
		plan.Parts[i].Statements = append(plan.Parts[i].Statements, stmt)
	}
	for i := range plan.Parts {
		sort.Strings(plan.Parts[i].Tables)
	}
	return plan, nil
}

// dumpSchema returns a CREATE statement for every table and view in the connected database.
func dumpSchema(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SHOW FULL TABLES")
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	type entity struct{ name, kind string }
	var entities []entity
	for rows.Next() {
		var e entity
		if err := rows.Scan(&e.name, &e.kind); err != nil {
			rows.Close()
			return nil, err
		}
		entities = append(entities, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var statements []string
	for _, e := range entities {
		quoted := "`" + strings.ReplaceAll(e.name, "`", "``") + "`"
		if e.kind == "VIEW" {
			// SHOW CREATE VIEW returns View, Create View, character_set_client, collation_connection.
			var name, create, cs, coll string
			if err := db.QueryRowContext(ctx, "SHOW CREATE VIEW "+quoted).Scan(&name, &create, &cs, &coll); err != nil {
				return nil, fmt.Errorf("show create view %s: %w", e.name, err)
			}
			statements = append(statements, create)
			continue
		}
		var name, create string
		if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE "+quoted).Scan(&name, &create); err != nil {
			return nil, fmt.Errorf("show create table %s: %w", e.name, err)
		}
		statements = append(statements, create)
	}
	return statements, nil
}

// openDatabaseURL connects using DATABASE_URL in its mysql://user[:pass]@host[:port]/db form, which is what `pscale connect --execute` sets.
func openDatabaseURL() (*sql.DB, error) {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		return nil, errors.New("DATABASE_URL is not set; run under `pscale connect --execute`")
	}
	cfg, err := mysqlConfig(raw)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	// One connection: apply's statements must run in order on the same session.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect: %w", err)
	}
	return db, nil
}

func mysqlConfig(raw string) (*mysql.Config, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if u.Scheme != "mysql" {
		return nil, fmt.Errorf("DATABASE_URL must start with mysql:// (got %s://)", u.Scheme)
	}

	cfg := mysql.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = u.Host
	if u.Port() == "" {
		cfg.Addr = u.Host + ":3306"
	}
	cfg.DBName = strings.TrimPrefix(u.Path, "/")
	if u.User != nil {
		cfg.User = u.User.Username()
		cfg.Passwd, _ = u.User.Password()
	}
	if tls := u.Query().Get("tls"); tls != "" {
		cfg.TLSConfig = tls
	}
	if cfg.DBName == "" {
		return nil, errors.New("DATABASE_URL has no database name")
	}
	return cfg, nil
}

func writeStatements(path string, statements []string) error {
	return os.WriteFile(path, []byte(strings.Join(statements, statementSeparator)+statementSeparator), 0o644)
}

func readStatements(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var statements []string
	for _, s := range strings.Split(string(data), statementSeparator) {
		if s = strings.TrimSpace(s); s != "" {
			statements = append(statements, s)
		}
	}
	return statements, nil
}
