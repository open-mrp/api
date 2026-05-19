package main

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

// TestEndpointGroupParity guards against silent drift between the runtime
// router (services/api-gateway/internal/router/init_groups.go) and the
// OpenAPI spec generator (tools/apidocs/endpoint_groups.go). Both files maintain a
// list of httpgroup.*EndpointGroup constructions, and any group that the
// router serves but the generator omits will silently disappear from the
// public/internal OpenAPI spec — and therefore from the generated SDK and
// the public docs.
//
// If this test fails, add the missing group to whichever file is missing
// it. The two files don't share a list yet because the router is wired
// for real gRPC clients while the generator uses dummy ones, but they
// must reference the same set of groups.
func TestEndpointGroupParity(t *testing.T) {
	const (
		routerFile  = "../../services/api-gateway/internal/router/init_groups.go"
		apidocsFile = "endpoint_groups.go"
	)

	routerGroups, err := extractEndpointGroups(routerFile)
	if err != nil {
		t.Fatalf("read router file: %v", err)
	}
	apidocsGroups, err := extractEndpointGroups(apidocsFile)
	if err != nil {
		t.Fatalf("read apidocs file: %v", err)
	}

	if len(routerGroups) == 0 {
		t.Fatalf("no groups discovered in %s — extractor regex may be broken", routerFile)
	}
	if len(apidocsGroups) == 0 {
		t.Fatalf("no groups discovered in %s — extractor regex may be broken", apidocsFile)
	}

	missingFromApidocs := setDiff(routerGroups, apidocsGroups)
	missingFromRouter := setDiff(apidocsGroups, routerGroups)

	if len(missingFromApidocs) > 0 {
		t.Errorf("groups registered in router but missing from OpenAPI spec generator (tools/apidocs/endpoint_groups.go): %v\n"+
			"Add them to tools/apidocs/endpoint_groups.go so they appear in the generated spec.", missingFromApidocs)
	}
	if len(missingFromRouter) > 0 {
		t.Errorf("groups in OpenAPI spec generator (tools/apidocs/endpoint_groups.go) but not registered in router: %v\n"+
			"Either remove them from tools/apidocs/endpoint_groups.go or register them in services/api-gateway/internal/router/init_groups.go.", missingFromRouter)
	}
}

var endpointGroupRe = regexp.MustCompile(`httpgroup\.([A-Za-z0-9_]+EndpointGroup)\{\}`)

// extractEndpointGroups returns the set of httpgroup.*EndpointGroup names
// referenced as struct literals (i.e. constructed) in the given Go file.
func extractEndpointGroups(path string) (map[string]struct{}, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	matches := endpointGroupRe.FindAllStringSubmatch(string(src), -1)
	out := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		out[m[1]] = struct{}{}
	}
	return out, nil
}

// setDiff returns elements present in a but not in b, sorted for stable test output.
func setDiff(a, b map[string]struct{}) []string {
	var out []string
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
