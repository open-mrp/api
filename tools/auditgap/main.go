// Command auditgap fails when a gRPC handler can reach a sqlc write but never reaches audit.Publisher.Publish.
//
// It builds a VTA call graph over every service, takes each method a handler type serves for its embedded Unimplemented*Server as a root, and reports roots that reach a write-shaped sqlc query without reaching Publish. Handlers that write without auditing on purpose are listed, each with its reason, in allowlist.txt.
//
// Only gRPC handlers are checked: queue consumers dispatch through broker callbacks the call graph cannot follow.
//
// Usage:
//
//	go run ./auditgap --root ..
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var (
	writeVerb = regexp.MustCompile(`^(Create|Insert|Update|Upsert|Delete|Set|Mark|Archive|Unarchive|Bulk|Increment|Decrement|Replace|Clear|Remove|Add|Restore|Soft|Hard|Touch|Bump|Claim|Release|Record|Save|Purge|Reset|Link|Unlink|Enqueue|Close|Open|Cancel|Complete|Finish|Approve|Reject|Assign|Unassign|Apply|Lock|Unlock|Merge|Move|Revoke|Rotate|Expire)`)
	// Bookkeeping tables that are never user-visible records.
	bookkeeping = regexp.MustCompile(`(?i)outbox|idempot|requestlog|request_log|inbox|lease|recoverypoint|cache`)
)

type gap struct {
	handler string
	writes  []string
}

// allowlist maps an exempt handler (service/Type.Method) to the reason it writes without auditing.
type allowlist map[string]string

func main() {
	root := flag.String("root", ".", "module root to analyze")
	allowPath := flag.String("allowlist", filepath.Join("auditgap", "allowlist.txt"), "exemptions file")
	why := flag.String("why", "", "print one call path from this handler (Type.Method) to each write it reaches")
	flag.Parse()

	allow, err := readAllowlist(*allowPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "auditgap:", err)
		os.Exit(2)
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "auditgap:", err)
		os.Exit(2)
	}

	gaps, stale, handlers, err := check(absRoot, []string{"./services/..."}, allow, *why)
	if err != nil {
		fmt.Fprintln(os.Stderr, "auditgap:", err)
		os.Exit(2)
	}
	if len(gaps) == 0 && len(stale) == 0 {
		fmt.Printf("auditgap: %d gRPC handlers, every write audited or allowlisted\n", handlers)
		return
	}
	if len(gaps) > 0 {
		fmt.Fprintf(os.Stderr, "auditgap: %d gRPC handlers write without publishing an audit event\n\n", len(gaps))
		for _, g := range gaps {
			fmt.Fprintf(os.Stderr, "  %s\n    writes: %s\n", g.handler, strings.Join(g.writes, ", "))
		}
		fmt.Fprintln(os.Stderr, "\nPublish an audit event for the write, or add the handler to tools/auditgap/allowlist.txt with the reason it is exempt.")
	}
	if len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "\nauditgap: %d allowlist entries no longer match an unaudited write; remove them:\n", len(stale))
		for _, s := range stale {
			fmt.Fprintf(os.Stderr, "  %s\n", s)
		}
	}
	os.Exit(1)
}

// readAllowlist parses `<service>/<Type>.<Method>  # reason` lines; every entry needs a reason.
func readAllowlist(path string) (allowlist, error) {
	a := allowlist{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entry, reason, _ := strings.Cut(line, "#")
		fields := strings.Fields(entry)
		if len(fields) != 1 || strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("%s:%d: want `<service>/<Type>.<Method>  # reason`", path, n)
		}
		a[fields[0]] = strings.TrimSpace(reason)
	}
	return a, sc.Err()
}

func check(dir string, patterns []string, allow allowlist, why string) ([]gap, []string, int, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax | packages.NeedModule, Dir: dir}, patterns...)
	if err != nil {
		return nil, nil, 0, err
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return nil, nil, 0, fmt.Errorf("%d package errors", n)
	}
	module := ""
	for _, p := range pkgs {
		if p.Module != nil {
			module = p.Module.Path
			break
		}
	}
	if module == "" {
		return nil, nil, 0, fmt.Errorf("no module found under %s", dir)
	}
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()
	funcs := ssautil.AllFunctions(prog)
	cg := vta.CallGraph(funcs, cha.CallGraph(prog))
	cg.DeleteSyntheticNodes()

	used := map[string]bool{}

	var gaps []gap
	handlers := 0
	for fn := range funcs {
		key, ok := handlerKey(fn)
		if !ok {
			continue
		}
		node := cg.CreateNode(fn)
		handlers++
		writes, paths, audited := walk(node, module)
		if why != "" && strings.HasSuffix(key, why) {
			for _, w := range sortedKeys(paths) {
				fmt.Fprintf(os.Stderr, "%s -> %s:\n  %s\n", key, w, strings.Join(paths[w], "\n  "))
			}
		}
		if len(writes) == 0 || audited {
			continue
		}
		if _, ok := allow[key]; ok {
			used[key] = true
			continue
		}
		gaps = append(gaps, gap{handler: key, writes: writes})
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i].handler < gaps[j].handler })

	var stale []string
	for k := range allow {
		if !used[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(stale)
	return gaps, stale, handlers, nil
}

// handlerKey names a gRPC method implementation as service/Type.Method, or reports that fn is not one.
func handlerKey(fn *ssa.Function) (string, bool) {
	if fn == nil || fn.Pkg == nil || fn.Synthetic != "" || fn.Signature.Recv() == nil {
		return "", false
	}
	path := fn.Pkg.Pkg.Path()
	if !strings.Contains(path, "/infrastructure/grpc") {
		return "", false
	}
	if pos := fn.Prog.Fset.Position(fn.Pos()); strings.HasSuffix(pos.Filename, "_test.go") {
		return "", false
	}
	recv := fn.Signature.Recv().Type()
	if p, ok := recv.(*types.Pointer); ok {
		recv = p.Elem()
	}
	named, ok := recv.(*types.Named)
	if !ok {
		return "", false
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return "", false
	}
	served := false
	for f := range st.Fields() {
		if !f.Embedded() || !strings.HasPrefix(f.Name(), "Unimplemented") || !strings.HasSuffix(f.Name(), "Server") {
			continue
		}
		if obj, _, _ := types.LookupFieldOrMethod(f.Type(), true, f.Pkg(), fn.Name()); obj != nil {
			served = true
			break
		}
	}
	if !served {
		return "", false
	}
	return serviceName(path) + "/" + named.Obj().Name() + "." + fn.Name(), true
}

func serviceName(pkgPath string) string {
	_, rest, ok := strings.Cut(pkgPath, "/services/")
	if !ok {
		return pkgPath
	}
	name, _, _ := strings.Cut(rest, "/")
	return name
}

// walk lists the write-shaped sqlc methods node can reach, with one call path to each, and whether it reaches audit.Publisher.Publish.
//
// The walk stays inside the analyzed module (and errgroup): VTA merges every func value of one signature that flows through shared library code such as hook lists, so following it there would connect every handler to every closure of that shape.
func walk(node *callgraph.Node, module string) ([]string, map[string][]string, bool) {
	parent := map[*callgraph.Node]*callgraph.Node{node: nil}
	queue := []*callgraph.Node{node}
	paths := map[string][]string{}
	audited := false
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, e := range c.Out {
			callee := e.Callee
			if _, seen := parent[callee]; seen {
				continue
			}
			parent[callee] = c
			if isAuditPublish(callee.Func) {
				audited = true
				continue
			}
			if name, ok := sqlcWrite(callee.Func); ok {
				if _, have := paths[name]; !have {
					paths[name] = pathTo(parent, callee)
				}
				continue
			}
			if followed(callee.Func, module) {
				queue = append(queue, callee)
			}
		}
	}
	return sortedKeys(paths), paths, audited
}

func pkgPath(fn *ssa.Function) string {
	if fn == nil {
		return ""
	}
	if fn.Pkg == nil && fn.Origin() != nil {
		fn = fn.Origin()
	}
	if fn.Pkg == nil {
		return ""
	}
	return fn.Pkg.Pkg.Path()
}

func followed(fn *ssa.Function, module string) bool {
	path := pkgPath(fn)
	return path == "" || path == module || strings.HasPrefix(path, module+"/") || strings.HasPrefix(path, "golang.org/x/sync/")
}

func isAuditPublish(fn *ssa.Function) bool {
	return fn != nil && fn.Name() == "Publish" && strings.HasSuffix(pkgPath(fn), "shared/audit")
}

func sqlcWrite(fn *ssa.Function) (string, bool) {
	if fn == nil || !strings.HasSuffix(pkgPath(fn), "/infrastructure/sqlc") {
		return "", false
	}
	name := fn.Name()
	if !writeVerb.MatchString(name) || bookkeeping.MatchString(name) {
		return "", false
	}
	return name, true
}

func pathTo(parent map[*callgraph.Node]*callgraph.Node, n *callgraph.Node) []string {
	var path []string
	for ; n != nil; n = parent[n] {
		path = append(path, n.Func.String())
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
