package main

import "testing"

// TestEndpointRoutesAreUnique guards against two endpoints claiming the same method and route. The router
// registers both without complaint and serves only one, and the spec keeps only one operation, so the
// other endpoint silently disappears from the API and the SDK.
func TestEndpointRoutesAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, group := range openAPIEndpointGroups() {
		for _, e := range group.Endpoints {
			key := e.GetMethod() + " " + e.GetRoute()
			if other, ok := seen[key]; ok {
				t.Errorf("%s is claimed by two endpoints, in the %q and %q groups", key, other, group.Title)
				continue
			}
			seen[key] = group.Title
		}
	}
	if len(seen) == 0 {
		t.Fatal("no endpoints found")
	}
}
