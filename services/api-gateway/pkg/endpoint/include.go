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
	// DefaultFields lists include keys that are always expanded, even when the
	// client does not send an include parameter. When the client explicitly
	// requests includes, defaults are merged in automatically.
	DefaultFields []string
}

// AllowedKeys returns the set of valid include parameter values.
func (c *IncludeConfig) AllowedKeys() []string {
	keys := make([]string, len(c.Fields))
	for i, f := range c.Fields {
		keys[i] = f.Key
	}
	return keys
}

// DefaultFieldSet returns the set of default include keys as a map.
func (c *IncludeConfig) DefaultFieldSet() map[string]bool {
	if len(c.DefaultFields) == 0 {
		return nil
	}
	m := make(map[string]bool, len(c.DefaultFields))
	for _, k := range c.DefaultFields {
		m[k] = true
	}
	return m
}

// FieldsByKey returns a map from client-facing key to IncludeField.
func (c *IncludeConfig) FieldsByKey() map[string]IncludeField {
	m := make(map[string]IncludeField, len(c.Fields))
	for _, f := range c.Fields {
		m[f.Key] = f
	}
	return m
}
