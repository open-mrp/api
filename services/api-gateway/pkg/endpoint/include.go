package apiendpoint

import "github.com/augno/api/shared/constants"

// IncludeField describes a single expandable sub-object on an API response.
type IncludeField struct {
	// Key is the client-facing include value (e.g. "role", "actor.role").
	Key string
	// ObjectType is the object type used in the collapsed reference.
	ObjectType constants.ObjectType
	// JSONPaths are the dot-separated paths where this sub-object appears in the
	// serialized response (e.g. "role", "api_key_info.role").
	JSONPaths []string
}

// IncludeConfig declares which sub-objects on an endpoint can be expanded via
// the include query parameter.
type IncludeConfig struct {
	Fields []IncludeField
	// ExtractRoots overrides the default root extraction for non-standard
	// response shapes (e.g. map-typed responses). When nil, the reflective
	// default handles *Resource and *List[Resource] shapes.
	ExtractRoots func(any) []any
}

// AllowedKeys returns the set of valid include parameter values.
func (c *IncludeConfig) AllowedKeys() []string {
	keys := make([]string, len(c.Fields))
	for i, f := range c.Fields {
		keys[i] = f.Key
	}
	return keys
}

// FieldsByKey returns a map from client-facing key to IncludeField.
func (c *IncludeConfig) FieldsByKey() map[string]IncludeField {
	m := make(map[string]IncludeField, len(c.Fields))
	for _, f := range c.Fields {
		m[f.Key] = f
	}
	return m
}
