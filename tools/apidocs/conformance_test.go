package main

import (
	"bufio"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/open-mrp/apikit/conformance"
)

const conformanceBaselinePath = "testdata/conformance_baseline.txt"

// TestConformance holds every endpoint to every forge.1 rule. Endpoints that broke a rule before the forge.1 sweep are
// listed in testdata/conformance_baseline.txt as "rule METHOD route"; the unit that fixes one deletes its line. A new
// violation fails, and so does a baseline line that no longer fails, so the list only shrinks.
//
// Regenerate the file with UPDATE_CONFORMANCE_BASELINE=1 only when deliberately accepting a violation.
func TestConformance(t *testing.T) {
	t.Parallel()
	got := map[string][]string{}
	for _, v := range conformance.Check(openAPIEndpointGroups()) {
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
