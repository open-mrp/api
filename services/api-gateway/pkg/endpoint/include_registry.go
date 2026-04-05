package apiendpoint

import (
	"fmt"
	"strings"

	"github.com/augno/api/shared/constants"
)

// IncludeFieldDef defines a single expandable field on a resource type.
type IncludeFieldDef struct {
	// Key is the client-facing key, matches JSON field name (e.g., "role").
	Key string
	// ObjectType is the type of the expanded sub-object.
	ObjectType constants.ObjectType
	// JSONPath overrides the JSON path if different from Key.
	JSONPath string
	// Children defines nested expandable fields inline, bypassing registry lookup.
	// Use when the nested type doesn't have its own ObjectType (e.g., RequestLogActor).
	// When nil, the system resolves children from the registry using ObjectType.
	Children []IncludeFieldDef
}

// ObjectIncludes defines the expandable fields for a single resource type.
type ObjectIncludes struct {
	ObjectType constants.ObjectType
	Fields     []IncludeFieldDef
}

var registry = map[constants.ObjectType]*ObjectIncludes{}

// RegisterIncludes registers the expandable fields for an object type.
// Called during init().
func RegisterIncludes(oi *ObjectIncludes) {
	if _, exists := registry[oi.ObjectType]; exists {
		panic(fmt.Sprintf("include_registry: duplicate registration for %s", oi.ObjectType))
	}
	registry[oi.ObjectType] = oi
}

// GetObjectIncludes returns the registered includes for an object type, or nil.
func GetObjectIncludes(ot constants.ObjectType) *ObjectIncludes {
	return registry[ot]
}

// IncludesParams configures which includes an endpoint exposes.
type IncludesParams struct {
	// ObjectType is the root resource type returned by this endpoint.
	ObjectType constants.ObjectType
	// Fields is the explicit whitelist of include keys to expose.
	// Supports dot-notation (e.g., "actor.role") for nested includes.
	// REQUIRED — the function panics if empty.
	Fields []string
	// DefaultFields lists include keys that are always expanded, even when the
	// client does not send an include parameter. Each key must also appear in Fields.
	DefaultFields []string
	// PathPrefix prepends a JSON path prefix for wrapper types
	// (e.g., "api_key_info" when the APIKey is nested inside CreatedAPIKey).
	PathPrefix string
}

// IncludesFor resolves an IncludeConfig from the registry, exposing ONLY
// the explicitly listed fields. Panics at startup if a field key doesn't
// match a registered include path.
func IncludesFor(p IncludesParams) *IncludeConfig {
	if len(p.Fields) == 0 {
		panic(fmt.Sprintf("include_registry: IncludesFor(%s) called with empty Fields", p.ObjectType))
	}

	oi := registry[p.ObjectType]
	if oi == nil {
		panic(fmt.Sprintf("include_registry: no registration for %s", p.ObjectType))
	}

	// Build the full set of resolvable includes by walking the registry graph.
	resolvable := map[string]IncludeField{}
	visited := map[constants.ObjectType]bool{p.ObjectType: true}
	walkFields("", "", oi.Fields, resolvable, visited)

	// Filter to only the whitelisted keys.
	var fields []IncludeField
	for _, key := range p.Fields {
		field, ok := resolvable[key]
		if !ok {
			panic(fmt.Sprintf("include_registry: field %q not resolvable for %s (available: %s)",
				key, p.ObjectType, resolvableKeys(resolvable)))
		}
		if p.PathPrefix != "" {
			for i, path := range field.JSONPaths {
				field.JSONPaths[i] = p.PathPrefix + "." + path
			}
		}
		fields = append(fields, field)
	}

	// Validate that all default fields are in the allowed set.
	allowedKeys := make(map[string]bool, len(p.Fields))
	for _, k := range p.Fields {
		allowedKeys[k] = true
	}
	for _, dk := range p.DefaultFields {
		if !allowedKeys[dk] {
			panic(fmt.Sprintf("include_registry: default field %q not in Fields for %s", dk, p.ObjectType))
		}
	}

	return &IncludeConfig{Fields: fields, DefaultFields: p.DefaultFields}
}

// walkFields recursively builds the full set of resolvable include fields.
// visited tracks ObjectTypes already expanded to prevent infinite recursion
// (e.g., AgentDefinition.config → AgentDefinition).
func walkFields(keyPrefix, pathPrefix string, defs []IncludeFieldDef, out map[string]IncludeField, visited map[constants.ObjectType]bool) {
	for _, def := range defs {
		key := def.Key
		if keyPrefix != "" {
			key = keyPrefix + "." + def.Key
		}

		jsonPath := def.Key
		if def.JSONPath != "" {
			jsonPath = def.JSONPath
		}
		if pathPrefix != "" {
			jsonPath = pathPrefix + "." + jsonPath
		}

		out[key] = IncludeField{
			Key:        key,
			ObjectType: def.ObjectType,
			JSONPaths:  []string{jsonPath},
		}

		// Resolve children: use inline Children if provided, otherwise look up registry.
		var children []IncludeFieldDef
		if def.Children != nil {
			children = def.Children
		} else if !visited[def.ObjectType] {
			if childOI := registry[def.ObjectType]; childOI != nil {
				visited[def.ObjectType] = true
				children = childOI.Fields
			}
		}

		if len(children) > 0 {
			walkFields(key, jsonPath, children, out, visited)
		}
	}
}

func resolvableKeys(m map[string]IncludeField) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
