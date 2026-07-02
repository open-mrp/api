package agents

import (
	"encoding/json"
	"testing"

	"github.com/augno/api/shared/constants"
)

// TestBuiltinTools_HaveRegisteredHandlers ensures every built-in tool in the code catalog has a runtime handler registered in RegisterTools. Without this, an agent could be granted a tool slug that the runner cannot execute.
func TestBuiltinTools_HaveRegisteredHandlers(t *testing.T) {
	registry := NewToolHandlerRegistry()
	RegisterTools(registry)

	for _, d := range BuiltinTools {
		if _, ok := registry.Get(string(d.Slug)); !ok {
			t.Errorf("built-in tool %q has no registered handler", d.Slug)
		}
	}
}

// TestBuiltinTools_CatalogIntegrity checks each descriptor is internally consistent: unique slug, valid JSON input schema, and a lookup that round-trips.
func TestBuiltinTools_CatalogIntegrity(t *testing.T) {
	seen := map[constants.Tool]bool{}
	for _, d := range BuiltinTools {
		if d.Slug == "" {
			t.Error("built-in tool with empty slug")
		}
		if seen[d.Slug] {
			t.Errorf("duplicate built-in tool slug %q", d.Slug)
		}
		seen[d.Slug] = true

		if !json.Valid([]byte(d.InputSchema)) {
			t.Errorf("built-in tool %q has invalid JSON input schema", d.Slug)
		}
		if d.Group.ID == "" {
			t.Errorf("built-in tool %q has no group", d.Slug)
		}

		got, ok := LookupBuiltinTool(string(d.Slug))
		if !ok || got.Slug != d.Slug {
			t.Errorf("LookupBuiltinTool(%q) did not round-trip", d.Slug)
		}
	}
}
