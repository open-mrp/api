// Command txaudit checks that database transaction callbacks are safe to run more than once.
//
// TransactionManager re-runs a transaction that the database rolled back as a deadlock victim.
// That is only correct while a callback's effects are confined to the database, because those
// are the effects the rollback undoes. Anything that escapes — an HTTP call to a payment
// provider, a message published straight to the broker, a value appended to a slice declared
// outside the callback — happens again on the retry, and nothing undid the first one.
//
// This tool reads every closure passed to WithTx, withTx, or WithTxSavepoint and reports the
// escaping effects it finds. It deliberately only looks at what a closure does directly: a
// name-based walk into everything a closure calls collapses on method names like Create and
// Publish, which appear on both repositories and external clients, and a check nobody can trust
// is a check nobody runs.
//
// Usage:
//
//	go run ./txaudit --root ..
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// externalReceivers name the fields whose methods leave the database. A call on one of these
// inside a transaction is not undone by a rollback, so it must not be there — the established
// pattern is to record the intent in the outbox and act on it after the commit.
var externalReceivers = []string{
	"stripeClient", "shippoClient", "hubspotClient", "vercelClient", "mapsClient",
	"s3Client", "s3Store", "objectStore", "emailClient", "notificationClient",
	"billingClient", "coreClient", "authClient", "platformClient", "agentClient",
	"addressValidator", "portalDomainProvider", "httpClient", "checkoutClient",
	"rabbitmq", "broker", "publisherClient",
}

type finding struct {
	pos    token.Position
	kind   string
	detail string
	why    string
}

func main() {
	root := flag.String("root", ".", "directory to scan")
	flag.Parse()

	findings, closures, err := audit(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "txaudit:", err)
		os.Exit(2)
	}

	if len(findings) == 0 {
		fmt.Printf("txaudit: %d transaction callbacks, all confined to the database\n", closures)
		return
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].pos.String() < findings[j].pos.String() })

	fmt.Fprintf(os.Stderr, "txaudit: %d transaction callbacks, %d with effects a rollback would not undo\n\n", closures, len(findings))
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "  %s\n    %s: %s\n    %s\n\n", f.pos, f.kind, f.detail, f.why)
	}
	os.Exit(1)
}

func audit(root string) ([]finding, int, error) {
	fset := token.NewFileSet()
	var findings []finding
	closures := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", ".git", "mock":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			// A file this tool cannot parse is one the compiler will reject anyway.
			return nil
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "WithTx", "withTx", "WithTxSavepoint":
			default:
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			lit, ok := call.Args[len(call.Args)-1].(*ast.FuncLit)
			if !ok {
				// A named function rather than a literal: the transaction managers' own
				// plumbing does this, and there is no closure to inspect.
				return true
			}

			closures++
			findings = append(findings, inspect(fset, lit)...)
			return true
		})
		return nil
	})

	return findings, closures, err
}

// inspect reports the effects in one transaction callback that a rollback would not undo.
func inspect(fset *token.FileSet, lit *ast.FuncLit) []finding {
	local := declaredInside(lit)

	var out []finding
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		switch d := n.(type) {

		case *ast.AssignStmt:
			// `xs = append(xs, ...)` where xs outlives the callback: a retry appends a second
			// time, and the caller is handed both runs' worth of results.
			if d.Tok != token.ASSIGN || len(d.Lhs) != 1 || len(d.Rhs) != 1 {
				return true
			}
			id, ok := d.Lhs[0].(*ast.Ident)
			if !ok || id.Name == "_" || local[id.Name] {
				return true
			}
			call, ok := d.Rhs[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "append" {
				out = append(out, finding{
					pos:    fset.Position(d.Pos()),
					kind:   "appends to a variable declared outside the callback",
					detail: id.Name,
					why:    "build the slice inside the callback and assign it out once, so a retry replaces the result instead of doubling it",
				})
			}

		case *ast.CallExpr:
			sel, ok := d.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			recv := receiverText(sel.X)
			for _, name := range externalReceivers {
				if strings.Contains(recv, name) {
					out = append(out, finding{
						pos:    fset.Position(d.Pos()),
						kind:   "calls outside the database",
						detail: recv + "." + sel.Sel.Name,
						why:    "record the intent in the outbox and act on it after the transaction commits",
					})
					break
				}
			}

		case *ast.GoStmt:
			out = append(out, finding{
				pos:    fset.Position(d.Pos()),
				kind:   "starts a goroutine",
				detail: "go ...",
				why:    "the goroutine outlives the transaction and runs again on a retry; start it after the commit",
			})

		case *ast.SendStmt:
			out = append(out, finding{
				pos:    fset.Position(d.Pos()),
				kind:   "sends on a channel",
				detail: "ch <- ...",
				why:    "the receiver has already acted on the first send by the time the transaction rolls back",
			})
		}
		return true
	})
	return out
}

// declaredInside collects the names a callback introduces itself. Those are re-initialized on
// every attempt, so writing to them is harmless; only writes that escape survive a retry.
func declaredInside(lit *ast.FuncLit) map[string]bool {
	local := map[string]bool{}

	for _, p := range lit.Type.Params.List {
		for _, n := range p.Names {
			local[n.Name] = true
		}
	}

	ast.Inspect(lit.Body, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.AssignStmt:
			if d.Tok == token.DEFINE {
				for _, lhs := range d.Lhs {
					if id, ok := lhs.(*ast.Ident); ok {
						local[id.Name] = true
					}
				}
			}
		case *ast.DeclStmt:
			gd, ok := d.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, id := range vs.Names {
						local[id.Name] = true
					}
				}
			}
		case *ast.RangeStmt:
			for _, e := range []ast.Expr{d.Key, d.Value} {
				if id, ok := e.(*ast.Ident); ok {
					local[id.Name] = true
				}
			}
		}
		return true
	})

	return local
}

func receiverText(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return receiverText(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		return receiverText(v.Fun)
	}
	return ""
}
