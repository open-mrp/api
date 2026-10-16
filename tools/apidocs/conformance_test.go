package main

import (
	"bufio"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/open-mrp/apikit/conformance"
	"github.com/open-mrp/apikit/openapi"
)

const conformanceBaselinePath = "testdata/conformance_baseline.txt"

func init() {
	// Create Integration "Connects a provider to your account."
	conformance.DocVerbs = append(conformance.DocVerbs, "Connects")
}

// TestConformance holds every endpoint to every forge.1 rule. Endpoints that broke a rule before the forge.1 sweep are
// listed in testdata/conformance_baseline.txt as "rule METHOD route"; the unit that fixes one deletes its line. A new
// violation fails, and so does a baseline line that no longer fails, so the list only shrinks.
//
// Regenerate the file with UPDATE_CONFORMANCE_BASELINE=1 only when deliberately accepting a violation.
func TestConformance(t *testing.T) {
	t.Parallel()
	got := map[string][]string{}
	for _, v := range conformance.Check(openAPIEndpointGroups(), append(slices.Clone(conformance.Rules), searchDocumented)...) {
		key := v.Rule + " " + v.Method + " " + v.Route
		got[key] = append(got[key], v.Message)
	}

	if os.Getenv("UPDATE_CONFORMANCE_BASELINE") == "1" {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		if err := os.WriteFile(conformanceBaselinePath, []byte(strings.Join(keys, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}

	baseline := readConformanceBaseline(t)
	for key, messages := range got {
		if !baseline[key] {
			for _, m := range messages {
				t.Errorf("%s: %s", key, m)
			}
		}
	}
	for key := range baseline {
		if _, still := got[key]; !still {
			t.Errorf("%s now conforms; delete its line from %s", key, conformanceBaselinePath)
		}
	}
}

var searchDocs = openapi.NewDocReader()

// searchDocumented holds every `q` to documenting exactly which columns it matches and how, such as "Case-insensitive substring match on `name`." Never "varies by endpoint".
var searchDocumented = conformance.Rule{
	Name: "search_documented",
	Check: func(e conformance.Endpoint) []string {
		if e.RequestType == nil || e.RequestType.Kind() != reflect.Struct {
			return nil
		}
		for _, f := range reflect.VisibleFields(e.RequestType) {
			if f.Tag.Get("query") != "q" {
				continue
			}
			declaring := e.RequestType
			for _, i := range f.Index[:len(f.Index)-1] {
				declaring = declaring.Field(i).Type
				if declaring.Kind() == reflect.Pointer {
					declaring = declaring.Elem()
				}
			}
			doc := searchDocs.GetTypeDoc(declaring).Fields[f.Name]
			if strings.Contains(doc, "varies by endpoint") || !strings.Contains(strings.ToLower(doc), "match") {
				return []string{"q must say which columns it matches and how, such as \"Case-insensitive substring match on `name`.\""}
			}
		}
		return nil
	},
}

func readConformanceBaseline(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open(conformanceBaselinePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			out[line] = true
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
