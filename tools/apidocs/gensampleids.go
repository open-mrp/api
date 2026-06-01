//go:build ignore

package main

// Run from api root:
//
//	go run ./tools/apidocs/gensampleids.go           # print name, old, new
//	go run ./tools/apidocs/gensampleids.go -apply   # replace in docs + api-gateway tree

import (
	"crypto/sha256"
	"encoding/hex"
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
)

// Nano segment after prefix: 01 or 02 followed by 22+ lowercase alnum (matches IDLength12 tails).
var augNanoSuffix = regexp.MustCompile(`^(.+)_(0[12][0-9a-z]{22,})$`)

var skipNames = map[string]struct{}{
	"SampleCheckoutSessionID": {},
	"SampleStripeCustomerID":  {},
}

func main() {
	pairs := collectReplacementPairs()
	if len(os.Args) > 1 && os.Args[1] == "-apply" {
		applyReplacements(pairs)
		return
	}
	for _, p := range pairs {
		fmt.Printf("%s\t%s\t%s\n", p.name, p.old, p.new)
	}
}

type replPair struct {
	name, old, new string
}

func collectReplacementPairs() []replPair {
	root := filepath.Join("services", "api-gateway", "pkg", "resource")
	entries, err := os.ReadDir(root)
	if err != nil {
		panic(err)
	}
	var names []string
	oldByName := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		p := filepath.Join(root, e.Name())
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			gs, ok := n.(*ast.GenDecl)
			if !ok || gs.Tok != token.CONST {
				return true
			}
			for _, spec := range gs.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
					continue
				}
				name := vs.Names[0].Name
				if !includeConstName(name) {
					continue
				}
				bl, ok := vs.Values[0].(*ast.BasicLit)
				if !ok || bl.Kind != token.STRING {
					continue
				}
				s, err := strconv.Unquote(bl.Value)
				if err != nil {
					continue
				}
				if _, skip := skipNames[name]; skip {
					fmt.Fprintf(os.Stderr, "skip (non-augno): %s %q\n", name, s)
					continue
				}
				if _, ok := splitPrefixNano(s); !ok {
					fmt.Fprintf(os.Stderr, "skip (unparsed id): %s %q\n", name, s)
					continue
				}
				names = append(names, name)
				oldByName[name] = s
			}
			return true
		})
	}

	sort.Strings(names)

	usedNanos := map[string]string{}
	nameToNano := map[string]string{}

	for attempt := 0; ; attempt++ {
		nameToNano = map[string]string{}
		usedNanos = map[string]string{}
		collision := false
		for _, name := range names {
			nano := newNano(name, attempt)
			if prev, ok := usedNanos[nano]; ok && prev != name {
				collision = true
				break
			}
			usedNanos[nano] = name
			nameToNano[name] = nano
		}
		if !collision {
			break
		}
		if attempt > 10000 {
			panic("could not resolve nano collisions")
		}
	}

	var pairs []replPair
	for _, n := range names {
		old := oldByName[n]
		prefix, ok := splitPrefixNano(old)
		if !ok {
			panic("split failed: " + n)
		}
		newVal := prefix + "_" + nameToNano[n]
		if old == newVal {
			fmt.Fprintf(os.Stderr, "warning: %s unchanged\n", n)
		}
		pairs = append(pairs, replPair{name: n, old: old, new: newVal})
	}

	oldSeen := map[string][]string{}
	for _, p := range pairs {
		oldSeen[p.old] = append(oldSeen[p.old], p.name)
	}
	for old, names := range oldSeen {
		if len(names) < 2 {
			continue
		}
		panic(fmt.Sprintf("duplicate old sample id %q from %v — give each const a distinct literal before -apply", old, names))
	}

	return pairs
}

func includeConstName(name string) bool {
	if !strings.HasPrefix(name, "Sample") {
		return false
	}
	if strings.HasSuffix(name, "ID") {
		return true
	}
	return strings.HasPrefix(name, "SamplePlanTypeID")
}

func newNano(name string, attempt int) string {
	salt := fmt.Sprintf("augno:sample-id:%s", name)
	if attempt > 0 {
		salt = fmt.Sprintf("%s:%d", salt, attempt)
	}
	h := sha256.Sum256([]byte(salt))
	s := hex.EncodeToString(h[:])
	out := make([]byte, 26)
	out[0] = '0'
	out[1] = '1'
	for i := 0; i < 24; i++ {
		out[i+2] = s[i]
	}
	return string(out)
}

func splitPrefixNano(s string) (string, bool) {
	m := augNanoSuffix.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func applyReplacements(pairs []replPair) {
	// Longest first so we never split a longer ID by a shorter replacement (unlikely).
	sort.Slice(pairs, func(i, j int) bool { return len(pairs[i].old) > len(pairs[j].old) })

	roots := []string{
		"services/api-gateway",
		"tools/apidocs",
		"docs",
	}
	exts := map[string]struct{}{
		".go": {}, ".md": {}, ".json": {}, ".yml": {}, ".yaml": {},
	}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "vendor" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if _, ok := exts[filepath.Ext(path)]; !ok {
				return nil
			}
			if filepath.Base(path) == "gensampleids.go" {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			s := string(b)
			orig := s
			for _, p := range pairs {
				if !strings.Contains(s, p.old) {
					continue
				}
				s = strings.ReplaceAll(s, p.old, p.new)
			}
			if s != orig {
				if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
					return err
				}
				fmt.Println(path)
			}
			return nil
		})
	}
}
