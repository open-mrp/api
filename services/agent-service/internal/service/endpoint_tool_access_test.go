package service

import (
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/agents"
)

func TestEndpointToolCatalogInfos(t *testing.T) {
	tools, groups := endpointToolCatalogInfos()

	if len(tools) != len(agents.EndpointTools) {
		t.Fatalf("got %d tool infos, want %d", len(tools), len(agents.EndpointTools))
	}

	groupIDs := map[string]bool{}
	for _, g := range groups {
		groupIDs[g.ID] = true
		if g.Name == "" || g.Slug == "" {
			t.Errorf("group %q has empty name/slug", g.ID)
		}
	}

	var sawListCustomers, sawPermissioned bool
	for _, tl := range tools {
		if tl.Category != "api_endpoint" {
			t.Errorf("%s: category = %q, want api_endpoint", tl.Slug, tl.Category)
		}
		if tl.GroupID == "" || !groupIDs[tl.GroupID] {
			t.Errorf("%s: GroupID %q has no matching group", tl.Slug, tl.GroupID)
		}
		// Permissions are declared per endpoint; unprotected reference endpoints
		// (e.g. enum reads) legitimately have none, so we don't require all tools
		// to carry them — only that protected ones do.
		if len(tl.RequiredPermissions) > 0 {
			sawPermissioned = true
		}
		if tl.Slug == "list_customers" {
			sawListCustomers = true
			if tl.GroupName != "Customers" {
				t.Errorf("list_customers group = %q, want Customers", tl.GroupName)
			}
			if len(tl.RequiredPermissions) != 1 || tl.RequiredPermissions[0] != "customers:read" {
				t.Errorf("list_customers permissions = %v, want [customers:read]", tl.RequiredPermissions)
			}
		}
	}
	if !sawListCustomers {
		t.Error("expected list_customers in the endpoint-tool catalog")
	}
	if !sawPermissioned {
		t.Error("expected most endpoint-tools to carry declared permissions")
	}
}

func TestResolveAllowedEndpointTools(t *testing.T) {
	// Wildcard grants the whole catalog.
	all := resolveAllowedEndpointTools([]string{"*"})
	if len(all) != len(agents.EndpointTools) {
		t.Errorf("wildcard granted %d, want %d (full catalog)", len(all), len(agents.EndpointTools))
	}

	// Explicit slugs grant exactly those; unknown slugs are dropped.
	got := resolveAllowedEndpointTools([]string{"list_customers", "not_a_real_tool"})
	if !got["list_customers"] {
		t.Error("list_customers should be granted")
	}
	if got["not_a_real_tool"] {
		t.Error("unknown slug must not be granted")
	}
	if len(got) != 1 {
		t.Errorf("expected exactly 1 grant, got %d", len(got))
	}

	// No grant => empty set.
	if len(resolveAllowedEndpointTools(nil)) != 0 {
		t.Error("nil grant must resolve to empty set")
	}
}

func TestBuildAgentToolDefsGatesSearchTool(t *testing.T) {
	reg := agents.NewToolHandlerRegistry()
	agents.RegisterTools(reg)
	s := &runnerSvc{toolRegistry: reg}

	// No endpoint grant: the search meta-tool is NOT exposed, and no endpoint
	// tools are injected up front.
	defs, _ := s.buildAgentToolDefs(nil, map[string]bool{}, nil, false)
	for _, d := range defs {
		if d.Name == agents.SearchAPIToolsSlug {
			t.Error("search_api_tools must not be exposed without a grant")
		}
	}

	// With a grant: the search meta-tool IS exposed, but the endpoint-tools
	// themselves are still not injected (progressive disclosure).
	allowed := map[string]bool{"list_customers": true, "create_customer": true}
	// The agent's per-agent override gates create_customer for review but leaves list_customers open.
	endpointReview := map[string]bool{"create_customer": true}
	defs, review := s.buildAgentToolDefs(nil, allowed, endpointReview, false)
	hasSearch := false
	for _, d := range defs {
		if d.Name == agents.SearchAPIToolsSlug {
			hasSearch = true
		}
		if d.Name == "list_customers" || d.Name == "create_customer" {
			t.Errorf("endpoint-tool %q must not be injected up front", d.Name)
		}
	}
	if !hasSearch {
		t.Error("search_api_tools must be exposed when the agent has a grant")
	}

	// Review is driven by the per-agent override, not the HTTP method: the override
	// gates create_customer, while list_customers (absent from the override) defaults off.
	if !review["create_customer"] {
		t.Error("create_customer should require review when the override marks it true")
	}
	if review["list_customers"] {
		t.Error("list_customers should default to no review when absent from the override")
	}
}

// TestBuildAgentToolDefsExposesFindAppPageOnChatRuns: linking a page is part of writing a chat reply, so
// the tool is added to every chat run rather than granted per agent — an existing agent has no grant for
// it and would otherwise never be able to link a page.
func TestBuildAgentToolDefsExposesFindAppPageOnChatRuns(t *testing.T) {
	reg := agents.NewToolHandlerRegistry()
	agents.RegisterTools(reg)
	s := &runnerSvc{toolRegistry: reg}
	slug := agents.FindAppPageSlug

	defs, _ := s.buildAgentToolDefs(nil, map[string]bool{}, nil, false)
	for _, d := range defs {
		if d.Name == slug {
			t.Error("find_app_page must not be exposed on a non-chat run")
		}
	}

	defs, review := s.buildAgentToolDefs(nil, map[string]bool{}, nil, true)
	found := false
	for _, d := range defs {
		if d.Name == slug {
			found = true
			if len(d.InputSchema) == 0 {
				t.Error("find_app_page exposed without an input schema")
			}
		}
	}
	if !found {
		t.Error("find_app_page must be exposed on a chat run, with no grant needed")
	}
	if review[slug] {
		t.Error("a read-only page lookup must not require approval")
	}
}
