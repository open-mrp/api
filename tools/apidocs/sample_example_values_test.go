package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/version"
	"github.com/stretchr/testify/require"
)

// TestSampleExampleValues walks every example against the schema it illustrates
// and fails on a blank value the schema says can never be blank: a required
// property or an enum property. These come from sample structs assembled as
// partial inline literals — an unset enum or required string marshals to "" and
// ships as documentation.
func TestSampleExampleValues(t *testing.T) {
	groups := openAPIEndpointGroups()

	for _, tc := range []struct {
		name       string
		publicOnly bool
	}{
		{"public", true},
		{"internal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			built, err := buildOpenAPIDocument(groups, tc.publicOnly, nil, version.Latest.Version)
			require.NoError(t, err)

			// The document is assembled from ordered maps; one JSON round-trip
			// gives the walk plain maps all the way down.
			doc, ok := jsonNormalizeExample(built).(map[string]any)
			require.True(t, ok)

			v := &exampleValidator{resolver: &schemaResolver{schemas: componentSchemas(doc)}}

			for name, raw := range v.resolver.schemas {
				sch, isMap := raw.(map[string]any)
				if !isMap {
					continue
				}
				if ex, has := sch["example"]; has {
					v.walk(ex, sch, name, name)
				}
			}
			walkOperationExamples(doc, v.resolver, func(ctx string, example, schema any) {
				v.walk(example, schema, ctx, ctx)
			})

			for _, f := range v.sorted() {
				t.Errorf("%s is blank in its example but the schema declares it %s — give the sample a real value (seen at %s)", f.owner, f.reason, f.origin)
			}
		})
	}
}

// arrayIndex collapses concrete list indices so one bad sample repeated down a
// list example is reported once, not once per element.
var arrayIndex = regexp.MustCompile(`\[\d+\]`)

type blankExampleField struct {
	// owner is the path rerooted at the nearest named schema, so the same
	// offending sample reached through many resources reports as one finding.
	owner  string
	origin string
	reason string
}

type exampleValidator struct {
	resolver *schemaResolver
	blank    map[string]blankExampleField
}

func (v *exampleValidator) record(owner, origin, reason string) {
	if v.blank == nil {
		v.blank = map[string]blankExampleField{}
	}
	owner = arrayIndex.ReplaceAllString(owner, "[]")
	if _, ok := v.blank[owner]; ok {
		return
	}
	v.blank[owner] = blankExampleField{
		owner:  owner,
		origin: arrayIndex.ReplaceAllString(origin, "[]"),
		reason: reason,
	}
}

func (v *exampleValidator) sorted() []blankExampleField {
	out := make([]blankExampleField, 0, len(v.blank))
	for _, f := range v.blank {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].owner < out[j].owner })
	return out
}

func (v *exampleValidator) walk(example, schema any, owner, origin string) {
	sch, name := v.resolver.resolve(schema)
	if sch == nil {
		return
	}
	if name != "" {
		owner = name
	}

	switch ex := example.(type) {
	case map[string]any:
		props, _ := sch["properties"].(map[string]any)
		required := stringSet(sch["required"])
		for key, val := range ex {
			propSchema, defined := props[key]
			if !defined {
				addl, isSchema := sch["additionalProperties"].(map[string]any)
				if !isSchema {
					continue
				}
				propSchema = addl
			}
			if s, isString := val.(string); isString && strings.TrimSpace(s) == "" {
				resolved, _ := v.resolver.resolve(propSchema)
				switch {
				case resolved != nil && resolved["enum"] != nil:
					v.record(owner+"."+key, origin+"."+key, "an enum")
				case required[key]:
					v.record(owner+"."+key, origin+"."+key, "required")
				}
				continue
			}
			v.walk(val, propSchema, owner+"."+key, origin+"."+key)
		}
	case []any:
		items := sch["items"]
		for i, val := range ex {
			v.walk(val, items, fmt.Sprintf("%s[%d]", owner, i), fmt.Sprintf("%s[%d]", origin, i))
		}
	}
}

type schemaResolver struct {
	schemas map[string]any
}

// resolve flattens $ref and allOf so a property's enum and required facts are
// visible where the example is checked, and reports the component name a $ref
// landed on. oneOf/anyOf are left alone: a blank value may be valid under one
// branch, so they are not worth guessing at.
func (r *schemaResolver) resolve(schema any) (resolved map[string]any, component string) {
	return r.resolveWithin(schema, nil)
}

func (r *schemaResolver) resolveWithin(schema any, seen []string) (map[string]any, string) {
	sch, ok := schema.(map[string]any)
	if !ok {
		return nil, ""
	}

	if ref, isRef := sch["$ref"].(string); isRef {
		name := ref[strings.LastIndex(ref, "/")+1:]
		for _, s := range seen {
			if s == name {
				return nil, ""
			}
		}
		inner, deeper := r.resolveWithin(r.schemas[name], append(seen, name))
		if deeper != "" {
			name = deeper
		}
		return inner, name
	}

	if allOf, isAllOf := sch["allOf"].([]any); isAllOf {
		merged := map[string]any{}
		props := map[string]any{}
		var required []any
		for _, sub := range allOf {
			part, _ := r.resolveWithin(sub, seen)
			for k, val := range part {
				switch k {
				case "properties":
					if p, isMap := val.(map[string]any); isMap {
						for pk, pv := range p {
							props[pk] = pv
						}
					}
				case "required":
					if req, isSlice := val.([]any); isSlice {
						required = append(required, req...)
					}
				default:
					if _, exists := merged[k]; !exists {
						merged[k] = val
					}
				}
			}
		}
		merged["properties"] = props
		merged["required"] = required
		return merged, ""
	}

	return sch, ""
}

func componentSchemas(doc map[string]any) map[string]any {
	components, ok := doc["components"].(map[string]any)
	if !ok {
		return nil
	}
	schemas, _ := components["schemas"].(map[string]any)
	return schemas
}

// walkOperationExamples visits request and success-response examples the
// component-schema pass does not already cover — an operation whose media
// schema resolves to a component carrying its own example is skipped, so one
// bad sample is not reported once per route that returns the resource.
func walkOperationExamples(doc map[string]any, r *schemaResolver, fn func(ctx string, example, schema any)) {
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		return
	}
	for route, methods := range paths {
		mm, isMap := methods.(map[string]any)
		if !isMap {
			continue
		}
		for method, opAny := range mm {
			op, isOp := opAny.(map[string]any)
			if !isOp {
				continue
			}

			media := map[string]map[string]any{}
			if rb, has := op["requestBody"].(map[string]any); has {
				if m, found := jsonMediaObject(rb); found {
					media["request body"] = m
				}
			}
			responses, _ := op["responses"].(map[string]any)
			for code, respAny := range responses {
				resp, isResp := respAny.(map[string]any)
				if !isResp || len(code) == 0 || code[0] != '2' {
					continue
				}
				if m, found := jsonMediaObject(resp); found {
					media[code+" response"] = m
				}
			}

			for label, m := range media {
				example, has := m["example"]
				if !has || example == nil {
					continue
				}
				if sch, _ := r.resolve(m["schema"]); sch != nil {
					if _, covered := sch["example"]; covered {
						continue
					}
				}
				fn(strings.ToUpper(method)+" "+route+" "+label, example, m["schema"])
			}
		}
	}
}

func jsonMediaObject(holder map[string]any) (map[string]any, bool) {
	content, ok := holder["content"].(map[string]any)
	if !ok {
		return nil, false
	}
	media, ok := content["application/json"].(map[string]any)
	return media, ok
}

func stringSet(raw any) map[string]bool {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make(map[string]bool, len(arr))
	for _, v := range arr {
		if s, isString := v.(string); isString {
			out[s] = true
		}
	}
	return out
}
