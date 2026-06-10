// Command publicendpoints generates a machine-readable inventory of all
// gateway routes marked Public: true (i.e. included in the consumer-facing
// specs/public_openapi_spec.json).
//
// It scans the APIEndpoint literals in services/api-gateway/endpoints and
// writes a TSV with one row per public route: HTTP method, route template,
// and the gateway endpoint file that declares it. The inventory is the
// starting point for the Public-route audit described in
// docs/patterns/authentication-patterns.md.
//
// Usage (from the tools module, as documented):
//
//	cd tools && go run ./publicendpoints --out ../specs/public_route_inventory.tsv
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const endpointsDir = "services/api-gateway/endpoints"

var (
	methodRe     = regexp.MustCompile(`Method:\s*(http\.Method[A-Za-z]+|"[A-Z]+")`)
	routeRe      = regexp.MustCompile(`Route:\s*"([^"]+)"`)
	routeIdentRe = regexp.MustCompile(`Route:\s*([A-Za-z_][A-Za-z0-9_]*)`)
	routeConstRe = regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"(/[^"]*)"`)
	publicRe     = regexp.MustCompile(`Public:\s*(true|false)`)
)

var httpMethodNames = map[string]string{
	"http.MethodGet":     "GET",
	"http.MethodHead":    "HEAD",
	"http.MethodPost":    "POST",
	"http.MethodPut":     "PUT",
	"http.MethodPatch":   "PATCH",
	"http.MethodDelete":  "DELETE",
	"http.MethodConnect": "CONNECT",
	"http.MethodOptions": "OPTIONS",
	"http.MethodTrace":   "TRACE",
}

type publicRoute struct {
	Method string
	Route  string
	File   string
}

// errBadFlags signals a flag-parse failure; the FlagSet already printed the
// message and usage to stderr, so main exits nonzero without re-printing.
var errBadFlags = errors.New("invalid command-line flags")

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errBadFlags) {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(1)
	}
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "..", "repository root (containing services/)")
	out := flags.String("out", "../specs/public_route_inventory.tsv", "output TSV path")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return errBadFlags
	}

	routes, err := collectPublicRoutes(*root)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("method\troute\tfile\n")
	for _, r := range routes {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", r.Method, r.Route, r.File)
	}

	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "wrote %d public routes to %s\n", len(routes), *out)
	return nil
}

func collectPublicRoutes(root string) ([]publicRoute, error) {
	dir := filepath.Join(root, endpointsDir)

	var routes []publicRoute
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}

		for _, chunk := range strings.Split(string(content), "apiendpoint.APIEndpoint[")[1:] {
			public := publicRe.FindStringSubmatch(chunk)
			if public == nil || public[1] != "true" {
				continue
			}
			method := methodRe.FindStringSubmatch(chunk)
			route := resolveRoute(chunk, filepath.Dir(path))
			if method == nil || route == "" {
				return fmt.Errorf("%s: could not extract method/route for a Public: true endpoint", rel)
			}
			routes = append(routes, publicRoute{
				Method: methodName(method[1]),
				Route:  route,
				File:   filepath.ToSlash(rel),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Route != routes[j].Route {
			return routes[i].Route < routes[j].Route
		}
		return routes[i].Method < routes[j].Method
	})
	return routes, nil
}

func methodName(token string) string {
	if name, ok := httpMethodNames[token]; ok {
		return name
	}
	return strings.Trim(token, `"`)
}

// resolveRoute extracts the route template from an endpoint literal. Routes
// are usually string literals, but some packages declare them as package-level
// string constants (e.g. endpoints/properties/routes.go).
func resolveRoute(chunk, pkgDir string) string {
	if m := routeRe.FindStringSubmatch(chunk); m != nil {
		return m[1]
	}
	if m := routeIdentRe.FindStringSubmatch(chunk); m != nil {
		return packageRouteConsts(pkgDir)[m[1]]
	}
	return ""
}

var routeConstCache = map[string]map[string]string{}

// packageRouteConsts returns the package-level `Name = "/..."` string
// constants declared in pkgDir, keyed by identifier.
func packageRouteConsts(pkgDir string) map[string]string {
	if cached, ok := routeConstCache[pkgDir]; ok {
		return cached
	}

	consts := map[string]string{}
	entries, err := os.ReadDir(pkgDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			content, readErr := os.ReadFile(filepath.Join(pkgDir, entry.Name()))
			if readErr != nil {
				continue
			}
			for _, m := range routeConstRe.FindAllStringSubmatch(string(content), -1) {
				consts[m[1]] = m[2]
			}
		}
	}

	routeConstCache[pkgDir] = consts
	return consts
}
