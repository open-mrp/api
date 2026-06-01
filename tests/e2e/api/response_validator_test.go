//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	objectSchemaIndexOnce sync.Once
	objectSchemaIndex     map[string][]*openAPISchema
	objectSchemaIndexErr  error
)

// AssertResponseBodyValid asserts that a JSON response body conforms to the
// OpenAPI schema identified by each object's "object" discriminator, recursively.
func AssertResponseBodyValid(t *testing.T, body []byte) {
	t.Helper()

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		assert.Fail(t, fmt.Sprintf("response body is not valid JSON: %v", err))
		return
	}

	AssertResponseValid(t, parsed)
}

// AssertResponseValid asserts that a parsed response object conforms to the
// OpenAPI schema identified by each nested object's "object" discriminator.
func AssertResponseValid(t *testing.T, parsed map[string]any) {
	t.Helper()

	if parsed == nil {
		assert.Fail(t, "response object is nil")
		return
	}

	spec, err := LoadFullSpec()
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed to load OpenAPI spec: %v", err))
		return
	}

	index, err := getObjectSchemaIndex(spec)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed to build object schema index: %v", err))
		return
	}

	objectType, ok := parsed["object"].(string)
	if !ok || objectType == "" {
		assert.Fail(t, `top-level response is missing non-empty string "object" field`)
		return
	}
	if objectType == "list" || objectType == "map" {
		validateTopLevelContainer(t, spec, index, parsed, "$")
		return
	}

	schema, ok := pickSchemaForObjectType(index, objectType, parsed)
	if !ok {
		assert.Fail(t, fmt.Sprintf(`top-level object %q not found in OpenAPI schema index`, objectType))
		return
	}

	walkSchemaValue(t, spec, index, parsed, schema, "$")
}

func validateTopLevelContainer(
	t *testing.T,
	spec *openAPISpec,
	index map[string][]*openAPISchema,
	parsed map[string]any,
	path string,
) {
	t.Helper()

	objectType, _ := parsed["object"].(string)
	if objectType == "list" {
		checkRequiredField(t, parsed, "object", &openAPISchema{Type: "string"}, path)
		checkRequiredField(t, parsed, "page_info", &openAPISchema{Type: "object"}, path)
		checkRequiredField(t, parsed, "data", &openAPISchema{Type: "array"}, path)

		if pageInfo := parsed["page_info"]; pageInfo != nil {
			if pageInfoSchema := schemaByName(spec, "PageInfo"); pageInfoSchema != nil {
				walkSchemaValue(t, spec, index, pageInfo, pageInfoSchema, joinPath(path, "page_info"))
			}
		}
		if data, ok := parsed["data"].([]any); ok {
			for i, item := range data {
				obj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				itemObject, ok := obj["object"].(string)
				if !ok || itemObject == "" {
					assert.Fail(t, fmt.Sprintf(`%s.data[%d]: missing non-empty "object" field`, path, i))
					continue
				}
				itemSchema, ok := pickSchemaForObjectType(index, itemObject, obj)
				if !ok {
					assert.Fail(t, fmt.Sprintf(`%s.data[%d].object=%q not found in OpenAPI schema index`, path, i, itemObject))
					continue
				}
				walkSchemaValue(t, spec, index, obj, itemSchema, fmt.Sprintf("%s.data[%d]", path, i))
			}
		}
		return
	}

	if objectType == "map" {
		checkRequiredField(t, parsed, "object", &openAPISchema{Type: "string"}, path)
		for key, raw := range parsed {
			if key == "object" || raw == nil {
				continue
			}
			obj, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			itemObject, ok := obj["object"].(string)
			if !ok || itemObject == "" {
				continue
			}
			itemSchema, ok := pickSchemaForObjectType(index, itemObject, obj)
			if !ok {
				assert.Fail(t, fmt.Sprintf(`%s.%s.object=%q not found in OpenAPI schema index`, path, key, itemObject))
				continue
			}
			walkSchemaValue(t, spec, index, obj, itemSchema, joinPath(path, key))
		}
	}
}

func getObjectSchemaIndex(spec *openAPISpec) (map[string][]*openAPISchema, error) {
	objectSchemaIndexOnce.Do(func() {
		objectSchemaIndex = make(map[string][]*openAPISchema)

		if spec == nil || spec.Components == nil {
			objectSchemaIndexErr = fmt.Errorf("spec components are nil")
			return
		}

		for schemaName, schema := range spec.Components.Schemas {
			objectProp, ok := schema.Properties["object"]
			if !ok || len(objectProp.Enum) != 1 {
				continue
			}
			objectType, ok := objectProp.Enum[0].(string)
			if !ok || objectType == "" {
				continue
			}

			// Ambiguous container discriminators are handled separately.
			if objectType == "list" || objectType == "map" {
				continue
			}

			s := schema
			objectSchemaIndex[objectType] = append(objectSchemaIndex[objectType], &s)
			_ = schemaName
		}
	})

	if objectSchemaIndexErr != nil {
		return nil, objectSchemaIndexErr
	}
	return objectSchemaIndex, nil
}

func walkSchemaValue(
	t *testing.T,
	spec *openAPISpec,
	index map[string][]*openAPISchema,
	value any,
	schema *openAPISchema,
	path string,
) {
	t.Helper()

	if schema == nil {
		return
	}

	resolved := resolveSchema(spec, schema, 0)
	if resolved == nil {
		assert.Fail(t, fmt.Sprintf("%s: could not resolve schema", path))
		return
	}

	if value == nil {
		return
	}

	if resolved.Type == "array" {
		items, ok := value.([]any)
		if !ok {
			assert.Fail(t, fmt.Sprintf("%s: expected array, got %T", path, value))
			return
		}
		for i, item := range items {
			walkSchemaValue(t, spec, index, item, resolved.Items, fmt.Sprintf("%s[%d]", path, i))
		}
		return
	}

	if !isObjectLikeSchema(resolved) {
		return
	}

	obj, ok := value.(map[string]any)
	if !ok {
		// Free-form schemas (type "object" with no properties/required)
		// represent json.RawMessage or similar — accept any JSON type.
		if len(resolved.Properties) == 0 && len(resolved.Required) == 0 {
			return
		}
		assert.Fail(t, fmt.Sprintf("%s: expected object, got %T", path, value))
		return
	}

	for _, reqName := range resolved.Required {
		propSchema := resolved.Properties[reqName]
		checkRequiredField(t, obj, reqName, &propSchema, path)
	}

	// If this value advertises a specific object discriminator, prefer that schema.
	if objectType, ok := obj["object"].(string); ok && objectType != "" && objectType != "list" && objectType != "map" {
		discriminatorSchema, exists := pickSchemaForObjectType(index, objectType, obj)
		if !exists {
			assert.Fail(t, fmt.Sprintf(`%s.object=%q not found in OpenAPI schema index`, path, objectType))
			return
		}
		resolvedDiscriminator := resolveSchema(spec, discriminatorSchema, 0)
		if resolvedDiscriminator != nil {
			resolved = resolvedDiscriminator
		}
	}

	// Recurse schema-defined properties.
	for propName, propSchema := range resolved.Properties {
		val, exists := obj[propName]
		if !exists || val == nil {
			continue
		}
		walkSchemaValue(t, spec, index, val, &propSchema, joinPath(path, propName))
	}

	// Special traversal for list payloads where item schema can vary.
	if objectType, _ := obj["object"].(string); objectType == "list" {
		pageInfoSchema := schemaByName(spec, "PageInfo")
		if pageInfo, ok := obj["page_info"]; ok && pageInfo != nil && pageInfoSchema != nil {
			walkSchemaValue(t, spec, index, pageInfo, pageInfoSchema, joinPath(path, "page_info"))
		}

		if rawData, exists := obj["data"]; exists && rawData != nil {
			data, ok := rawData.([]any)
			if !ok {
				assert.Fail(t, fmt.Sprintf("%s.data: expected array, got %T", path, rawData))
				return
			}
			for i, item := range data {
				itemPath := fmt.Sprintf("%s.data[%d]", path, i)
				child, ok := item.(map[string]any)
				if !ok {
					continue
				}
				childObject, ok := child["object"].(string)
				if !ok || childObject == "" {
					assert.Fail(t, fmt.Sprintf(`%s: missing non-empty "object" field`, itemPath))
					continue
				}
				childSchema, exists := pickSchemaForObjectType(index, childObject, child)
				if !exists {
					assert.Fail(t, fmt.Sprintf(`%s.object=%q not found in OpenAPI schema index`, itemPath, childObject))
					continue
				}
				walkSchemaValue(t, spec, index, child, childSchema, itemPath)
			}
		}
	}

	// Special traversal for map payloads where values are dynamic keys.
	if objectType, _ := obj["object"].(string); objectType == "map" {
		for key, raw := range obj {
			if key == "object" {
				continue
			}
			if raw == nil {
				continue
			}

			switch v := raw.(type) {
			case map[string]any:
				childPath := joinPath(path, key)
				childObject, ok := v["object"].(string)
				if !ok || childObject == "" {
					// Non-discriminated object: still recurse shallowly from current schema if possible.
					continue
				}
				childSchema, exists := pickSchemaForObjectType(index, childObject, v)
				if !exists {
					assert.Fail(t, fmt.Sprintf(`%s.object=%q not found in OpenAPI schema index`, childPath, childObject))
					continue
				}
				walkSchemaValue(t, spec, index, v, childSchema, childPath)
			case []any:
				for i, arrItem := range v {
					child, ok := arrItem.(map[string]any)
					if !ok {
						continue
					}
					childPath := fmt.Sprintf("%s.%s[%d]", path, key, i)
					childObject, ok := child["object"].(string)
					if !ok || childObject == "" {
						assert.Fail(t, fmt.Sprintf(`%s: missing non-empty "object" field`, childPath))
						continue
					}
					childSchema, exists := pickSchemaForObjectType(index, childObject, child)
					if !exists {
						assert.Fail(t, fmt.Sprintf(`%s.object=%q not found in OpenAPI schema index`, childPath, childObject))
						continue
					}
					walkSchemaValue(t, spec, index, child, childSchema, childPath)
				}
			}
		}
	}
}

func pickSchemaForObjectType(index map[string][]*openAPISchema, objectType string, obj map[string]any) (*openAPISchema, bool) {
	candidates := index[objectType]
	if len(candidates) == 0 {
		return nil, false
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}

	bestIdx := 0
	bestScore := -1
	bestMissing := len(obj) + 1

	for i, candidate := range candidates {
		score := 0
		missing := 0
		for _, req := range candidate.Required {
			if _, ok := obj[req]; ok {
				score++
			} else {
				missing++
			}
		}
		if missing < bestMissing || (missing == bestMissing && score > bestScore) {
			bestIdx = i
			bestScore = score
			bestMissing = missing
		}
	}

	return candidates[bestIdx], true
}

func checkRequiredField(t *testing.T, obj map[string]any, field string, schema *openAPISchema, parentPath string) {
	t.Helper()

	fieldPath := joinPath(parentPath, field)
	val, exists := obj[field]
	if !exists {
		assert.Fail(t, fmt.Sprintf("%s: required field is missing", fieldPath))
		return
	}

	s := schema
	if s != nil && s.Nullable && val == nil {
		return
	}

	if val == nil {
		assert.Fail(t, fmt.Sprintf("%s: required field is null", fieldPath))
		return
	}

	if s == nil {
		return
	}

	resolvedType := s.Type
	if resolvedType == "string" {
		str, ok := val.(string)
		if !ok {
			assert.Fail(t, fmt.Sprintf("%s: expected string, got %T", fieldPath, val))
			return
		}
		if str == "" {
			assert.Fail(t, fmt.Sprintf("%s: required string field is empty", fieldPath))
			return
		}
		if s.Format == "date-time" && isZeroTimeString(str) {
			assert.Fail(t, fmt.Sprintf("%s: required date-time field has zero timestamp %q", fieldPath, str))
		}
	}
}

func resolveSchema(spec *openAPISpec, schema *openAPISchema, depth int) *openAPISchema {
	if schema == nil {
		return nil
	}
	if depth > 25 {
		return schema
	}

	out := *schema

	if out.Ref != "" {
		resolved, ok := spec.ResolveSchemaRef(out.Ref)
		if !ok {
			return &out
		}
		return resolveSchema(spec, resolved, depth+1)
	}

	if len(out.AllOf) > 0 {
		merged := openAPISchema{
			Type:                 out.Type,
			Properties:           map[string]openAPISchema{},
			Required:             append([]string{}, out.Required...),
			Nullable:             out.Nullable,
			AdditionalProperties: out.AdditionalProperties,
		}

		for _, sub := range out.AllOf {
			r := resolveSchema(spec, &sub, depth+1)
			if r == nil {
				continue
			}
			if merged.Type == "" {
				merged.Type = r.Type
			}
			for k, v := range r.Properties {
				merged.Properties[k] = v
			}
			merged.Required = append(merged.Required, r.Required...)
			if r.AdditionalProperties != nil && merged.AdditionalProperties == nil {
				merged.AdditionalProperties = r.AdditionalProperties
			}
		}

		for k, v := range out.Properties {
			merged.Properties[k] = v
		}
		merged.Required = uniqueStrings(merged.Required)

		if len(merged.AllOf) == 0 {
			merged.AllOf = nil
		}
		return &merged
	}

	return &out
}

func schemaByName(spec *openAPISpec, name string) *openAPISchema {
	if spec == nil || spec.Components == nil {
		return nil
	}
	s, ok := spec.Components.Schemas[name]
	if !ok {
		return nil
	}
	schema := s
	return &schema
}

func isObjectLikeSchema(schema *openAPISchema) bool {
	if schema == nil {
		return false
	}
	if schema.Type == "object" {
		return true
	}
	return len(schema.Properties) > 0 || len(schema.AllOf) > 0
}

func joinPath(parent, field string) string {
	if parent == "" || parent == "$" {
		return "$." + field
	}
	return parent + "." + field
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func isZeroTimeString(v string) bool {
	return strings.HasPrefix(v, "0001-01-01T00:00:00")
}
