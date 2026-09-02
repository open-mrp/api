// Command vtparse checks that every generated MySQL query parses on Vitess.
//
// Production is PlanetScale, so every statement goes through vtgate before it reaches MySQL. vtgate has its own SQL parser and it is stricter than MySQL 8: `FOR UPDATE OF a, b` is valid MySQL that vtgate rejects outright with "Error 1105 (HY000): syntax error near 'OF'". Nothing in this repository can catch that, because every test here — the prepare smoke test, the ledger concurrency suite, e2e — runs against plain mysql:8, which accepts it. A query can therefore be green on every check and still fail on every execution in production.
//
// This tool closes that gap by running the corpus through vtgate's own parser, which is a Go package. It reads each service's sqlc.yaml, takes the ones declaring engine: mysql, and parses every query string sqlc generated for them. A parse error here is the error production would return.
//
// It sees generated sqlc queries and nothing else. SQL assembled in Go at runtime is outside its reach, as is anything vtgate accepts but plans differently — cross-shard behavior and the transaction time limit need a real vtgate, not a parser.
//
// Usage:
//
//	go run ./vtparse --root ..
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"vitess.io/vitess/go/vt/sqlparser"
)

// sqlcConfig is the part of a service's sqlc.yaml this tool needs: which engine the queries target, and where the generated Go landed.
type sqlcConfig struct {
	SQL []struct {
		Engine string `yaml:"engine"`
		Gen    struct {
			Go struct {
				Out string `yaml:"out"`
			} `yaml:"go"`
		} `yaml:"gen"`
	} `yaml:"sql"`
}

type finding struct {
	pos   token.Position
	name  string
	query string
	err   error
}

func main() {
	root := flag.String("root", ".", "directory to scan")
	flag.Parse()

	findings, queries, err := run(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vtparse:", err)
		os.Exit(2)
	}

	if len(findings) == 0 {
		fmt.Printf("vtparse: %d generated MySQL queries, all parse on Vitess\n", queries)
		return
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].pos.String() < findings[j].pos.String() })

	fmt.Fprintf(os.Stderr, "vtparse: %d generated MySQL queries, %d that vtgate cannot parse\n\n", queries, len(findings))
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "  %s\n    %s: %s\n    %s\n\n", f.pos, f.name, oneLine(f.err.Error()), firstLine(f.query))
	}
	fmt.Fprintln(os.Stderr, "These are valid MySQL that PlanetScale rejects at parse time, so they fail on every execution in production while passing every test here.")
	os.Exit(1)
}

// run parses every generated query belonging to a MySQL service and returns the ones vtgate rejects, along with the total examined.
func run(root string) ([]finding, int, error) {
	dirs, err := generatedDirs(root)
	if err != nil {
		return nil, 0, err
	}
	if len(dirs) == 0 {
		return nil, 0, fmt.Errorf("no MySQL sqlc configs found under %s; is --root pointing at the repository?", root)
	}

	p, err := sqlparser.New(sqlparser.Options{})
	if err != nil {
		return nil, 0, fmt.Errorf("creating parser: %w", err)
	}

	fset := token.NewFileSet()
	var findings []finding
	var total int
	for _, dir := range dirs {
		paths, err := filepath.Glob(filepath.Join(dir, "*.sql.go"))
		if err != nil {
			return nil, 0, err
		}
		sort.Strings(paths)
		for _, path := range paths {
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return nil, 0, fmt.Errorf("parsing %s: %w", path, err)
			}
			for _, decl := range queryConsts(file) {
				total++
				if _, err := p.Parse(decl.query); err != nil {
					findings = append(findings, finding{
						pos:   fset.Position(decl.pos),
						name:  decl.name,
						query: decl.query,
						err:   err,
					})
				}
			}
		}
	}
	return findings, total, nil
}

// generatedDirs returns the output directory of every service whose sqlc.yaml declares engine: mysql. Reading the config rather than listing services is what keeps a Postgres service out without naming it: agent-service generates $1 placeholders, which this parser would reject for the wrong reason.
func generatedDirs(root string) ([]string, error) {
	configs, err := filepath.Glob(filepath.Join(root, "services", "*", "sqlc.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(configs)

	var dirs []string
	for _, path := range configs {
		body, err := os.ReadFile(path) // #nosec G304 -- globbed from the repository tree
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		var cfg sqlcConfig
		if err := yaml.Unmarshal(body, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		for _, block := range cfg.SQL {
			if block.Engine != "mysql" || block.Gen.Go.Out == "" {
				continue
			}
			dirs = append(dirs, filepath.Join(filepath.Dir(path), block.Gen.Go.Out))
		}
	}
	return dirs, nil
}

type queryConst struct {
	name  string
	query string
	pos   token.Pos
}

// queryConsts returns the query strings in one generated file. sqlc emits each as a package-level string const, which is the exact text handed to the driver.
func queryConsts(file *ast.File) []queryConst {
	var out []queryConst
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
				continue
			}
			lit, ok := value.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			query, err := strconv.Unquote(lit.Value)
			if err != nil || !looksLikeSQL(query) {
				continue
			}
			out = append(out, queryConst{name: value.Names[0].Name, query: query, pos: value.Pos()})
		}
	}
	return out
}

// looksLikeSQL keeps the walk to query consts. sqlc's generated files hold nothing else at package scope, but a const that is not a statement would be reported as a parse failure it has no business failing.
func looksLikeSQL(s string) bool {
	trimmed := strings.TrimSpace(stripLeadingComments(s))
	for _, verb := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "REPLACE", "WITH"} {
		if len(trimmed) >= len(verb) && strings.EqualFold(trimmed[:len(verb)], verb) {
			return true
		}
	}
	return false
}

// stripLeadingComments drops the `-- name: X :many` header sqlc puts above every query, so the verb below it is what gets inspected.
func stripLeadingComments(s string) string {
	for {
		s = strings.TrimLeft(s, " \t\r\n")
		if !strings.HasPrefix(s, "--") {
			return s
		}
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[idx+1:]
			continue
		}
		return ""
	}
}

// oneLine flattens the parser's multi-line error so each finding stays one indented block.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func firstLine(query string) string {
	line := strings.TrimSpace(stripLeadingComments(query))
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	if len(line) > 120 {
		line = line[:120] + "..."
	}
	return line
}
