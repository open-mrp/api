package main

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func findDescriptor(t *testing.T, descriptors []agentToolDescriptor, slug string) agentToolDescriptor {
	t.Helper()
	for _, d := range descriptors {
		if d.Slug == slug {
			return d
		}
	}
	t.Fatalf("no agent-tool descriptor with slug %q (got %d descriptors)", slug, len(descriptors))
	return agentToolDescriptor{}
}

func paramByName(d agentToolDescriptor, name string) (agentToolParam, bool) {
	for _, p := range d.Params {
		if p.Name == name {
			return p, true
		}
	}
	return agentToolParam{}, false
}

// TestAgentToolDescriptors verifies that endpoints flagged AgentTool=true are turned into tool descriptors with self-contained schemas and a correct param-location map.
func TestAgentToolDescriptors(t *testing.T) {
	descriptors := collectAgentToolDescriptors(openAPIEndpointGroups())
	if len(descriptors) == 0 {
		t.Fatal("expected at least one agent-tool descriptor")
	}

	// Schemas handed to the LLM must be self-contained: no component refs or
	// OpenAPI-only noise that the model cannot resolve.
	for _, d := range descriptors {
		s := string(d.InputSchema)
		for _, banned := range []string{"$ref", "allOf", "x-stainless", "x-expandable"} {
			if strings.Contains(s, banned) {
				t.Errorf("%s: input schema must not contain %q (not self-contained):\n%s", d.Slug, banned, s)
			}
		}
		var obj map[string]any
		if err := json.Unmarshal(d.InputSchema, &obj); err != nil {
			t.Errorf("%s: input schema is not valid JSON: %v", d.Slug, err)
			continue
		}
		if obj["type"] != "object" {
			t.Errorf("%s: input schema root type = %v, want object", d.Slug, obj["type"])
		}
	}

	t.Run("create_sales_order body params", func(t *testing.T) {
		d := findDescriptor(t, descriptors, "create_sales_order")
		if d.Method != "POST" {
			t.Errorf("method = %s, want POST", d.Method)
		}
		p, ok := paramByName(d, "buyer_account_id")
		if !ok {
			t.Fatal("missing buyer_account_id param")
		}
		if p.In != "body" {
			t.Errorf("buyer_account_id In = %s, want body", p.In)
		}

		// Required body fields must surface in the schema's required list.
		var obj struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(d.InputSchema, &obj); err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(obj.Required, "buyer_account_id") {
			t.Errorf("required = %v, want it to include buyer_account_id", obj.Required)
		}
	})

	t.Run("list_sales_orders query params", func(t *testing.T) {
		d := findDescriptor(t, descriptors, "list_sales_orders")
		if d.Method != "GET" {
			t.Errorf("method = %s, want GET", d.Method)
		}
		p, ok := paramByName(d, "status_codes")
		if !ok {
			t.Fatal("missing status_codes param")
		}
		if p.In != "query" {
			t.Errorf("status_codes In = %s, want query", p.In)
		}
		if !p.Array {
			t.Error("status_codes should be flagged Array")
		}
	})
}

func TestToSnakeSlug(t *testing.T) {
	cases := map[string]string{
		"List Sales Orders":  "list_sales_orders",
		"Create Sales Order": "create_sales_order",
		"  Spaced  Title  ":  "spaced_title",
	}
	for in, want := range cases {
		if got := toSnakeSlug(in); got != want {
			t.Errorf("toSnakeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEndpointToolSchemasAreEmbedded(t *testing.T) {
	descriptors := collectAgentToolDescriptors(openAPIEndpointGroups())
	for _, d := range descriptors {
		if len(d.InputSchema) == 0 {
			t.Errorf("%s: InputSchema must be embedded in the catalog (no DB seed exists)", d.Slug)
		}
	}
}

func TestRewriteNullableToTypeUnion(t *testing.T) {
	in := json.RawMessage(`{
		"type":"object",
		"properties":{
			"note":{"type":"string","nullable":true},
			"name":{"type":"string"},
			"credit_limit":{"type":"object","nullable":true,"properties":{"value":{"type":"string"}}},
			"tags":{"type":"array","nullable":true,"items":{"type":"string"}},
			"already":{"type":["string","null"],"nullable":true}
		}
	}`)
	out, err := rewriteNullableToTypeUnion(in)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Properties map[string]struct {
			Type     any   `json:"type"`
			Nullable *bool `json:"nullable"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	for name, p := range got.Properties {
		if p.Nullable != nil {
			t.Errorf("%s: nullable keyword should be removed, still present", name)
		}
	}
	assertTypeUnion := func(name string, want string) {
		got, ok := got.Properties[name].Type.([]any)
		if !ok {
			t.Errorf("%s: type should be an array, got %#v", name, got)
			return
		}
		if len(got) != 2 || got[0] != want || got[1] != "null" {
			t.Errorf("%s: type = %v, want [%q null]", name, got, want)
		}
	}
	assertTypeUnion("note", "string")
	assertTypeUnion("credit_limit", "object")
	assertTypeUnion("tags", "array")
	// A non-nullable field keeps its scalar type.
	if got.Properties["name"].Type != "string" {
		t.Errorf("name: type = %#v, want scalar \"string\"", got.Properties["name"].Type)
	}
	// An already-unioned type does not get a duplicate "null".
	if arr, ok := got.Properties["already"].Type.([]any); !ok || len(arr) != 2 {
		t.Errorf("already: type = %#v, want no duplicate null", got.Properties["already"].Type)
	}
}

// TestAgentToolNullableFieldsUseTypeUnion guards the end-to-end output: a real clearable field comes
// through as a JSON-Schema null union (so models send a real null to clear it), never OpenAPI `nullable`.
func TestAgentToolNullableFieldsUseTypeUnion(t *testing.T) {
	descriptors := collectAgentToolDescriptors(openAPIEndpointGroups())
	for _, d := range descriptors {
		if strings.Contains(string(d.InputSchema), `"nullable"`) {
			t.Errorf("%s: InputSchema still contains OpenAPI \"nullable\" — models ignore it; use a \"null\" type member", d.Slug)
		}
	}

	d := findDescriptor(t, descriptors, "update_customer")
	var schema struct {
		Properties map[string]struct {
			Type any `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(d.InputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	billTo, ok := schema.Properties["bill_to_address_id"]
	if !ok {
		t.Fatal("update_customer should have a bill_to_address_id property")
	}
	arr, ok := billTo.Type.([]any)
	if !ok || !slices.Contains(arr, any("null")) {
		t.Errorf("bill_to_address_id type = %#v, want a union containing \"null\"", billTo.Type)
	}
}
