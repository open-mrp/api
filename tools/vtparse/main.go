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
	_ "embed"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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
	updateBaseline := flag.Bool("update-baseline", false, "rewrite or-sentinel-baseline.txt from the current corpus")
	flag.Parse()

	result, err := run(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vtparse:", err)
		os.Exit(2)
	}

	if *updateBaseline {
		if err := writeBaseline(result.sentinels); err != nil {
			fmt.Fprintln(os.Stderr, "vtparse:", err)
			os.Exit(2)
		}
		fmt.Printf("vtparse: baseline updated — %d queries carrying OR-sentinel filters\n", len(result.sentinels))
		return
	}

	baseline, err := parseBaseline(baselineFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vtparse:", err)
		os.Exit(2)
	}
	regressions := sentinelRegressions(result.sentinels, baseline)

	findings, queries := result.findings, result.total

	if len(findings) == 0 && len(regressions) == 0 {
		fmt.Printf("vtparse: %d generated MySQL queries, all parse on Vitess\n", queries)
		fmt.Printf("vtparse: %d queries carry OR-sentinel filters, all baselined\n", len(result.sentinels))
		return
	}

	if len(regressions) > 0 {
		fmt.Fprintf(os.Stderr, "vtparse: %d query/queries gained an OR-sentinel filter\n\n", len(regressions))
		for _, r := range regressions {
			fmt.Fprintf(os.Stderr, "  %s\n    %s: %d sentinel(s), baseline has %d\n", r.file, r.name, r.found, r.baseline)
		}
		fmt.Fprintln(os.Stderr, "\n`? = false OR ...` and `? IS NULL OR ...` are not sargable, and they cost more than the predicate they guard: the optimizer abandons the keyset composite for the whole query. Build the SQL in Go so an absent filter emits no predicate — see inventory_change_log_list_query.go. If the query was rewritten to remove sentinels, refresh with --update-baseline.")
		if len(findings) == 0 {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr)
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].pos.String() < findings[j].pos.String() })

	fmt.Fprintf(os.Stderr, "vtparse: %d generated MySQL queries, %d that vtgate cannot parse\n\n", queries, len(findings))
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "  %s\n    %s: %s\n    %s\n\n", f.pos, f.name, oneLine(f.err.Error()), firstLine(f.query))
	}
	fmt.Fprintln(os.Stderr, "These are valid MySQL that PlanetScale rejects at parse time, so they fail on every execution in production while passing every test here.")
	os.Exit(1)
}

// scanResult is one pass over the corpus: what vtgate cannot parse, and what carries OR-sentinel filters.
type scanResult struct {
	findings  []finding
	total     int
	sentinels map[string]int // "<path relative to root>:<const name>" -> sentinel count
}

// sentinelRe matches the two shapes sqlc reaches for when a filter is optional. Both compare a bind parameter to a constant, so the predicate is decided before any row is read and no index can serve it.
var sentinelRe = regexp.MustCompile(`(?is)\?\s*(?:=\s*(?:false|true|0|1)|IS\s+NULL)\s+OR\b`)

// run parses every generated query belonging to a MySQL service and returns the ones vtgate rejects, along with the total examined.
func run(root string) (scanResult, error) {
	dirs, err := generatedDirs(root)
	if err != nil {
		return scanResult{}, err
	}
	if len(dirs) == 0 {
		return scanResult{}, fmt.Errorf("no MySQL sqlc configs found under %s; is --root pointing at the repository?", root)
	}

	p, err := sqlparser.New(sqlparser.Options{})
	if err != nil {
		return scanResult{}, fmt.Errorf("creating parser: %w", err)
	}

	fset := token.NewFileSet()
	result := scanResult{sentinels: map[string]int{}}
	for _, dir := range dirs {
		paths, err := filepath.Glob(filepath.Join(dir, "*.sql.go"))
		if err != nil {
			return scanResult{}, err
		}
		sort.Strings(paths)
		for _, path := range paths {
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return scanResult{}, fmt.Errorf("parsing %s: %w", path, err)
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			rel = filepath.ToSlash(rel)
			for _, decl := range queryConsts(file) {
				result.total++
				if _, err := p.Parse(decl.query); err != nil {
					result.findings = append(result.findings, finding{
						pos:   fset.Position(decl.pos),
						name:  decl.name,
						query: decl.query,
						err:   err,
					})
				}
				if n := len(sentinelRe.FindAllString(decl.query, -1)); n > 0 {
					result.sentinels[rel+":"+decl.name] = n
				}
			}
		}
	}
	return result, nil
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

// baselineFile records the OR-sentinel filters already in the corpus, as "<path>:<query> <count>".
//
// Embedded rather than read from disk so the check does not depend on the working directory it is invoked from, and checked in so that adding a sentinel shows up as a reviewable line in the diff rather than a silently rising number.
//
//go:embed or-sentinel-baseline.txt
var baselineFile string

// baselinePath is where --update-baseline writes, relative to the tools module the Makefile runs this from.
const baselinePath = "vtparse/or-sentinel-baseline.txt"

type regression struct {
	file     string
	name     string
	found    int
	baseline int
}

// sentinelRegressions reports queries carrying more sentinels than the baseline allows. A query with fewer is not reported: removing them is the point, and failing the build for it would punish the fix.
func sentinelRegressions(found, baseline map[string]int) []regression {
	var out []regression
	for key, n := range found {
		if n <= baseline[key] {
			continue
		}
		file, name, _ := strings.Cut(key, ":")
		out = append(out, regression{file: file, name: name, found: n, baseline: baseline[key]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].file != out[j].file {
			return out[i].file < out[j].file
		}
		return out[i].name < out[j].name
	})
	return out
}

func parseBaseline(body string) (map[string]int, error) {
	baseline := map[string]int{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, count, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("malformed baseline line %q; want \"<path>:<query> <count>\"", line)
		}
		n, err := strconv.Atoi(count)
		if err != nil {
			return nil, fmt.Errorf("malformed count in baseline line %q: %w", line, err)
		}
		baseline[key] = n
	}
	return baseline, nil
}

func writeBaseline(found map[string]int) error {
	keys := make([]string, 0, len(found))
	for k := range found {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("# OR-sentinel filters already in the corpus, written by `go run ./vtparse --root .. --update-baseline`.\n")
	b.WriteString("# Each is an optional filter sqlc could not express conditionally, and each costs the query its keyset composite.\n")
	b.WriteString("# Lines only ever come off this list. A new one fails the build; see the vtparse package doc.\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "%s %d\n", k, found[k])
	}
	return os.WriteFile(baselinePath, []byte(b.String()), 0o600)
}
