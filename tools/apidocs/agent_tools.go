package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
)

// agentToolsGoPath is the generated Go catalog, relative to the project root the tool chdirs into. Endpoint-tools live entirely in code — there is intentionally no DB seed, so adding endpoints never needs a migration.
const agentToolsGoPath = "services/agent-service/internal/agents/endpoint_tools_gen.go"

// agentToolParam mirrors agents.EndpointToolParam in the generated Go output.
type agentToolParam struct {
	Name  string
	In    string // path | query | body
	Array bool
}

// agentToolDescriptor is the fully resolved description of one endpoint exposed as an agent tool.
type agentToolDescriptor struct {
	Slug                string
	DisplayName         string
	Description         string
	Method              string
	RouteTemplate       string
	InputSchema         json.RawMessage
	Params              []agentToolParam
	RequiredPermissions []string
	RequiredRoleType    string
	Group               string
}

// generateAgentTools collects every endpoint that is public or flagged AgentTool, builds a self-contained input schema per endpoint, and writes the Go catalog (no DB seed).
func generateAgentTools(groups []apiendpoint.APIEndpointGroup) error {
	descriptors := collectAgentToolDescriptors(groups)
	logInfof("Found %d agent-tool endpoints", len(descriptors))

	if err := writeAgentToolsGo(descriptors); err != nil {
		return fmt.Errorf("writing Go catalog: %w", err)
	}
	return nil
}

func collectAgentToolDescriptors(groups []apiendpoint.APIEndpointGroup) []agentToolDescriptor {
	docReader := NewDocReader()
	var out []agentToolDescriptor
	seen := map[string]bool{}

	for _, group := range groups {
		for _, e := range group.Endpoints {
			spec := endpointSpecField(e)
			// AgentTool is the explicit opt-in. Being public does NOT make an
			// endpoint a tool — each must set AgentTool: true.
			if !spec.FieldByName("AgentTool").Bool() {
				continue
			}

			title := strings.TrimSpace(spec.FieldByName("Title").String())
			method := strings.ToUpper(strings.TrimSpace(spec.FieldByName("Method").String()))
			route := strings.TrimSpace(spec.FieldByName("Route").String())
			if title == "" || method == "" || route == "" {
				panic(fmt.Errorf("agent-tool endpoint missing title/method/route: %q", title))
			}

			description := title
			if epTypeVal := spec.FieldByName("EndpointType"); epTypeVal.IsValid() && !epTypeVal.IsNil() {
				if doc := docReader.GetTypeDoc(epTypeVal.Interface().(reflect.Type)).Doc; doc != "" {
					description = doc
				}
			}

			slug := toSnakeSlug(title)
			if seen[slug] {
				panic(fmt.Errorf("duplicate agent-tool slug %q (from title %q)", slug, title))
			}
			seen[slug] = true

			schema, params := buildToolSchema(e, method, docReader)
			raw, err := json.Marshal(schema)
			if err != nil {
				panic(fmt.Errorf("marshaling input schema for %q: %w", slug, err))
			}
			// Tool-use models read JSON Schema, not OpenAPI: rewrite `nullable: true` into a
			// `"null"` member of the field's `type` so the model can clear a field by sending a
			// real JSON null instead of the string "null".
			raw, err = rewriteNullableToTypeUnion(raw)
			if err != nil {
				panic(fmt.Errorf("normalizing nullables for %q: %w", slug, err))
			}

			// Permissions are NOT derived — each endpoint declares exactly what
			// its services check (or nothing, for unprotected reference data).
			out = append(out, agentToolDescriptor{
				Slug:                slug,
				DisplayName:         title,
				Description:         strings.TrimSpace(description),
				Method:              method,
				RouteTemplate:       route,
				InputSchema:         raw,
				Params:              params,
				RequiredPermissions: explicitPermissions(spec),
				RequiredRoleType:    requiredRoleType(spec),
				Group:               groupName(deriveResource(route)),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

// deriveResource extracts the resource segment from a route: the last static segment before the first path parameter, or the last static segment when there is no parameter.
func deriveResource(route string) string {
	resource := ""
	for _, seg := range strings.Split(route, "/") {
		if strings.HasPrefix(seg, "{") {
			break
		}
		if seg == "" || seg == "v1" {
			continue
		}
		resource = seg
	}
	return resource
}

// explicitPermissions reads the endpoint's declared RequiredPermissions (the source of truth) and renders each typed {Domain, Action} as a "<domain>:<action>" string.
func explicitPermissions(spec reflect.Value) []string {
	f := spec.FieldByName("RequiredPermissions")
	if !f.IsValid() || f.Kind() != reflect.Slice {
		return nil
	}
	out := make([]string, 0, f.Len())
	for i := 0; i < f.Len(); i++ {
		el := f.Index(i)
		dom := strings.TrimSpace(el.FieldByName("Domain").String())
		act := strings.TrimSpace(el.FieldByName("Action").String())
		if dom != "" && act != "" {
			out = append(out, dom+":"+act)
		}
	}
	return out
}

// requiredRoleType reads the endpoint's declared RequiredRoleType (e.g. "admin"), empty when none.
func requiredRoleType(spec reflect.Value) string {
	f := spec.FieldByName("RequiredRoleType")
	if !f.IsValid() {
		return ""
	}
	return strings.TrimSpace(f.String())
}

// groupName produces the display group for the tool-selection UI from the route resource (e.g. "product-lines" -> "Product Lines").
func groupName(resource string) string {
	if resource == "" {
		return "API"
	}
	parts := strings.FieldsFunc(resource, func(r rune) bool { return r == '-' || r == '_' })
	for i, p := range parts {
		parts[i] = capitalizeASCII(p)
	}
	return strings.Join(parts, " ")
}

// rewriteNullableToTypeUnion converts OpenAPI-style `"nullable": true` into JSON-Schema-standard nullability (a `"null"` member of the `type`) everywhere in an agent tool's input schema. Tool-use models read JSON Schema, where `nullable` is not a keyword and is silently ignored — so a clearable field shown as `{"type":"string","nullable":true}` reads as string-only, and the model "clears" it by sending the text "null" rather than a JSON null. Emitting `{"type":["string","null"]}` makes the null option explicit so the model sends a real null. A nullable field with no declared scalar type is left unchanged (the generator always sets one for the body fields this matters for).
func rewriteNullableToTypeUnion(raw json.RawMessage) (json.RawMessage, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	walkNullableToTypeUnion(root)
	return json.Marshal(root)
}

func walkNullableToTypeUnion(node any) {
	switch n := node.(type) {
	case map[string]any:
		if nullable, _ := n["nullable"].(bool); nullable {
			delete(n, "nullable")
			switch t := n["type"].(type) {
			case string:
				n["type"] = []any{t, "null"}
			case []any:
				if !containsString(t, "null") {
					n["type"] = append(t, "null")
				}
			}
		}
		for _, v := range n {
			walkNullableToTypeUnion(v)
		}
	case []any:
		for _, v := range n {
			walkNullableToTypeUnion(v)
		}
	}
}

func containsString(xs []any, want string) bool {
	for _, x := range xs {
		if s, ok := x.(string); ok && s == want {
			return true
		}
	}
	return false
}

// buildToolSchema produces a self-contained JSON-Schema object flattening the endpoint's body, path, and query inputs into a single object, plus the param-location map the runtime executor needs. Header inputs (auth/version) are excluded — the gateway client supplies those.
func buildToolSchema(e apiendpoint.APIEndpointer, method string, docReader *DocReader) (Schema, []agentToolParam) {
	reqType := e.GetRequestType()
	if reqType.Kind() == reflect.Pointer {
		reqType = reqType.Elem()
	}

	comps := &Components{Schemas: map[string]Schema{}}
	props := map[string]Schema{}
	var required []string
	var params []agentToolParam

	// Body: the json-tagged fields. generateSchema already applies all field-tag
	// semantics (enum, field.Optional/Clearable, required) for json fields.
	if method != http.MethodGet && method != http.MethodDelete && endpointRequestHasJSONFields(reqType) {
		body := simplifySchema(generateSchema(reqType, comps, docReader), comps, map[string]bool{})
		for name, s := range body.Properties {
			props[name] = s
			params = append(params, agentToolParam{Name: name, In: "body"})
		}
		required = append(required, body.Required...)
	}

	// Path and query params are not json-tagged, so they are handled per-field like
	// the OpenAPI parameter loop does.
	if reqType.Kind() == reflect.Struct {
		for _, f := range flattenStructFields(reqType) {
			if p := f.Tag.Get("path"); p != "" {
				fs := simplifySchema(generateSchema(f.Type, comps, docReader), comps, map[string]bool{})
				fs.Description = firstNonEmpty(getFieldDoc(reqType, f, docReader), fs.Description)
				props[p] = fs
				required = append(required, p)
				params = append(params, agentToolParam{Name: p, In: "path"})
				continue
			}
			if q := f.Tag.Get("query"); q != "" {
				fs := simplifySchema(generateSchema(f.Type, comps, docReader), comps, map[string]bool{})
				fs.Description = firstNonEmpty(getFieldDoc(reqType, f, docReader), fs.Description)
				props[q] = fs
				if strings.Contains(f.Tag.Get("validate"), "required") {
					required = append(required, q)
				}
				params = append(params, agentToolParam{Name: q, In: "query", Array: fs.Type == "array"})
			}
		}
	}

	// Expandable sub-objects: the endpoint's IncludeConfig declares which nested objects can be
	// expanded via the `include` query param (they come back null otherwise). Surface it to the agent
	// with its enumerated keys and a description, so the model knows expansion is possible and which
	// fields are valid — mirroring the OpenAPI generator, which adds the same parameter.
	if keys := endpointIncludeKeys(e); len(keys) > 0 {
		enumValues := make([]any, len(keys))
		for i, k := range keys {
			enumValues[i] = k
		}
		props["include"] = Schema{
			Type:        "array",
			Items:       &Schema{Type: "string", Enum: enumValues},
			Description: "Sub-objects to expand in the response. These nested objects are returned as null by default; pass the field keys you need (e.g. \"parent_account\") to get their full objects inline. Expand to get authoritative data rather than inferring relationships from names.",
		}
		// Replace any include param the request struct already declared, else append.
		found := false
		for i := range params {
			if params[i].Name == "include" {
				params[i] = agentToolParam{Name: "include", In: "query", Array: true}
				found = true
				break
			}
		}
		if !found {
			params = append(params, agentToolParam{Name: "include", In: "query", Array: true})
		}
	}

	// Sort params for deterministic generated output; body params would otherwise
	// follow map-iteration order.
	sort.Slice(params, func(i, j int) bool { return params[i].Name < params[j].Name })

	schema := Schema{Type: "object", Properties: props, Required: required}
	return schema, params
}

// endpointIncludeKeys returns the expandable include keys declared on an endpoint's IncludeConfig, or
// nil when it has none. IncludeConfig is a field on the concrete *APIEndpoint, not on the APIEndpointer
// interface, so it's read by reflection — the same way the OpenAPI generator reaches it.
func endpointIncludeKeys(e apiendpoint.APIEndpointer) []string {
	ev := reflect.ValueOf(e)
	if ev.Kind() == reflect.Pointer {
		ev = ev.Elem()
	}
	if ev.Kind() != reflect.Struct {
		return nil
	}
	f := ev.FieldByName("IncludeConfig")
	if !f.IsValid() || f.IsNil() {
		return nil
	}
	cfg, ok := f.Interface().(*apiendpoint.IncludeConfig)
	if !ok || cfg == nil {
		return nil
	}
	return cfg.AllowedKeys()
}

// simplifySchema turns an OpenAPI-tuned schema (with $ref into components and allOf wrappers) into a self-contained schema suitable for an LLM tool: refs are inlined, allOf is flattened into properties, and doc-only noise (examples, x-* extensions, readOnly) is stripped.
func simplifySchema(s Schema, comps *Components, seen map[string]bool) Schema {
	if s.Ref != "" {
		name := refLastSegment(s.Ref)
		if seen[name] {
			return Schema{Type: "object"}
		}
		seen[name] = true
		resolved := simplifySchema(comps.Schemas[name], comps, seen)
		seen[name] = false
		return resolved
	}

	if s.Items != nil {
		it := simplifySchema(*s.Items, comps, seen)
		s.Items = &it
	}
	if s.AdditionalProperties != nil {
		ap := simplifySchema(*s.AdditionalProperties, comps, seen)
		s.AdditionalProperties = &ap
	}
	if len(s.Properties) > 0 {
		np := make(map[string]Schema, len(s.Properties))
		for k, v := range s.Properties {
			np[k] = simplifySchema(v, comps, seen)
		}
		s.Properties = np
	}

	// Flatten allOf (used to attach a description sibling to a $ref, and to model
	// embedded structs) by merging each member's properties into this object.
	if len(s.AllOf) > 0 {
		if s.Properties == nil {
			s.Properties = map[string]Schema{}
		}
		for _, m := range s.AllOf {
			rm := simplifySchema(m, comps, seen)
			for k, v := range rm.Properties {
				if _, ok := s.Properties[k]; !ok {
					s.Properties[k] = v
				}
			}
			s.Required = append(s.Required, rm.Required...)
			if s.Type == "" {
				s.Type = rm.Type
			}
		}
		s.AllOf = nil
		if s.Type == "" {
			s.Type = "object"
		}
	}

	for i := range s.OneOf {
		s.OneOf[i] = simplifySchema(s.OneOf[i], comps, seen)
	}
	for i := range s.AnyOf {
		s.AnyOf[i] = simplifySchema(s.AnyOf[i], comps, seen)
	}

	// Strip documentation-only noise that would bloat the tool schema.
	s.Example = nil
	s.PropertyOrder = nil
	s.XStainlessEmptyObject = false
	s.XStainlessPaginationProperty = nil
	s.XExpandable = false
	s.ReadOnly = false

	return s
}

func capitalizeASCII(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// toSnakeSlug converts an endpoint Title ("List Sales Orders") into a stable tool slug ("list_sales_orders").
func toSnakeSlug(title string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func writeAgentToolsGo(descriptors []agentToolDescriptor) error {
	var b strings.Builder
	b.WriteString("// Code generated by tools/apidocs --agent-tools. DO NOT EDIT.\n\n")
	b.WriteString("package agents\n\n")
	b.WriteString("// EndpointTools is the generated catalog of api-gateway endpoints exposed as agent tools (endpoints flagged AgentTool=true). Regenerate with `make generate` or `make gen-agent-tools`.\n")
	b.WriteString("var EndpointTools = []EndpointToolDescriptor{\n")
	for _, d := range descriptors {
		b.WriteString("\t{\n")
		fmt.Fprintf(&b, "\t\tSlug:          %q,\n", d.Slug)
		fmt.Fprintf(&b, "\t\tDisplayName:   %q,\n", d.DisplayName)
		fmt.Fprintf(&b, "\t\tDescription:   %q,\n", d.Description)
		fmt.Fprintf(&b, "\t\tMethod:        %q,\n", d.Method)
		fmt.Fprintf(&b, "\t\tRouteTemplate: %q,\n", d.RouteTemplate)
		fmt.Fprintf(&b, "\t\tInputSchema:   %q,\n", string(d.InputSchema))
		fmt.Fprintf(&b, "\t\tGroup:         %q,\n", d.Group)
		if d.RequiredRoleType != "" {
			fmt.Fprintf(&b, "\t\tRequiredRoleType: %q,\n", d.RequiredRoleType)
		}
		if len(d.RequiredPermissions) > 0 {
			quoted := make([]string, len(d.RequiredPermissions))
			for i, p := range d.RequiredPermissions {
				quoted[i] = fmt.Sprintf("%q", p)
			}
			fmt.Fprintf(&b, "\t\tRequiredPermissions: []string{%s},\n", strings.Join(quoted, ", "))
		}
		if len(d.Params) > 0 {
			b.WriteString("\t\tParams: []EndpointToolParam{\n")
			for _, p := range d.Params {
				loc := "EndpointToolParam" + capitalizeASCII(p.In)
				if p.Array {
					fmt.Fprintf(&b, "\t\t\t{Name: %q, In: %s, Array: true},\n", p.Name, loc)
				} else {
					fmt.Fprintf(&b, "\t\t\t{Name: %q, In: %s},\n", p.Name, loc)
				}
			}
			b.WriteString("\t\t},\n")
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")

	// Run gofmt so the output is canonical. The hand-written padding above only
	// approximates alignment; gofmt re-aligns each struct literal's columns to the
	// widest field present, which varies per descriptor. Without this the committed
	// (gofmt'd) file and a fresh generation never match, so `make generate` is not
	// idempotent.
	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("gofmt generated catalog: %w", err)
	}

	return os.WriteFile(agentToolsGoPath, formatted, 0o644)
}
